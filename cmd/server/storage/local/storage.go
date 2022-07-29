package local

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	mu        sync.RWMutex
	storeFile string
	metrics   map[string][]metrics.Metrics
}

// NewStorage
func NewStorage(restore bool, storeInterval *time.Duration, storeFile ...string) (*Storage, error) {
	s := &Storage{
		metrics: make(map[string][]metrics.Metrics),
	}
	if len(storeFile) != 0 {
		s.storeFile = storeFile[0]
	}
	if restore && storeFile != nil {
		if _, err := os.Stat(s.storeFile); err == nil {
			data, err := ioutil.ReadFile(s.storeFile)
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
					logrus.Error(err)
					return
				}
			}
		}()
	}
	return s, nil
}

var _ repositories.Repository = new(Storage)

func (s *Storage) Ping(context.Context) error {
	return nil
}

func (s *Storage) Save() error {
	m, _ := s.Metrics("")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if len(s.storeFile) == 0 {
		logrus.Infof("Эмуляция сохранения:\n%s", string(data))
		return nil
	}
	err = ioutil.WriteFile(s.storeFile, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
func (s *Storage) Metrics(target string) (map[string][]metrics.Metrics, error) {
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
func (s *Storage) GetMetric(target, mtype, name string) (*metrics.Metrics, error) {
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

func (s *Storage) UpdateMetric(ctx context.Context, target string, mm ...metrics.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range mm {
		switch {
		case len(target) == 0:
			return repositories.ErrWrongTarget
		case len(m.ID) == 0:
			return repositories.ErrWrongMetricID
		case len(m.MType) == 0 || (m.MType != string(metrics.CounterType) && m.MType != string(metrics.GaugeType)):
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
				case string(metrics.CounterType):
					m := *res.Delta + *m.Delta
					res.Delta = &m
				case string(metrics.GaugeType):
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

func (s *Storage) List() (map[string][]string, error) {
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

func (s *Storage) ListProm(targets ...string) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}
