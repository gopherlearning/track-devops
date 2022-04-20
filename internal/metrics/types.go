package metrics

import (
	"fmt"
	"math/rand"
	"time"
)

type Metric interface {
	Name() string
	Desc() string
	Type() string
	String() string
	Scrape() error
	Metrics() Metrics
}

type Counter interface {
	Metric
	Set(int64)
	Get() int64
}

type Gauge interface {
	Metric
	Set(float64)
	Get() float64
}

type MetricType string

const (
	CounterType MetricType = "counter"
	GaugeType   MetricType = "gauge"
)

const (
	tPollCount = iota + 1
	tRandomValue
)

var metricNames = map[int]string{
	tPollCount:   "PollCount",
	tRandomValue: "RandomValue",
}
var metricDesc = map[int]string{
	tPollCount:   "Счётчик, увеличивающийся на 1 при каждом обновлении метрики из пакета runtime",
	tRandomValue: "Обновляемое рандомное значение",
}

// PollCount Счётчик, увеличивающийся на 1 при каждом обновлении метрики из пакета runtime
type PollCount int64

var _ Counter = new(PollCount)

func (m PollCount) Name() string {
	return metricNames[tPollCount]
}
func (m PollCount) Desc() string {
	return metricDesc[tPollCount]
}
func (m PollCount) Type() string {
	return "counter"
}
func (m PollCount) String() string {
	return fmt.Sprintf("%d", m)
}
func (m *PollCount) Get() int64 {
	return int64(*m)
}
func (m *PollCount) Set(i int64) {
	*m = PollCount(i)
	// m.v = i
}
func (m *PollCount) Scrape() error {
	a := *m + 1
	*m = a
	return nil
}
func (m *PollCount) Metrics() Metrics {
	return Metrics{ID: m.Name(), MType: m.Type(), Delta: GetInt64Pointer(int64(*m))}
}

// RandomValue Обновляемое рандомное значение
type RandomValue float64

var _ Gauge = new(RandomValue)

func (m RandomValue) Name() string {
	return metricNames[tRandomValue]
}
func (m RandomValue) Desc() string {
	return metricDesc[tRandomValue]
}
func (m RandomValue) Type() string {
	return "gauge"
}
func (m RandomValue) String() string {
	return fmt.Sprintf("%f", m)
}
func (m *RandomValue) Get() float64 {
	return float64(*m)
}
func (m *RandomValue) Set(i float64) {
	*m = RandomValue(i)
}
func (m *RandomValue) Scrape() error {
	rand.Seed(time.Now().Unix())
	*m = RandomValue(rand.Float64())
	return nil
}
func (m *RandomValue) Metrics() Metrics {
	return Metrics{ID: m.Name(), MType: m.Type(), Value: GetFloat64Pointer(float64(*m))}
}
