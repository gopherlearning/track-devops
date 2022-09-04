package metrics

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrNoSuchMetricType = errors.New("нет метрики такого типа")
	ErrTooSHortKey      = errors.New("слошком короткий ключ")
)

// Metrics используется для универсального представления метрики
type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
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
			return ""
		}
		return fmt.Sprintf(`%d`, *s.Delta)
	case string(GaugeType):
		if s.Value == nil {
			return ""
		}
		return fmt.Sprintf(`%g`, *s.Value)
	default:
		return ""
	}
}

// StringFull show metric for listing in console
func (s Metrics) StringFull() string {
	var hash string
	if len(s.Hash) != 0 {
		hash = fmt.Sprintf(" - %s", s.Hash)
	}
	switch s.MType {
	case string(CounterType):
		if s.Delta == nil {
			return fmt.Sprintf(`%s - %s%s`, s.MType, s.ID, hash)
		}
		return fmt.Sprintf(`%s - %s - %d%s`, s.MType, s.ID, *s.Delta, hash)
	case string(GaugeType):
		if s.Value == nil {
			return fmt.Sprintf(`%s - %s%s`, s.MType, s.ID, hash)
		}
		return fmt.Sprintf(`%s - %s - %g%s`, s.MType, s.ID, *s.Value, hash)
	default:
		return ""
	}
}

// MarshalJSON реализует интерфейс json.Marshaler.
func (s *Metrics) Sign(key []byte) error {
	if len(key) < 3 {
		return ErrTooSHortKey
	}
	var src []byte
	switch s.MType {
	case string(CounterType):
		src = []byte(fmt.Sprintf("%s:counter:%d", s.ID, *s.Delta))
	case string(GaugeType):
		src = []byte(fmt.Sprintf("%s:gauge:%f", s.ID, *s.Value))
	default:
		return ErrNoSuchMetricType
	}
	h := hmac.New(sha256.New, key)
	h.Write(src)
	s.Hash = hex.EncodeToString(h.Sum(nil))
	return nil
}

// MarshalJSON реализует интерфейс json.Marshaler.
func (s *Metrics) MarshalJSON() ([]byte, error) {
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
	switch raw.MType {
	case "counter":
		(*s) = Metrics{
			ID:    raw.ID,
			Hash:  raw.Hash,
			MType: raw.MType,
			Delta: raw.Delta,
		}
	case "gauge":
		(*s) = Metrics{
			ID:    raw.ID,
			Hash:  raw.Hash,
			MType: raw.MType,
			Value: raw.Value,
		}
	default:
		return ErrNoSuchMetricType
	}
	return nil
}

// GetInt64Pointer return int64 pointer
func GetInt64Pointer(val int64) *int64 {
	return &val
}

// GetFloat64Pointer return float64 pointer
func GetFloat64Pointer(val float64) *float64 {
	return &val
}
