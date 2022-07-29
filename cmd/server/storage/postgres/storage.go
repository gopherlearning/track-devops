package postgres

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gopherlearning/track-devops/cmd/server/storage/postgres/migrations"
	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/migrate"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	mu                 sync.Mutex
	db                 *pgxpool.Pool
	connConfig         *pgxpool.Config
	loger              logrus.FieldLogger
	maxConnectAttempts int
}

func NewStorage(dsn string, loger logrus.FieldLogger) (*Storage, error) {
	connConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	s := &Storage{connConfig: connConfig, loger: loger, maxConnectAttempts: 10}
	err = migrate.MigrateFromFS(context.Background(), s.GetConn(context.Background()), &migrations.Migrations, loger)
	if err != nil {
		loger.Error(err)
		return nil, err
	}
	return s, nil
}
func (s *Storage) Close(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	s.db.Close()
	return nil
}

func (s *Storage) reconnect(ctx context.Context) (*pgxpool.Pool, error) {

	pool, err := pgxpool.ConnectConfig(context.Background(), s.connConfig)

	if err != nil {
		return nil, fmt.Errorf("unable to connection to database: %v", err)
	}
	if err = pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("couldn't ping postgre database: %v", err)
	}
	return pool, err
}

func (s *Storage) GetConn(ctx context.Context) *pgxpool.Pool {
	var err error

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil || s.db.Ping(ctx) != nil {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.loger.Info("reconnecting...")
			s.db, err = s.reconnect(ctx)
			if err != nil {
				continue
			}
			return s.db
		}
		return nil
	}
	return s.db
}

// Get(target, metric, name string) string
func (s *Storage) GetMetric(target string, mType string, name string) (*metrics.Metrics, error) {

	var id string
	var hash string
	var mtype string
	var mdelta int64
	var mvalue float64
	err := s.GetConn(context.Background()).QueryRow(context.Background(), `select id,hash,mtype,COALESCE(mdelta, 0),COALESCE( mvalue, 0 )  from metrics where target = $1`, target).Scan(&id, &hash, &mtype, &mdelta, &mvalue)
	if err != nil {
		s.loger.Error(err)
		return nil, err
	}
	switch mtype {
	case string(metrics.CounterType):
		return &metrics.Metrics{
			ID:    id,
			MType: mtype,
			Delta: &mdelta,
			Hash:  hash,
		}, nil
	case string(metrics.GaugeType):
		return &metrics.Metrics{
			ID:    id,
			MType: mtype,
			Value: &mvalue,
			Hash:  hash,
		}, nil
	default:
		return nil, metrics.ErrNoSuchMetricType
	}
}

// Update(target, metric, name, value string) error
func (s *Storage) UpdateMetric(ctx context.Context, target string, mm ...metrics.Metrics) error {
	mmOld, err := s.Metrics(target)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	for j, m := range mmOld[target] {
		for i := range mm {
			if mm[i].ID != m.ID || mm[i].MType != m.MType {
				if i == len(mm)-1 {
					mmOld[target] = append(mmOld[target], mm[i])
				}
				continue
			}
			res := mm[i]
			if m.MType == string(metrics.CounterType) {
				m := *res.Delta + *m.Delta
				res.Delta = &m
			}
			mmOld[target][j] = res
		}
	}
	tx, err := s.GetConn(ctx).Begin(ctx)
	if err != nil {
		return err
	}
	for i := range mm {
		_, err = tx.Exec(context.Background(), `INSERT INTO metrics (target,id, hash, mtype, mdelta, mvalue) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (target,id) DO UPDATE SET mdelta = $5, mvalue = $6`, target, mm[i].ID, mm[i].Hash, mm[i].MType, mm[i].Delta, mm[i].Value)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
	// var data = make([]metrics.Metrics, 0)
	// err := s.GetConn(context.Background()).QueryRow(context.Background(), `select data::jsonb from metrics where target = $1`, target).Scan(&data)
	// if err != nil {
	// 	if !errors.Is(err, pgx.ErrNoRows) {
	// 		return err
	// 	}
	// 	_, err = s.GetConn(context.Background()).Exec(context.Background(), `INSERT INTO metrics (target, data) VALUES($1, $2)`, target, data)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// if len(data) == 0 {
	// 	data = mm
	// } else {
	// 	for _, m := range mm {
	// 		for i := range data {
	// 			if data[i].ID != m.ID || data[i].MType != m.MType {
	// 				if i == len(data)-1 {
	// 					data = append(data, m)
	// 				}
	// 				continue
	// 			}
	// 			res := data[i]
	// 			switch m.MType {
	// 			case string(metrics.CounterType):
	// 				m := *res.Delta + *m.Delta
	// 				res.Delta = &m
	// 			case string(metrics.GaugeType):
	// 				res.Value = m.Value
	// 			}
	// 			data[i] = res
	// 			break
	// 		}
	// 	}
	// }
	// _, err = s.GetConn(context.Background()).Exec(context.Background(), `UPDATE metrics SET data = $1 WHERE target = $2`, data, target)
	// if err != nil {
	// 	return err
	// }
	// return nil
}

func (s *Storage) Metrics(target string) (map[string][]metrics.Metrics, error) {
	res := make(map[string][]metrics.Metrics)
	SQL := `select target,id,hash,mtype,COALESCE(mdelta, 0),COALESCE( mvalue, 0 ) from metrics`
	if len(target) != 0 {
		SQL = fmt.Sprintf(`%s where target = '%s'`, SQL, target)
	}
	rows, err := s.GetConn(context.Background()).Query(context.Background(), SQL)
	if err != nil {
		err = fmt.Errorf("queryRow failed: %v", err)
		s.loger.Error(err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var target string
		var id string
		var hash string
		var mtype string
		var mdelta int64
		var mvalue float64
		err := rows.Scan(&target, &id, &hash, &mtype, &mdelta, &mvalue)
		if err != nil {
			s.loger.Error(err)
			return nil, err
		}
		if _, ok := res[target]; !ok {
			res[target] = make([]metrics.Metrics, 0)
		}
		switch mtype {
		case string(metrics.CounterType):
			res[target] = append(res[target], metrics.Metrics{
				ID:    id,
				MType: mtype,
				Delta: &mdelta,
				Hash:  hash,
			})
		case string(metrics.GaugeType):
			res[target] = append(res[target], metrics.Metrics{
				ID:    id,
				MType: mtype,
				Value: &mvalue,
				Hash:  hash,
			})

		}
	}
	if rows.Err() != nil {
		s.loger.Error(err)
		return nil, err
	}
	return res, nil
}

func (s *Storage) List() (map[string][]string, error) {
	mm, err := s.Metrics("")
	if err != nil {
		s.loger.Error(err)
		return nil, err
	}
	res := make(map[string][]string, len(mm))
	for k := range mm {
		res[k] = make([]string, len(mm[k]))
		for i := range mm[k] {
			res[k][i] = mm[k][i].StringFull()
		}
	}
	for k, v := range res {
		sort.Strings(v)
		res[k] = v
	}
	return res, nil
}

func (s *Storage) ListProm(targets ...string) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Storage) Ping(ctx context.Context) error {
	ctx_, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ping := make(chan error)
	go func() {
		ping <- s.GetConn(ctx_).Ping(ctx_)
	}()
	select {
	case err := <-ping:
		return err
	case <-ctx_.Done():
		return fmt.Errorf("context closed")
	}

}
