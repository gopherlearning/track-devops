package metrics

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics(t *testing.T) {
	tests := []struct {
		name        string
		m           Metric
		wantName    string
		wantType    string
		wantDesc    string
		wantMetrics Metrics
		wantScrape  error
		wantString  string
	}{
		{
			name:        "PollCount",
			m:           new(PollCount),
			wantType:    "counter",
			wantName:    "PollCount",
			wantString:  "1",
			wantDesc:    "Счётчик, увеличивающийся на 1 при каждом обновлении метрики из пакета runtime",
			wantMetrics: new(PollCount).Metrics(),
			wantScrape:  nil,
		},
		{
			name:        "RandomValue",
			m:           new(RandomValue),
			wantType:    "gauge",
			wantName:    "RandomValue",
			wantString:  ".",
			wantDesc:    "Обновляемое рандомное значение",
			wantMetrics: new(RandomValue).Metrics(),
			wantScrape:  nil,
		},
		{
			name:        "TotalMemory",
			m:           new(TotalMemory),
			wantType:    "gauge",
			wantName:    "TotalMemory",
			wantString:  ".",
			wantDesc:    "Total amount of RAM on this system (gopsutil)",
			wantMetrics: new(TotalMemory).Metrics(),
			wantScrape:  nil,
		},
		{
			name:        "FreeMemory",
			m:           new(FreeMemory),
			wantType:    "gauge",
			wantName:    "FreeMemory",
			wantString:  ".",
			wantDesc:    "Available is what you really want (gopsutil)",
			wantMetrics: new(FreeMemory).Metrics(),
			wantScrape:  nil,
		},
		{
			name:        "CPUutilization1",
			m:           new(CPUutilization1),
			wantType:    "gauge",
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

func TestStore(t *testing.T) {
	m := NewStore([]byte("secret"))
	m.AddCustom(
		new(PollCount),
		new(RandomValue),
		new(TotalMemory),
		new(FreeMemory),
		new(CPUutilization1),
	)
	require.NotEmpty(t, m.MemStats())
	require.NotEmpty(t, m.All())
	assert.Nil(t, m.Save(nil, nil, true, false))
	assert.Error(t, m.Save(&http.Client{}, new(string), true, false))
	assert.Error(t, m.Save(&http.Client{}, new(string), true, true))
	assert.Error(t, m.Save(&http.Client{}, new(string), false, true))
	badURL := "bla"
	assert.Error(t, m.Save(&http.Client{}, &badURL, false, true))
	assert.Equal(t, m.Scrape(), nil)
	for _, v := range m.AllMetrics() {
		assert.NotEmpty(t, v.String())
		assert.Contains(t, v.StringFull(), " - ")
	}
	m.key = []byte("1")
	assert.Nil(t, m.AllMetrics())
	m.key = []byte("secret")
	ms := &Metrics{MType: "counter", ID: "test", Delta: nil, Value: nil}
	assert.Equal(t, ms.String(), "")
	ms = &Metrics{MType: "gauge", ID: "test", Delta: nil, Value: nil}
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
	ms = &Metrics{MType: string(CounterType), ID: "test"}
	assert.Contains(t, ms.StringFull(), " - ")
	ms = &Metrics{MType: string(GaugeType), ID: "test"}
	assert.Contains(t, ms.StringFull(), " - ")
	runtimeMetrics["test"] = "badMetric"
	assert.Nil(t, m.MemStats())
}
