package postgres

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/gopherlearning/track-devops/cmd/server/storage/postgres/migrations"
	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/migrate"

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
	var hash string
	var mdelta int64
	var mvalue float64
	err := s.GetConn(context.Background()).QueryRow(context.Background(), `select hash,COALESCE(mdelta, 0),COALESCE( mvalue, 0 ) from metrics where target = $1 AND id = $2 AND mtype = $3`, target, name, mType).Scan(&hash, &mdelta, &mvalue)
	if err != nil {
		s.loger.Error(err)
		return nil, err
	}
	switch mType {
	case string(metrics.CounterType):
		return &metrics.Metrics{
			ID:    name,
			MType: mType,
			Delta: &mdelta,
			Hash:  hash,
		}, nil
	case string(metrics.GaugeType):
		return &metrics.Metrics{
			ID:    name,
			MType: mType,
			Value: &mvalue,
			Hash:  hash,
		}, nil
	default:
		return nil, metrics.ErrNoSuchMetricType
	}

}

// Update(target, metric, name, value string) error
func (s *Storage) UpdateMetric(ctx context.Context, target string, mm ...metrics.Metrics) (err error) {
	old, err := s.Metrics(target)
	if err != nil && err != pgx.ErrNoRows {
		s.loger.Error(err)
		return err
	}
	oldMap := make(map[string]metrics.Metrics, len(old[target]))
	for _, v := range old[target] {
		oldMap[v.ID] = v
	}
	forAdd := make(map[string]metrics.Metrics, 0)
	forUpdate := make(map[string]metrics.Metrics, 0)
	for _, n := range mm {
		o, ok := forAdd[n.ID]
		if ok {
			if o.MType == string(metrics.CounterType) {
				m := *o.Delta + *n.Delta
				n.Delta = &m
			}
			forAdd[n.ID] = n
			continue
		}
		o, ok = oldMap[n.ID]
		if !ok {
			forAdd[n.ID] = n
			continue
		}
		if n.MType == string(metrics.CounterType) {
			m := *o.Delta + *n.Delta
			n.Delta = &m
		}
		forUpdate[n.ID] = n
	}

	tx, err := s.GetConn(ctx).Begin(ctx)
	if err != nil {
		s.loger.Error(err)
		return
	}
	defer func() {
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil {
				s.loger.Error(err)
			}
		}
	}()
	for _, n := range forAdd {
		_, err = tx.Exec(context.Background(), `INSERT INTO metrics (target,id, hash, mtype, mdelta, mvalue) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`, target, n.ID, n.Hash, n.MType, n.Delta, n.Value)
		if err != nil {

			s.loger.Error(err)
			return
		}
	}
	for _, n := range forUpdate {
		_, err = tx.Exec(context.Background(), `UPDATE metrics SET mdelta = $1, mvalue = $2 WHERE id = $3 AND target = $4`, n.Delta, n.Value, n.ID, target)
		if err != nil {
			s.loger.Error(err)
			return
		}
	}
	err = tx.Commit(ctx)
	return
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
