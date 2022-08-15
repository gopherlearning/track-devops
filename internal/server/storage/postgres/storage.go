package postgres

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/migrate"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/gopherlearning/track-devops/internal/server/storage/postgres/migrations"
)

var _ repositories.Repository = (*Storage)(nil)

// Storage postgres storage
type Storage struct {
	db                 *pgxpool.Pool
	connConfig         *pgxpool.Config
	loger              logrus.FieldLogger
	maxConnectAttempts int
}

// NewStorage reterns new  postgres storage
func NewStorage(dsn string, loger logrus.FieldLogger) (*Storage, error) {
	connConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	connConfig.HealthCheckPeriod = 2 * time.Second
	s := &Storage{connConfig: connConfig, loger: loger, maxConnectAttempts: 10}
	err = migrate.MigrateFromFS(context.Background(), s.db, &migrations.Migrations, loger)
	if err != nil {
		loger.Error(err)
		return nil, err
	}
	return s, nil
}

// Close database connection
func (s *Storage) Close(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	s.db.Close()
	return nil
}

// GetMetric ...
func (s *Storage) GetMetric(ctx context.Context, target string, mType string, name string) (*metrics.Metrics, error) {
	var hash string
	var mdelta int64
	var mvalue float64
	err := s.db.QueryRow(ctx, `select hash,COALESCE(mdelta, 0),COALESCE( mvalue, 0 ) from metrics where target = $1 AND id = $2 AND mtype = $3`, target, name, mType).Scan(&hash, &mdelta, &mvalue)
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

// UpdateMetric ...
func (s *Storage) UpdateMetric(ctx context.Context, target string, mm ...metrics.Metrics) (err error) {
	old, err := s.Metrics(ctx, target)
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

	tx, err := s.db.Begin(ctx)
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

	stmtInsert, err := tx.Prepare(ctx, "insert", `INSERT INTO metrics (target,id, hash, mtype, mdelta, mvalue) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`)
	if err != nil {
		return err
	}
	for _, n := range forAdd {
		_, err = tx.Exec(ctx, stmtInsert.Name, target, n.ID, n.Hash, n.MType, n.Delta, n.Value)
		if err != nil {

			s.loger.Error(err)
			return
		}
	}

	// шаг 2 — готовим инструкцию
	stmtUpdate, err := tx.Prepare(ctx, "update", "UPDATE metrics SET mdelta = $1, mvalue = $2 WHERE id = $3 AND target = $4")
	if err != nil {
		return err
	}
	for _, n := range forUpdate {
		_, err = tx.Exec(ctx, stmtUpdate.Name, n.Delta, n.Value, n.ID, target)
		if err != nil {
			s.loger.Error(err)
			return
		}
	}
	err = tx.Commit(ctx)
	return
}

// Metrics returns metrics view of stored metrics
func (s *Storage) Metrics(ctx context.Context, target string) (map[string][]metrics.Metrics, error) {
	res := make(map[string][]metrics.Metrics)
	SQL := `select target,id,hash,mtype,COALESCE(mdelta, 0),COALESCE( mvalue, 0 ) from metrics`
	if len(target) != 0 {
		SQL = fmt.Sprintf(`%s where target = '%s'`, SQL, target)
	}
	rows, err := s.db.Query(ctx, SQL)
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

// List all metrics for all targets
func (s *Storage) List(ctx context.Context) (map[string][]string, error) {
	mm, err := s.Metrics(ctx, "")
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
