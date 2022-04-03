package storage

import (
	"reflect"
	"testing"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/stretchr/testify/assert"
)

func TestStorage_List(t *testing.T) {
	type fields struct {
		v map[metrics.MetricType]map[string]map[string]interface{}
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
			fields: fields{v: NewStorage().v},
			args: args{
				targets: nil,
			},
			want: make(map[string][]string),
		},
		{
			name: "Наполненный Storage",
			fields: fields{v: map[metrics.MetricType]map[string]map[string]interface{}{
				metrics.CounterType: {
					"PollCount": map[string]interface{}{"1.1.1.1": []int{1, 2}},
				},
				metrics.GaugeType: {
					"RandomValue": map[string]interface{}{"1.1.1.2": float64(11.22)},
				},
			}},
			args: args{
				targets: nil,
			},
			want: map[string][]string{
				"1.1.1.1": []string{"counter - PollCount - [1 2]"},
				"1.1.1.2": []string{"gauge - RandomValue - 11.22"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				v: tt.fields.v,
			}
			if got := s.List(tt.args.targets...); !reflect.DeepEqual(got, tt.want) {

				t.Errorf("Storage.List() = %v, want %v", got, tt.want)
			}

		})
	}

}

func TestStorage_ListProm(t *testing.T) {
	type fields struct {
		v map[metrics.MetricType]map[string]map[string]interface{}
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
			fields: fields{v: NewStorage().v},
			args:   args{targets: nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				v: tt.fields.v,
			}
			assert.Panics(t, func() { s.ListProm(tt.args.targets...) })
		})
	}
}

func TestStorage_Update(t *testing.T) {
	type args struct {
		target string
		metric string
		name   string
		value  string
	}
	tests := []struct {
		name    string
		storage *Storage
		args    args
		err     error
		wantErr bool
	}{
		{
			name: "Нулевой target",
			args: args{
				target: "",
				metric: "1",
				name:   "1",
				value:  "1",
			},
			err: repositories.ErrWrongTarget,
		},
		{
			name: "Нулевой metric",
			args: args{
				target: "1",
				metric: "",
				name:   "1",
				value:  "1",
			},
			err: repositories.ErrWrongMetricType,
		},
		{
			name: "Нулевой name",
			args: args{
				target: "1",
				metric: "1",
				name:   "",
				value:  "1",
			},
			err: repositories.ErrWrongMetricType,
		},
		{
			name: "Нулевой value",
			args: args{
				target: "1",
				metric: "1",
				name:   "1",
				value:  "",
			},
			err: repositories.ErrWrongMetricValue,
		},
		{
			name: "Неправильная метрика",
			args: args{
				target: "1.1.1.1",
				metric: "11111",
			},
			err: repositories.ErrWrongMetricType,
		},
		{
			name: "Неправильно создано хранилище",
			args: args{
				target: "1.1.1.1",
				metric: "gauge",
				name:   "BlaBla",
				value:  "123.456",
			},
			storage: &Storage{v: map[metrics.MetricType]map[string]map[string]interface{}{}},
			err:     repositories.ErrWrongValueInStorage,
		},
		{
			name: "Правильная метрика gauge",
			args: args{
				target: "1.1.1.1",
				metric: "gauge",
				name:   "BlaBla",
				value:  "123.456",
			},
			storage: NewStorage(),
			err:     nil,
		},
		{
			name: "Неправильная метрика gauge",
			args: args{
				target: "1.1.1.1",
				metric: "gauge",
				name:   "BlaBla",
				value:  "none",
			},
			storage: NewStorage(),
			// не тестируется, так как отсекается регулярным выражением
			err: repositories.ErrWrongMetricValue,
		},
		{
			name: "Правильная метрика couter",
			args: args{
				target: "1.1.1.1",
				metric: "counter",
				name:   "BlaBla",
				value:  "123",
			},
			storage: NewStorage(),
			err:     nil,
		},
		{
			name: "Неправильная метрика couter",
			args: args{
				target: "1.1.1.1",
				metric: "counter",
				name:   "BlaBla",
				value:  "123.123",
			},
			storage: NewStorage(),
			err:     repositories.ErrWrongMetricValue,
		},
		// {
		// 	name: "Правильная запись couter в хранилище",
		// 	args: args{
		// 		target: "1.1.1.1",
		// 		metric: "counter",
		// 		name:   "BlaBla",
		// 		value:  "123",
		// 	},
		// 	storage: &Storage{
		// 		v: map[metrics.MetricType]map[string]map[string]interface{}{
		// 			metrics.CounterType: {"BlaBla": map[string]interface{}{"1.1.1.1": 1}},
		// 		},
		// 	},
		// 	err: nil,
		// },
		// {
		// 	name: "Неправильная запись gauge в хранилище",
		// 	args: args{
		// 		target: "1.1.1.1",
		// 		metric: "/gauge/BlaBla/123",
		// 	},
		// 	storage: &Storage{
		// 		v: map[metrics.MetricType]map[string]map[string]interface{}{
		// 			metrics.GaugeType: {"BlaBla": map[string]interface{}{"1.1.1.1": "ggg"}},
		// 		},
		// 	},
		// 	err: repositories.ErrWrongMetricType,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.storage.Update(tt.args.target, tt.args.metric, tt.args.name, tt.args.value), tt.err)
		})
	}
}
