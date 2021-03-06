package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"runtime"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
)

type Store struct {
	mu      sync.RWMutex
	memstat *runtime.MemStats
	custom  map[string]Metric
	key     []byte
}

var runtimeMetrics = map[string]string{
	"Alloc":         "gauge",
	"BuckHashSys":   "gauge",
	"Frees":         "gauge",
	"GCCPUFraction": "gauge",
	"GCSys":         "gauge",
	"HeapAlloc":     "gauge",
	"HeapIdle":      "gauge",
	"HeapInuse":     "gauge",
	"HeapObjects":   "gauge",
	"HeapReleased":  "gauge",
	"HeapSys":       "gauge",
	"LastGC":        "gauge",
	"Lookups":       "gauge",
	"MCacheInuse":   "gauge",
	"MCacheSys":     "gauge",
	"MSpanInuse":    "gauge",
	"MSpanSys":      "gauge",
	"Mallocs":       "gauge",
	"NextGC":        "gauge",
	"NumForcedGC":   "gauge",
	"NumGC":         "gauge",
	"OtherSys":      "gauge",
	"PauseTotalNs":  "gauge",
	"StackInuse":    "gauge",
	"StackSys":      "gauge",
	"Sys":           "gauge",
	"TotalAlloc":    "gauge",
}

func NewStore(key []byte) *Store {

	return &Store{
		memstat: &runtime.MemStats{},
		custom:  make(map[string]Metric),
		key:     key,
	}
}
func (s *Store) MemStats() []string {
	s.mu.RLock()
	res := make([]string, 0)
	r := reflect.ValueOf(s.memstat)
	for k, v := range runtimeMetrics {
		f := r.Elem().FieldByName(k)
		if !f.IsValid() {
			fmt.Println("Bad Name - ", k)
			return nil
		}
		res = append(res, fmt.Sprintf("/update/%s/%s/%v", v, k, f))
	}
	s.mu.RUnlock()
	sort.Strings(res)
	return res
}

func (s *Store) All() []string {
	res := make([]string, 0)
	for _, v := range s.Custom() {
		res = append(res, fmt.Sprintf("/update/%s/%s/%s", v.Type(), v.Name(), v))
	}
	res = append(res, s.MemStats()...)
	sort.Strings(res)
	return res
}

func (s *Store) AllMetrics() []Metrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make([]Metrics, 0)
	keys := make([]string, 0)
	for k := range s.custom {
		keys = append(keys, k)
	}
	for k := range runtimeMetrics {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	rM := reflect.ValueOf(s.memstat)

	for _, k := range keys {
		if _, ok := s.custom[k]; ok {
			m := s.custom[k].Metrics()
			if len(s.key) != 0 {
				if err := m.Sign(s.key); err != nil {
					logrus.Error(err)
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
		m := Metrics{ID: k, MType: runtimeMetrics[k]}
		switch runtimeMetrics[k] {
		case string(CounterType):
			m.Delta = GetInt64Pointer(f.Int())
		case string(GaugeType):
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
				logrus.Error(err)
				return nil
			}
		}
		res = append(res, m)
	}

	return res
}

func (s *Store) Custom() map[string]Metric {
	s.mu.Lock()
	result := make(map[string]Metric, len(s.custom))
	for k, v := range s.custom {
		result[k] = v
	}
	s.mu.Unlock()
	return result
}
func (s *Store) Save(client *http.Client, baseURL *string, isJSON bool, batch bool) error {
	if client != nil && baseURL != nil {
		if !isJSON {
			res := s.All()
			fmt.Println(res)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			errC := make(chan error, len(res))
			for i := 0; i < len(res); i++ {
				go func(ctx context.Context, c *http.Client, url string) {
					req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
					if err != nil {
						errC <- err
						return
					}
					req.Header.Set("Content-Type", "text/plain")
					resp, err := c.Do(req)
					if err != nil {
						errC <- err
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						body, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							errC <- err
							return
						}
						errC <- nil
						fmt.Println("save failed: ", string(body))
						return
					}
					_, err = io.Copy(io.Discard, resp.Body)
					if err != nil {
						errC <- err
						return
					}
					errC <- nil
				}(ctx, client, *baseURL+res[i])
			}
			for i := 0; i < len(res); i++ {
				if err := <-errC; err != nil {
					return err
				}
			}
			return nil
		}
		res := s.AllMetrics()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if batch {
			return sendMetrics(ctx, client, *baseURL+"/updates/", res)
		}
		errC := make(chan error, len(res))
		for i := 0; i < len(res); i++ {
			go sendMetric(ctx, errC, client, *baseURL+"/update/", res[i])
		}
		for i := 0; i < len(res); i++ {
			if err := <-errC; err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

func sendMetric(ctx context.Context, errC chan error, c *http.Client, url string, metric Metrics) {
	b, err := json.Marshal(metric)
	if err != nil {
		errC <- err
		return
	}
	logrus.Info(string(b))
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
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errC <- err
			return
		}
		errC <- nil
		fmt.Println("save failed: ", string(body))
		return
	}
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		errC <- err
		return
	}
	errC <- nil
}
func sendMetrics(ctx context.Context, c *http.Client, url string, metrics []Metrics) error {
	b, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	logrus.Info(string(b))
	buf := bytes.NewBuffer(b)
	req, err := http.NewRequestWithContext(ctx, "POST", url, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("save failed: %s. %v", string(body), err)
	}
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) AddCustom(m ...Metric) {
	s.mu.Lock()
	for _, v := range m {
		fmt.Println("Metric ", v.Name(), " was added")
		s.custom[v.Name()] = v
	}
	s.mu.Unlock()
}
func (s *Store) Scrape() error {
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
