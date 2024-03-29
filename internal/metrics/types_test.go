package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/gopherlearning/track-devops/internal/agent"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"go.uber.org/zap"
// )

func TestMetrics(t *testing.T) {
	tests := []struct {
		name        string
		m           Metric
		wantName    string
		wantType    MetricType
		wantDesc    string
		wantMetrics Metrics
		wantScrape  error
		wantString  string
	}{
		{
			name:        "PollCount",
			m:           new(PollCount),
			wantType:    CounterType,
			wantName:    "PollCount",
			wantString:  "1",
			wantDesc:    "Счётчик, увеличивающийся на 1 при каждом обновлении метрики из пакета runtime",
			wantMetrics: new(PollCount).Metrics(),
			wantScrape:  nil,
		},
		{
			name:        "RandomValue",
			m:           new(RandomValue),
			wantType:    GaugeType,
			wantName:    "RandomValue",
			wantString:  ".",
			wantDesc:    "Обновляемое рандомное значение",
			wantMetrics: new(RandomValue).Metrics(),
			wantScrape:  nil,
		},
		{
			name:        "TotalMemory",
			m:           new(TotalMemory),
			wantType:    GaugeType,
			wantName:    "TotalMemory",
			wantString:  ".",
			wantDesc:    "Total amount of RAM on this system (gopsutil)",
			wantMetrics: new(TotalMemory).Metrics(),
			wantScrape:  nil,
		},
		{
			name:        "FreeMemory",
			m:           new(FreeMemory),
			wantType:    GaugeType,
			wantName:    "FreeMemory",
			wantString:  ".",
			wantDesc:    "Available is what you really want (gopsutil)",
			wantMetrics: new(FreeMemory).Metrics(),
			wantScrape:  nil,
		},
		{
			name:        "CPUutilization1",
			m:           new(CPUutilization1),
			wantType:    GaugeType,
			wantName:    "CPUutilization1",
			wantString:  ".",
			wantDesc:    "CPU utilization (точное количество — по числу CPU, определяемому во время исполнения)",
			wantMetrics: new(CPUutilization1).Metrics(),
			wantScrape:  nil,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.m.Type() {
			case "counter":
				tt.m.(Counter).Get()
				tt.m.(Counter).Set(0)
			case "gauge":
				tt.m.(Gauge).Get()
				tt.m.(Gauge).Set(0)
			default:
			}
			assert.Equal(t, tt.wantDesc, tt.m.Desc())
			assert.Equal(t, tt.wantName, tt.m.Name())
			assert.Equal(t, tt.wantMetrics, tt.m.Metrics())
			assert.Equal(t, tt.wantScrape, tt.m.Scrape())
			assert.Contains(t, tt.m.String(), tt.wantString)
			assert.Equal(t, tt.wantType, tt.m.Type())
		})
	}
}

func TestSendMetrics(t *testing.T) {
	ms := &Metrics{MType: "test", ID: "test", Delta: nil, Value: nil}
	badURL := "#a"
	var ctxnil context.Context
	ctx := context.TODO()
	defaultClient := &testClient{http: http.DefaultClient, t: "http"}
	t.Run("nil context, bad metric", func(t *testing.T) {
		errs := make(chan error, 1)
		sendMetric(ctxnil, errs, defaultClient, badURL, *ms)
		assert.Error(t, <-errs)
	})
	t.Run("normal context, bad metric", func(t *testing.T) {
		errs := make(chan error, 1)
		sendMetric(ctx, errs, defaultClient, badURL, *ms)
		assert.Error(t, <-errs)
	})
	ms = &Metrics{MType: "counter", ID: "test", Delta: GetInt64Pointer(1111), Value: nil}
	errs := make(chan error, 1)
	sendMetric(ctxnil, errs, defaultClient, badURL, *ms)
	assert.Error(t, <-errs)
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(`Not OK`))
	}))
	errs = make(chan error, 1)
	sendMetric(ctx, errs, defaultClient, server.URL, *ms)
	assert.Error(t, <-errs)
	assert.Error(t, sendMetrics(ctx, defaultClient, server.URL, []Metrics{*ms}))
	emulateError = true
	sendMetric(ctx, errs, defaultClient, server.URL, *ms)
	assert.Error(t, <-errs)
	assert.Error(t, sendMetrics(ctx, defaultClient, server.URL, []Metrics{*ms}))
	emulateError = false
	server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}))
	errs = make(chan error, 1)
	sendMetric(ctx, errs, defaultClient, server.URL, *ms)
	assert.Nil(t, <-errs)
	assert.Nil(t, sendMetrics(ctx, defaultClient, server.URL, []Metrics{*ms}))
	emulateError = true
	sendMetric(ctx, errs, defaultClient, server.URL, *ms)
	assert.Error(t, <-errs)
	assert.Error(t, sendMetrics(ctx, defaultClient, server.URL, []Metrics{*ms}))
	assert.Error(t, sendMetrics(ctxnil, defaultClient, server.URL, []Metrics{*ms}))
	emulateError = false
}

type testClient struct {
	http *http.Client
	t    string
}

func (c *testClient) Do(req *http.Request) (*http.Response, error) {
	return c.http.Do(req)
}

func (c *testClient) SendMetrics(context.Context, []Metrics) error {

	return nil
}
func (c *testClient) Type() string {
	return c.t
}

func TestSave(t *testing.T) {
	wg := &sync.WaitGroup{}
	m := NewStore([]byte("secret"), zap.L())
	ctx := context.TODO()
	m.AddCustom(
		new(PollCount),
		new(RandomValue),
	)
	t.Run("транспорт не поддерживается", func(t *testing.T) {
		wg.Add(1)
		assert.ErrorContains(t, m.Save(ctx, wg, &testClient{http: http.DefaultClient, t: "test"}, "", true, false), "транспорт не поддерживается")
	})
	t.Run("тест grpc", func(t *testing.T) {
		wg.Add(1)
		assert.NoError(t, m.Save(ctx, wg, &testClient{http: http.DefaultClient, t: "grpc"}, "", true, false))
	})
	assert.Nil(t, m.Scrape())
	defaultClient := &testClient{http: http.DefaultClient, t: "http"}
	assert.Error(t, sendMetrics(ctx, defaultClient, "", nil))
	wg.Add(1)
	assert.NoError(t, m.Save(ctx, wg, nil, "", true, false))
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, "", true, false))
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, "", true, true))
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, "", false, true))
	badURL := "#a"
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, badURL, false, true))
	var ctxnil context.Context
	wg.Add(1)
	assert.Error(t, m.Save(ctxnil, wg, defaultClient, badURL, false, true))
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		// assert.Equal(t, req.URL.String(), "/updates/")
		rw.WriteHeader(http.StatusOK)
		// rw.Write([]byte(`OK`))
	}))
	wg.Add(1)
	assert.NoError(t, m.Save(ctx, wg, defaultClient, server.URL, true, true))
	wg.Add(1)
	assert.NoError(t, m.Save(ctx, wg, defaultClient, server.URL, true, false))
	wg.Add(1)
	assert.NoError(t, m.Save(ctx, wg, defaultClient, server.URL, false, false))
	emulateError = true
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, server.URL, true, true))
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, server.URL, true, false))
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, server.URL, false, false))
	emulateError = false
	server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// assert.Equal(t, req.URL.String(), "/updates/")
		rw.WriteHeader(http.StatusInternalServerError)
		// rw.Write([]byte(`OK`))
	}))
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, server.URL, true, true))
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, server.URL, true, false))
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, server.URL, false, false))
	emulateError = true
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, server.URL, true, true))
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, server.URL, true, false))
	wg.Add(1)
	assert.Error(t, m.Save(ctx, wg, defaultClient, server.URL, false, false))
	emulateError = false
}

func TestAllMetrics(t *testing.T) {
	m := NewStore([]byte("1234"), zap.L())
	m.AddCustom(new(PollCount))
	assert.NotEmpty(t, m.AllMetrics())
	for _, v := range m.AllMetrics() {
		assert.NotEmpty(t, v.String())
		assert.Contains(t, v.StringFull(), " - ")
	}
	m = NewStore([]byte("12"), zap.L())
	assert.Nil(t, m.AllMetrics())
	runtimeMetricsOld := make(map[string]MetricType)
	for k, v := range m.runtimeMetrics {
		runtimeMetricsOld[k] = v
	}
	m.runtimeMetrics = nil
	m.AddCustom(new(TotalMemory))
	m.AddCustom(new(PollCount))
	assert.Nil(t, m.AllMetrics())
	m.runtimeMetrics = make(map[string]MetricType)
	for k, v := range runtimeMetricsOld {
		m.runtimeMetrics[k] = v
	}
	t.Run("emulate valid error", func(t *testing.T) {
		emulateError = true
		m.mu.Lock()
		m.runtimeMetrics["TestBadName"] = "guag"

		m.mu.Unlock()
		assert.Nil(t, m.AllMetrics())
		assert.Nil(t, m.MemStats())
		emulateError = false
		delete(m.runtimeMetrics, "TestBadName")
	})

}

func TestStore(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		assert.Equal(t, req.URL.String(), "/")
		// Send response to be tested
		rw.Write([]byte(`OK`))
	}))
	// Close the server when test finishes
	defer server.Close()
	m := NewStore([]byte("secret"), zap.L())
	m.AddCustom(
		new(TestErrorMetric),
	)
	assert.Error(t, m.Scrape(), errors.New("TestErrorMetric error"))
	emulateError = true
	m = NewStore([]byte("secret"), zap.L())
	m.AddCustom(new(CPUutilization1))
	assert.Error(t, m.Scrape())
	m = NewStore([]byte("secret"), zap.L())
	m.AddCustom(new(TotalMemory))
	assert.Error(t, m.Scrape())
	m = NewStore([]byte("secret"), zap.L())
	m.AddCustom(new(FreeMemory))
	assert.Error(t, m.Scrape())
	emulateError = false
	m = NewStore([]byte("secret"), zap.L())
	m.AddCustom(
		new(PollCount),
		new(RandomValue),
		new(TotalMemory),
		new(FreeMemory),
		new(CPUutilization1),
	)
	require.NotEmpty(t, m.MemStats())
	require.NotEmpty(t, m.All())

	m.key = []byte("secret")
	ms := &Metrics{MType: "counter", ID: "test", Delta: nil, Value: nil}
	assert.Equal(t, ms.String(), "")
	ms = &Metrics{MType: GaugeType, ID: "test", Delta: nil, Value: nil}
	assert.Equal(t, ms.String(), "")
	ms = &Metrics{MType: "test", ID: "test", Delta: nil, Value: nil}

	assert.Equal(t, ms.String(), "")
	assert.Equal(t, ms.Sign([]byte("secret")), ErrNoSuchMetricType)
	assert.Equal(t, ms.Sign(nil), ErrTooSHortKey)
	assert.Contains(t, ms.StringFull(), "")
	_, err := ms.MarshalJSON()
	assert.Equal(t, err, ErrNoSuchMetricType)
	err = ms.UnmarshalJSON([]byte(""))
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
	err = ms.UnmarshalJSON([]byte("{}"))
	assert.Contains(t, err.Error(), "нет метрики такого типа")
	ms = &Metrics{MType: CounterType, ID: "test"}
	assert.Contains(t, ms.StringFull(), " - ")
	ms = &Metrics{MType: GaugeType, ID: "test"}
	assert.Contains(t, ms.StringFull(), " - ")
	m.runtimeMetrics["test"] = "badMetric"
	assert.Nil(t, m.MemStats())
}

type TestErrorMetric int64

var _ Counter = new(TestErrorMetric)

func (m TestErrorMetric) Name() string {
	return "TestErrorMetric"
}
func (m TestErrorMetric) Desc() string {
	return "TestErrorMetric"
}
func (m TestErrorMetric) Type() MetricType {
	return CounterType
}

func (m TestErrorMetric) String() string {
	return fmt.Sprintf("%d", m)
}

func (m *TestErrorMetric) Get() int64 {
	return int64(*m)
}
func (m *TestErrorMetric) Set(i int64) {
	*m = TestErrorMetric(i)
}

// Scrape увеличивает собственное значение на единицу
func (m *TestErrorMetric) Scrape() error {
	return errors.New("TestErrorMetric error")
}

func (m *TestErrorMetric) Metrics() Metrics {
	return Metrics{ID: m.Name(), MType: m.Type(), Delta: GetInt64Pointer(int64(*m))}
}
