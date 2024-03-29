package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type store struct {
	custom         map[string]Metric
	memstat        *runtime.MemStats
	key            []byte
	mu             sync.RWMutex
	logger         *zap.Logger
	runtimeMetrics map[string]MetricType
}
type Sender interface {
	Do(req *http.Request) (*http.Response, error)
	// SendMetric(context.Context, Metrics) error
	SendMetrics(context.Context, []Metrics) error
	Type() string
}

var defaultRuntimeMetrics = map[string]MetricType{
	"Alloc":         GaugeType,
	"BuckHashSys":   GaugeType,
	"Frees":         GaugeType,
	"GCCPUFraction": GaugeType,
	"GCSys":         GaugeType,
	"HeapAlloc":     GaugeType,
	"HeapIdle":      GaugeType,
	"HeapInuse":     GaugeType,
	"HeapObjects":   GaugeType,
	"HeapReleased":  GaugeType,
	"HeapSys":       GaugeType,
	"LastGC":        GaugeType,
	"Lookups":       GaugeType,
	"MCacheInuse":   GaugeType,
	"MCacheSys":     GaugeType,
	"MSpanInuse":    GaugeType,
	"MSpanSys":      GaugeType,
	"Mallocs":       GaugeType,
	"NextGC":        GaugeType,
	"NumForcedGC":   GaugeType,
	"NumGC":         GaugeType,
	"OtherSys":      GaugeType,
	"PauseTotalNs":  GaugeType,
	"StackInuse":    GaugeType,
	"StackSys":      GaugeType,
	"Sys":           GaugeType,
	"TotalAlloc":    GaugeType,
}

// NewStore create in memory metrics store
func NewStore(key []byte, logger *zap.Logger) *store {
	runtimeMetrics := make(map[string]MetricType)
	for k, v := range defaultRuntimeMetrics {
		runtimeMetrics[k] = v
	}
	return &store{
		memstat:        &runtime.MemStats{},
		custom:         make(map[string]Metric),
		key:            key,
		logger:         logger,
		runtimeMetrics: runtimeMetrics,
	}
}

// MemStats returns memstat metrics in URL view
func (s *store) MemStats() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]string, 0)
	r := reflect.ValueOf(s.memstat)
	for k, v := range s.runtimeMetrics {
		f := r.Elem().FieldByName(k)
		if emulateError || !f.IsValid() {
			s.logger.Error("Bad Name - " + k)
			return nil
		}
		res = append(res, fmt.Sprintf("/update/%s/%s/%v", v, k, f))
	}
	sort.Strings(res)
	return res
}

// All returns all metrics
func (s *store) All() []string {
	res := make([]string, 0)
	for _, v := range s.Custom() {
		res = append(res, fmt.Sprintf("/update/%s/%s/%s", v.Type(), v.Name(), v))
	}
	res = append(res, s.MemStats()...)
	sort.Strings(res)
	return res
}

// AllMetrics returns in Metrics view
func (s *store) AllMetrics() []Metrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make([]Metrics, 0)
	keys := make([]string, 0)
	for k := range s.custom {
		keys = append(keys, k)
	}
	for k := range s.runtimeMetrics {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	rM := reflect.ValueOf(s.memstat)

	for _, k := range keys {
		if _, ok := s.custom[k]; ok {
			m := s.custom[k].Metrics()
			if len(s.key) != 0 {
				if err := m.Sign(s.key); err != nil {
					return nil
				}
			}
			res = append(res, m)
			continue
		}
		f := rM.Elem().FieldByName(k)
		if !f.IsValid() {
			fmt.Println("Bad Name - ", k)
			return nil
		}
		m := Metrics{ID: k, MType: s.runtimeMetrics[k]}
		switch s.runtimeMetrics[k] {
		// case string(CounterType):
		// 	m.Delta = GetInt64Pointer(f.Int())
		case GaugeType:
			var a float64
			switch f.Type().String() {
			case "uint64", "uint32":
				a = float64(f.Uint())
			default:
				a = f.Float()
			}
			m.Value = &a
		}
		if len(s.key) != 0 {
			if err := m.Sign(s.key); err != nil {
				s.logger.Error(err.Error())
				return nil
			}
		}
		res = append(res, m)
	}

	return res
}

// Custom returns custom metrics
func (s *store) Custom() map[string]Metric {
	s.mu.Lock()
	result := make(map[string]Metric, len(s.custom))
	for k, v := range s.custom {
		result[k] = v
	}
	s.mu.Unlock()
	return result
}

// Save send metrics to store server
func (s *store) Save(ctx context.Context, wg *sync.WaitGroup, client Sender, baseURL string, isJSON bool, batch bool) error {
	defer wg.Done()
	if client == nil && len(baseURL) == 0 {
		return nil
	}

	switch client.Type() {
	case "http":
		if !strings.Contains(baseURL, "http://") {
			baseURL = fmt.Sprintf("http://%s", baseURL)
		}
		if !isJSON {
			res := s.All()
			errC := make(chan error, len(res))
			for i := 0; i < len(res); i++ {
				go func(ctx context.Context, c Sender, url string) {
					req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
					if err != nil {
						errC <- err
						return
					}
					req.Header.Set("Content-Type", "text/plain")
					resp, err := c.Do(req)
					if err != nil {
						errC <- err
						s.logger.Error(err.Error())
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						var body []byte
						body, err = io.ReadAll(resp.Body)
						if err != nil || emulateError {
							if err == nil {
								err = errors.New("emulateError")
							}
							errC <- err
							return
						}
						errC <- fmt.Errorf("save failed: %v", string(body))
						return
					}
					_, err = io.Copy(io.Discard, resp.Body)
					if err != nil || emulateError {
						if err == nil {
							err = errors.New("emulateError")
						}
						errC <- err
						return
					}
					errC <- nil
				}(ctx, client, baseURL+res[i])
			}
			for i := 0; i < len(res); i++ {
				if err := <-errC; err != nil {
					return err
				}
			}
			return nil
		}
		res := s.AllMetrics()
		if batch {
			return sendMetrics(ctx, client, baseURL+"/updates/", res)
		}
		errC := make(chan error, len(res))
		for i := 0; i < len(res); i++ {
			go sendMetric(ctx, errC, client, baseURL+"/update/", res[i])
		}
		for i := 0; i < len(res); i++ {
			if err := <-errC; err != nil {
				return err
			}
		}
	case "grpc":
		res := s.AllMetrics()
		return client.SendMetrics(ctx, res)
	default:
		return fmt.Errorf("транспорт не поддерживается %s", client.Type())
	}
	return nil
}

func sendMetric(ctx context.Context, errC chan error, c Sender, url string, metric Metrics) {
	b, err := json.Marshal(metric)
	if err != nil || len(fmt.Sprint(metric)) == 0 {
		if len(fmt.Sprint(metric)) == 0 {
			err = ErrNoSuchMetricType
		}
		errC <- err
		return
	}
	buf := bytes.NewBuffer(b)
	req, err := http.NewRequestWithContext(ctx, "POST", url, buf)
	if err != nil {
		errC <- err
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		errC <- err
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var body []byte
		body, err = io.ReadAll(resp.Body)
		if err != nil || emulateError {
			if err == nil {
				err = errors.New("emulateError")
			}
			errC <- err
			return
		}
		errC <- errors.New("save failed: " + string(body))
		return
	}
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil || emulateError {
		if err == nil {
			err = errors.New("emulateError")
		}
		errC <- err
		return
	}
	errC <- nil
}
func sendMetrics(ctx context.Context, c Sender, url string, metrics []Metrics) error {
	b, err := json.Marshal(metrics)
	if metrics == nil || err != nil {
		if err == nil {
			err = errors.New("metrics is nil")
		}
		return err
	}
	buf := bytes.NewBuffer(b)
	req, err := http.NewRequestWithContext(ctx, "POST", url, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	var resp *http.Response
	resp, err = c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body []byte
		body, err = io.ReadAll(resp.Body)
		if err != nil || emulateError {
			if err == nil {
				err = errors.New("emulateError")
			}
			return err
		}
		return fmt.Errorf("save failed: %s. %v", string(body), err)
	}
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil || emulateError {
		if err == nil {
			err = errors.New("emulateError")
		}
		return err
	}
	return nil
}

// AddCustom add custom metrics to store
func (s *store) AddCustom(m ...Metric) {
	s.mu.Lock()
	for _, v := range m {
		fmt.Println("Metric ", v.Name(), " was added")
		s.custom[v.Name()] = v
	}
	s.mu.Unlock()
}

// Scrape perform collect metrics
func (s *store) Scrape() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	runtime.ReadMemStats(s.memstat)

	var errC = make(chan error, len(s.custom))
	for k := range s.custom {
		go func(n string) {
			m := s.custom[n]
			err := m.Scrape()
			if err != nil {
				errC <- err
				return
			}
			errC <- nil
		}(k)
	}
	for i := 0; i < len(s.custom); i++ {
		if err := <-errC; err != nil {
			return err
		}
	}
	return nil
}
