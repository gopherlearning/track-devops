package storage

import (
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
)

type Storage struct {
	// map[type]map[metric_name]map[target]value
	mu sync.RWMutex
	v  map[metrics.MetricType]map[string]map[string]interface{}
}

// NewStorage
func NewStorage() *Storage {
	return &Storage{
		v: map[metrics.MetricType]map[string]map[string]interface{}{
			metrics.CounterType: make(map[string]map[string]interface{}),
			metrics.GaugeType:   make(map[string]map[string]interface{}),
		},
	}
}

var rMetricURL = regexp.MustCompile(`^.*\/(gauge|counter)\/(\w+)\/(-?\S+)$`)

// var rMetricURL = regexp.MustCompile(`^.*\/(\w+)\/(\w+)\/(-?[0-9\.]+)$`)
var _ repositories.Repository = new(Storage)

func (s *Storage) Update(target, metric, name, value string) error {
	switch {
	case len(target) == 0:
		fmt.Println(target)
		return repositories.ErrWrongTarget
	case len(metric) == 0:
		return repositories.ErrWrongMetricType
	case len(name) == 0:
		return repositories.ErrWrongMetricType
	case len(value) == 0:
		return repositories.ErrWrongMetricValue
	}
	if len(s.v) == 0 {
		return repositories.ErrWrongValueInStorage
	}

	metricType := metrics.MetricType(metric)

	if _, ok := s.v[metricType]; !ok {
		return repositories.ErrWrongMetricType
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.v[metricType][name]; !ok {
		s.v[metricType][name] = make(map[string]interface{})
	}
	if _, ok := s.v[metricType][name][target]; !ok {
		s.v[metricType][name][target] = nil
	}
	switch metricType {
	case metrics.CounterType:
		m, err := strconv.Atoi(value)
		if err != nil {
			return repositories.ErrWrongMetricValue
		}
		if s.v[metricType][name][target] == nil {
			s.v[metricType][name][target] = make([]int, 0)
		}
		mm, ok := s.v[metricType][name][target].([]int)
		if !ok {
			return repositories.ErrWrongValueInStorage
		}
		s.v[metricType][name][target] = append(mm, m)
	case metrics.GaugeType:
		m, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return repositories.ErrWrongMetricValue
		}
		s.v[metricType][name][target] = m
	}
	return nil
}

func (s *Storage) List(targets ...string) map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string][]string)
	for mType := range s.v {

		for mName := range s.v[mType] {
			for target, value := range s.v[mType][mName] {
				if _, ok := res[target]; !ok {
					res[target] = make([]string, 0)
				}
				res[target] = append(res[target], fmt.Sprintf(`%s - %s - %v`, mType, mName, value))
			}
		}
	}
	return res
}

func (s *Storage) ListProm(targets ...string) []byte {
	panic("not implemented") // TODO: Implement
}
