package local

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
)

var _ repositories.Repository = (*Storage)(nil)

// Storage inmemory storage
type Storage struct {
	metrics   map[string][]metrics.Metrics
	storeFile string
	mu        sync.RWMutex
	logger    *zap.Logger
	PingError bool
}

// NewStorage inmemory storage
func NewStorage(restore bool, storeInterval *time.Duration, logger *zap.Logger, storeFile ...string) (*Storage, error) {
	s := &Storage{
		metrics: make(map[string][]metrics.Metrics),
		logger:  logger,
	}
	if len(storeFile) != 0 {
		s.storeFile = storeFile[0]
	}
	if restore && storeFile != nil {
		if _, err := os.Stat(s.storeFile); err == nil {
			data, err := os.ReadFile(s.storeFile)
			if err != nil {
				return nil, err
			}
			err = json.Unmarshal(data, &s.metrics)
			if err != nil {
				return nil, err
			}
		}
	}
	if storeInterval != nil {
		ticker := time.NewTicker(*storeInterval)
		go func() {
			for range ticker.C {
				err := s.Save()
				if err != nil {
					logger.Error(err.Error())
					return
				}
			}
		}()
	}
	return s, nil
}

// Save perform dump of storage to disk
func (s *Storage) Save() error {
	m, _ := s.Metrics(context.Background(), "")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if len(s.storeFile) == 0 {
		s.logger.Info(fmt.Sprintf("Эмуляция сохранения:\n%s", string(data)))
		return nil
	}
	err = os.WriteFile(s.storeFile, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// GetMetric ...
func (s *Storage) GetMetric(ctx context.Context, target string, mtype metrics.MetricType, name string) (*metrics.Metrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.metrics[target]; ok {
		for i := range s.metrics[target] {
			if s.metrics[target][i].MType == mtype && s.metrics[target][i].ID == name {
				res := s.metrics[target][i]
				return &res, nil
			}
		}
	}
	return nil, nil
}

// Ping заглушка
func (s *Storage) Ping(context.Context) error {
	if s.PingError {
		return fmt.Errorf("emulate error for test")
	}
	return nil
}

// UpdateMetric ...
func (s *Storage) UpdateMetric(ctx context.Context, target string, mm ...metrics.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range mm {
		switch {
		case len(target) == 0:
			return repositories.ErrWrongTarget
		case len(m.ID) == 0:
			return repositories.ErrWrongMetricID
		case len(m.MType) == 0 || (m.MType != metrics.CounterType && m.MType != metrics.GaugeType):
			return repositories.ErrWrongMetricType
		case m.Delta == nil && m.Value == nil:
			return repositories.ErrWrongMetricValue
		}
		if _, ok := s.metrics[target]; !ok {
			s.metrics[target] = make([]metrics.Metrics, 0)
		}
		for i := range s.metrics[target] {
			if s.metrics[target][i].MType == m.MType && s.metrics[target][i].ID == m.ID {
				res := s.metrics[target][i]
				switch m.MType {
				case metrics.CounterType:
					m := *res.Delta + *m.Delta
					res.Delta = &m
				case metrics.GaugeType:
					res.Value = m.Value
				}
				s.metrics[target][i] = res
				return nil
			}
		}
		s.metrics[target] = append(s.metrics[target], m)
	}
	return nil
}

// Metrics returns metrics view of stored metrics
func (s *Storage) Metrics(ctx context.Context, target string) (map[string][]metrics.Metrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string][]metrics.Metrics)
	for target := range s.metrics {
		for k := range s.metrics[target] {
			if _, ok := res[target]; !ok {
				res[target] = make([]metrics.Metrics, 0)
			}
			res[target] = append(res[target], s.metrics[target][k])
		}
	}
	return res, nil
}

// List all metrics for all targets
func (s *Storage) List(ctx context.Context) (map[string][]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string][]string)
	for target := range s.metrics {
		for m := range s.metrics[target] {
			if _, ok := res[target]; !ok {
				res[target] = make([]string, 0)
			}
			res[target] = append(res[target], fmt.Sprint(s.metrics[target][m].StringFull()))
		}
		for k, v := range res {
			sort.Strings(v)
			res[k] = v
		}
	}
	return res, nil
}
