package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
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
		attempt := 0
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if attempt >= s.maxConnectAttempts {
				s.loger.Errorf("connection failed after %d attempt\n", attempt)
			}
			attempt++

			s.loger.Info("reconnecting...")
			s.loger.Info(1)

			s.db, err = s.reconnect(ctx)
			if err == nil {
				return s.db
			}

			s.loger.Errorf("connection was lost. Error: %s. Waiting for 5 sec...\n", err)
		}
		return nil
	}
	return s.db
}

// Get(target, metric, name string) string
func (s *Storage) GetMetric(target string, mType string, name string) *metrics.Metrics {
	var data []metrics.Metrics
	err := s.GetConn(context.Background()).QueryRow(context.Background(), `select data::jsonb from metrics where target = $1`, target).Scan(&data)
	if err != nil {
		s.loger.Error(err)
		return nil
	}
	var res *metrics.Metrics
	for i := range data {
		if data[i].ID != name || data[i].MType != mType {
			continue
		}
		res = &data[i]
	}
	if res == nil {
		s.loger.Error("no such metric ", mType, " - ", name)
	}
	return res
}

// Update(target, metric, name, value string) error
func (s *Storage) UpdateMetric(target string, mm ...metrics.Metrics) error {
	var data = make([]metrics.Metrics, 0)
	err := s.GetConn(context.Background()).QueryRow(context.Background(), `select data::jsonb from metrics where target = $1`, target).Scan(&data)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		_, err = s.GetConn(context.Background()).Exec(context.Background(), `INSERT INTO metrics (target, data) VALUES($1, $2)`, target, data)
		if err != nil {
			return err
		}
	}
	if len(data) == 0 {
		data = mm
	} else {
		for _, m := range mm {
			for i := range data {
				if data[i].ID != m.ID || data[i].MType != m.MType {
					if i == len(data)-1 {
						data = append(data, m)
					}
					continue
				}
				res := data[i]
				switch m.MType {
				case string(metrics.CounterType):
					m := *res.Delta + *m.Delta
					res.Delta = &m
				case string(metrics.GaugeType):
					res.Value = m.Value
				}
				data[i] = res
				break
			}
		}
	}
	_, err = s.GetConn(context.Background()).Exec(context.Background(), `UPDATE metrics SET data = $1 WHERE target = $2`, data, target)
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) Metrics() map[string][]metrics.Metrics {
	res := make(map[string][]metrics.Metrics)
	rows, err := s.GetConn(context.Background()).Query(context.Background(), `select target, data::jsonb from metrics`)
	if err != nil {
		s.loger.Errorf("queryRow failed: %v", err)
		os.Exit(1)
	}
	defer rows.Close()
	for rows.Next() {
		var target string
		var data []metrics.Metrics
		err := rows.Scan(&target, &data)
		if err != nil {
			s.loger.Error(err)
			return nil
		}
		res[target] = data
	}
	if rows.Err() != nil {
		s.loger.Error(err)
		return nil
	}
	return res
}

func (s *Storage) List(targets ...string) map[string][]string {

	res := make(map[string][]string)
	rows, err := s.GetConn(context.Background()).Query(context.Background(), `select target, data::jsonb from metrics`)
	if err != nil {
		s.loger.Errorf("QueryRow failed: %v", err)
		os.Exit(1)
	}
	defer rows.Close()
	for rows.Next() {
		var target string
		var data []metrics.Metrics
		err := rows.Scan(&target, &data)
		if err != nil {
			s.loger.Error(err)
			return nil
		}
		for m := range data {
			if _, ok := res[target]; !ok {
				res[target] = make([]string, 0)
			}
			res[target] = append(res[target], fmt.Sprint(data[m].StringFull()))
		}
		for k, v := range res {
			sort.Strings(v)
			res[k] = v
		}
	}
	if rows.Err() != nil {
		s.loger.Error(err)
		return nil
	}
	return res
}

func (s *Storage) ListProm(targets ...string) []byte {
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
