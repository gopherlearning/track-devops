package local

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
)

func newStorage(t *testing.T) *Storage {
	s, err := NewStorage(false, nil, zap.L())
	require.NoError(t, err)
	return s
}
func TestStorage_List(t *testing.T) {
	tmp, err := os.CreateTemp(os.TempDir(), "go_test")
	assert.NoError(t, err)
	assert.FileExists(t, tmp.Name())
	storeInt := time.Second
	st, err := NewStorage(false, &storeInt, zap.L(), tmp.Name())
	assert.NoError(t, err)
	assert.NotNil(t, st)
	assert.NoError(t, st.Save())

	st, err = NewStorage(true, &storeInt, zap.L(), tmp.Name())
	assert.NoError(t, err)
	assert.NotNil(t, st)
	assert.NoError(t, st.Save())
	time.Sleep(2 * time.Second)
	type fields struct {
		metrics map[string][]metrics.Metrics
	}
	type args struct {
		target string
	}
	tests := []struct {
		name   string
		want   map[string][]string
		fields fields
		args   args
	}{
		{
			name:   "Пустой Storage",
			fields: fields{metrics: newStorage(t).metrics},
			args: args{
				target: "",
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
				target: "",
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
			if got, _ := s.List(context.TODO()); !reflect.DeepEqual(got, tt.want) {

				t.Errorf("Storage.List() = %v, want %v", got, tt.want)
			}

		})
	}

}

func TestStorage_Update(t *testing.T) {
	type args struct {
		target string
		metric metrics.Metrics
	}
	tests := []struct {
		storage *Storage
		args    args
		err     error
		name    string
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
			err := tt.storage.UpdateMetric(context.TODO(), tt.args.target, tt.args.metric)
			assert.Equal(t, err, tt.err)
		})
	}

}
