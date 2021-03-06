package local

import (
	"reflect"
	"testing"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStorage(t *testing.T) *Storage {
	s, err := NewStorage(false, nil)
	require.NoError(t, err)
	return s
}
func TestStorage_List(t *testing.T) {

	type fields struct {
		metrics map[string][]metrics.Metrics
	}
	type args struct {
		targets []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string][]string
	}{
		{
			name:   "Пустой Storage",
			fields: fields{metrics: newStorage(t).metrics},
			args: args{
				targets: nil,
			},
			want: make(map[string][]string),
		},
		{
			name: "Наполненный Storage",
			fields: fields{
				metrics: map[string][]metrics.Metrics{
					"1.1.1.1": {
						metrics.Metrics{ID: "PollCount", MType: string(metrics.CounterType), Delta: metrics.GetInt64Pointer(3)},
					},
					"1.1.1.2": {
						metrics.Metrics{ID: "RandomValue", MType: string(metrics.GaugeType), Value: metrics.GetFloat64Pointer(11.22)},
					},
				},
			},
			args: args{
				targets: nil,
			},
			want: map[string][]string{
				"1.1.1.1": {"counter - PollCount - 3"},
				"1.1.1.2": {"gauge - RandomValue - 11.22"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				metrics: tt.fields.metrics,
			}
			if got := s.List(tt.args.targets...); !reflect.DeepEqual(got, tt.want) {

				t.Errorf("Storage.List() = %v, want %v", got, tt.want)
			}

		})
	}

}

func TestStorage_ListProm(t *testing.T) {
	type fields struct {
		metrics map[string][]metrics.Metrics
	}
	type args struct {
		targets []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{
			name:   "Не реализована",
			fields: fields{metrics: newStorage(t).metrics},
			args:   args{targets: nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				metrics: tt.fields.metrics,
			}
			assert.Panics(t, func() { s.ListProm(tt.args.targets...) })
		})
	}
}

func TestStorage_Update(t *testing.T) {
	type args struct {
		target string
		metric metrics.Metrics
	}
	tests := []struct {
		name    string
		storage *Storage
		args    args
		err     error
		wantErr bool
	}{
		{
			name: "Несуществующий тип",
			args: args{
				target: "1",
				metric: metrics.Metrics{
					MType: "unknown",
					ID:    "1",
					Delta: metrics.GetInt64Pointer(1),
				},
			},
			storage: newStorage(t),
			err:     repositories.ErrWrongMetricType,
		},
		{
			name: "Нулевой target",
			args: args{
				target: "",
				metric: metrics.Metrics{
					MType: "1",
					ID:    "1",
					Delta: metrics.GetInt64Pointer(1),
				},
			},
			storage: newStorage(t),
			err:     repositories.ErrWrongTarget,
		},
		{
			name: "Нулевой metric",
			args: args{
				target: "1",
				metric: metrics.Metrics{
					MType: "",
					ID:    "1",
					Delta: metrics.GetInt64Pointer(1),
				},
			},
			storage: newStorage(t),
			err:     repositories.ErrWrongMetricType,
		},
		{
			name: "Нулевой name",
			args: args{
				target: "1",
				metric: metrics.Metrics{
					MType: "1",
					ID:    "",
					Delta: metrics.GetInt64Pointer(1),
				},
			},
			storage: newStorage(t),
			err:     repositories.ErrWrongMetricID,
		},
		{
			name: "Нулевой value",
			args: args{
				target: "1",
				metric: metrics.Metrics{
					MType: "counter",
					ID:    "1",
				},
			},
			storage: newStorage(t),
			err:     repositories.ErrWrongMetricValue,
		},
		{
			name: "Правильная метрика gauge",
			args: args{
				target: "1.1.1.1",
				metric: metrics.Metrics{
					MType: "gauge",
					ID:    "BlaBla",
					Value: metrics.GetFloat64Pointer(123.456),
				},
			},
			storage: newStorage(t),
			err:     nil,
		},
		{
			name: "Правильная метрика couter",
			args: args{
				target: "1.1.1.1",
				metric: metrics.Metrics{
					MType: "counter",
					ID:    "BlaBla",
					Delta: metrics.GetInt64Pointer(123),
				},
			},
			storage: newStorage(t),
			err:     nil,
		},
		{
			name: "Правильная запись couter в хранилище",
			args: args{
				target: "1.1.1.1",
				metric: metrics.Metrics{
					MType: "counter",
					ID:    "BlaBla",
					Delta: metrics.GetInt64Pointer(123),
				},
			},
			storage: &Storage{
				metrics: map[string][]metrics.Metrics{
					"1.1.1.1": {
						{ID: "BlaBla", MType: string(metrics.CounterType), Delta: metrics.GetInt64Pointer(10)},
					},
				},
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.storage.UpdateMetric(tt.args.target, tt.args.metric)
			assert.Equal(t, err, tt.err)
		})
	}
}
