package metrics

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"runtime"
	"sort"
	"sync"
)

type Store struct {
	mu      sync.RWMutex
	memstat *runtime.MemStats
	custom  map[string]Metric
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

func NewStore() *Store {
	return &Store{
		memstat: &runtime.MemStats{},
		custom:  make(map[string]Metric),
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
	sort.Strings(res)
	return append(res, s.MemStats()...)
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
func (s *Store) Save(client *http.Client, baseURL *string) error {
	res := s.All()
	fmt.Println(res)
	if client != nil && baseURL != nil {
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
	if _, ok := s.custom[metricNames[tPollCount]]; ok {
		i := s.custom[metricNames[tPollCount]].(*PollCount).Get() + 1
		s.custom[metricNames[tPollCount]].(*PollCount).Set(i)
	}
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
