package metrics

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrNoSuchMetricType = errors.New("нет метрики такого типа")
)

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

type MetricsJSON struct {
	Metrics
}
type MetricsAlias Metrics

// MarshalJSON реализует интерфейс json.Marshaler.
func (s Metrics) String() string {
	switch s.MType {
	case string(CounterType):
		if s.Delta == nil {
			return fmt.Sprintf(`%s - %s`, s.MType, s.ID)
		}
		return fmt.Sprintf(`%s - %s - %d`, s.MType, s.ID, *s.Delta)
	case string(GaugeType):
		if s.Value == nil {
			return fmt.Sprintf(`%s - %s`, s.MType, s.ID)
		}
		return fmt.Sprintf(`%s - %s - %g`, s.MType, s.ID, *s.Value)
	default:
		return ""
	}
}

// MarshalJSON реализует интерфейс json.Marshaler.
func (s *Metrics) MarshalJSON() ([]byte, error) {
	// logrus.Info(s.MType)
	switch s.MType {
	case string(CounterType):
		aliasValue := struct {
			MetricsAlias
			Delta int64 `json:"delta"`
		}{
			MetricsAlias: MetricsAlias(*s),
			Delta:        int64(*s.Delta),
		}
		return json.Marshal(aliasValue)
	case string(GaugeType):
		aliasValue := struct {
			MetricsAlias
			Value float64 `json:"value"`
		}{
			MetricsAlias: MetricsAlias(*s),
			Value:        float64(*s.Value),
		}
		return json.Marshal(aliasValue)
	default:
		return nil, ErrNoSuchMetricType
	}
}

// UnmarshalJSON реализует интерфейс json.Unmarshaler.
func (s *Metrics) UnmarshalJSON(data []byte) error {
	raw := MetricsAlias{}
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	// logrus.Info(raw.MType)
	switch raw.MType {
	case "counter":
		(*s) = Metrics{
			ID:    raw.ID,
			MType: raw.MType,
			Delta: raw.Delta,
		}
	case "gauge":
		(*s) = Metrics{
			ID:    raw.ID,
			MType: raw.MType,
			Value: raw.Value,
		}
	default:
		return ErrNoSuchMetricType
	}
	return nil
}

func GetInt64Pointer(val int64) *int64 {
	return &val
}
func GetFloat64Pointer(val float64) *float64 {
	return &val
}
