package metrics

import (
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// EmulateError needs for test coverage
var emulateError bool

// Metric интерфейс универсальной метрики
type Metric interface {
	Name() string
	Desc() string
	Type() string
	String() string
	Scrape() error
	// Metrics преобразует в объект для хранения
	Metrics() Metrics
}

// Counter целочисленная метрика
type Counter interface {
	Metric
	Set(int64)
	Get() int64
}

// Gauge вещественная метрика
type Gauge interface {
	Metric
	Set(float64)
	Get() float64
}

// MetricType алиас для типа метрики
type MetricType string

const (
	CounterType MetricType = "counter"
	GaugeType   MetricType = "gauge"
)

const (
	tPollCount = iota + 1
	tRandomValue
	tTotalMemory
	tFreeMemory
	tCPUutilization1
)

var metricNames = map[int]string{
	tPollCount:       "PollCount",
	tRandomValue:     "RandomValue",
	tTotalMemory:     "TotalMemory",
	tFreeMemory:      "FreeMemory",
	tCPUutilization1: "CPUutilization1",
}
var metricDesc = map[int]string{
	tPollCount:       "Счётчик, увеличивающийся на 1 при каждом обновлении метрики из пакета runtime",
	tRandomValue:     "Обновляемое рандомное значение",
	tTotalMemory:     "Total amount of RAM on this system (gopsutil)",
	tFreeMemory:      "Available is what you really want (gopsutil)",
	tCPUutilization1: "CPU utilization (точное количество — по числу CPU, определяемому во время исполнения)",
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
}

// Scrape увеличивает собственное значение на единицу
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

// Scrape заменяет собственное значение на случайное
func (m *RandomValue) Scrape() error {
	rand.Seed(time.Now().Unix())
	*m = RandomValue(rand.Float64())
	return nil
}
func (m *RandomValue) Metrics() Metrics {
	return Metrics{ID: m.Name(), MType: m.Type(), Value: GetFloat64Pointer(float64(*m))}
}

// TotalMemory Обновляемое рандомное значение
type TotalMemory float64

var _ Gauge = new(TotalMemory)

func (m TotalMemory) Name() string {
	return metricNames[tTotalMemory]
}
func (m TotalMemory) Desc() string {
	return metricDesc[tTotalMemory]
}
func (m TotalMemory) Type() string {
	return "gauge"
}
func (m TotalMemory) String() string {
	return fmt.Sprintf("%f", m)
}
func (m *TotalMemory) Get() float64 {
	return float64(*m)
}
func (m *TotalMemory) Set(i float64) {
	*m = TotalMemory(i)
}
func (m *TotalMemory) Scrape() error {
	v, err := mem.VirtualMemory()
	if err != nil || emulateError {
		if err == nil {
			err = errors.New("emulateError")
		}
		return err
	}
	*m = TotalMemory(float64(v.Total))
	return nil
}
func (m *TotalMemory) Metrics() Metrics {
	return Metrics{ID: m.Name(), MType: m.Type(), Value: GetFloat64Pointer(float64(*m))}
}

// FreeMemory Обновляемое рандомное значение
type FreeMemory float64

var _ Gauge = new(FreeMemory)

func (m FreeMemory) Name() string {
	return metricNames[tFreeMemory]
}
func (m FreeMemory) Desc() string {
	return metricDesc[tFreeMemory]
}
func (m FreeMemory) Type() string {
	return "gauge"
}
func (m FreeMemory) String() string {
	return fmt.Sprintf("%f", m)
}
func (m *FreeMemory) Get() float64 {
	return float64(*m)
}
func (m *FreeMemory) Set(i float64) {
	*m = FreeMemory(i)
}
func (m *FreeMemory) Scrape() error {
	v, err := mem.VirtualMemory()
	if err != nil || emulateError {
		if err == nil {
			err = errors.New("emulateError")
		}
		return err
	}
	*m = FreeMemory(float64(v.Free))
	return nil
}
func (m *FreeMemory) Metrics() Metrics {
	return Metrics{ID: m.Name(), MType: m.Type(), Value: GetFloat64Pointer(float64(*m))}
}

// FreeMemory Обновляемое рандомное значение
type CPUutilization1 float64

var _ Gauge = new(FreeMemory)

func (m CPUutilization1) Name() string {
	return metricNames[tCPUutilization1]
}
func (m CPUutilization1) Desc() string {
	return metricDesc[tCPUutilization1]
}
func (m CPUutilization1) Type() string {
	return "gauge"
}
func (m CPUutilization1) String() string {
	return fmt.Sprintf("%f", m)
}
func (m *CPUutilization1) Get() float64 {
	return float64(*m)
}
func (m *CPUutilization1) Set(i float64) {
	*m = CPUutilization1(i)
}
func (m *CPUutilization1) Scrape() error {

	c, err := cpu.Percent(time.Second, false)
	if err != nil || emulateError {
		if err == nil {
			err = errors.New("emulateError")
		}
		return err
	}
	var sum float64
	for _, v := range c {
		sum = +v
	}
	*m = CPUutilization1(sum / float64(runtime.NumCPU()))
	return nil
}
func (m *CPUutilization1) Metrics() Metrics {
	return Metrics{ID: m.Name(), MType: m.Type(), Value: GetFloat64Pointer(float64(*m))}
}
