package handlers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gopherlearning/track-devops/cmd/server/storage"
	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoHandler_Get(t *testing.T) {
	// type fields struct {
	// 	v map[metrics.MetricType]map[string]map[string]interface{}
	// }
	tests := []struct {
		name string
		// fields  fields
		request string
		status  int
		target  string
		value   map[string]interface{}
		want    string
	}{
		{
			name:    "Существующее значение",
			request: "/value/counter/PollCount",
			status:  http.StatusOK,
			value:   map[string]interface{}{"counter": int64(123), "gauge": float64(0)},
			target:  "192.0.2.1",
			want:    "counter - PollCount - 123",
		},
		{
			name:    "Несуществующее значение",
			request: "/value/counter/Unknown",
			status:  http.StatusNotFound,
			target:  "192.0.2.1",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := storage.NewStorage()
			if len(tt.want) != 0 {
				m := strings.Split(tt.request, "/")
				// require.NoError(t, s.Update(tt.target, m[2], m[3], tt.value))
				require.NoError(t, s.UpdateMetric(tt.target, metrics.Metrics{MType: m[2], ID: m[3], Delta: metrics.GetInt64Pointer(tt.value["counter"].(int64)), Value: metrics.GetFloat64Pointer(tt.value["gauge"].(float64))}))
			}
			handler := NewEchoHandler(s)
			request := httptest.NewRequest(http.MethodGet, tt.request, nil)
			w := httptest.NewRecorder()
			handler.Echo().ServeHTTP(w, handler.Echo().NewContext(request, w).Request())
			result := w.Result()

			assert.Equal(t, tt.status, result.StatusCode)
			body, err := ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want, string(body))

		})
	}

}
func TestEchoHandler_Update(t *testing.T) {
	type fields struct {
		s repositories.Repository
	}
	type want struct {
		statusCode int
		body       string
		value1     interface{}
		value2     interface{}
	}
	tests := []struct {
		name     string
		fields   fields
		content  string
		request1 string
		request2 string
		method   string
		want     want
	}{
		{
			name:   "TestIteration2/TestCounterHandlers/invalid_value",
			fields: fields{s: storage.NewStorage()},
			// content:  "text/plain",
			method:   http.MethodPost,
			request1: "/update/counter/testCounter/none",
			request2: "",
			want: want{
				statusCode: http.StatusBadRequest,
				body:       "неверное значение метрики",
				value1:     "",
				value2:     "",
			},
		},
		{
			name:   "TestIteration2/TestCounterHandlers/update",
			fields: fields{s: storage.NewStorage()},
			// content:  "text/plain",
			method:   http.MethodPost,
			request1: "/update/counter/testCounter/100",
			request2: "/update/counter/testCounter/101",
			want: want{
				statusCode: http.StatusOK,
				body:       "",
				value1:     100,
				value2:     201,
			},
		},
		{
			name:   "TestIteration2/TestCounterHandlers/without_id",
			fields: fields{s: storage.NewStorage()},
			// content:  "text/plain",
			method:   http.MethodPost,
			request1: "/update/counter/",
			request2: "",
			want: want{
				statusCode: http.StatusNotFound,
				body:       "{\"message\":\"Not Found\"}\n",
				value1:     "",
				value2:     "",
			},
		},
		{
			name:   "TestIteration2/TestGaugeHandlers/update",
			fields: fields{s: storage.NewStorage()},
			// content:  "text/plain",
			method:   http.MethodPost,
			request1: "/update/gauge/testGauge/100",
			request2: "/update/gauge/testGauge/100.1",
			want: want{
				statusCode: http.StatusOK,
				body:       "",
				value1:     float64(100),
				value2:     float64(100.1),
			},
		},
		{
			name:   "TestIteration2/TestGaugeHandlers/without_id",
			fields: fields{s: storage.NewStorage()},
			// content:  "text/plain",
			method:   http.MethodPost,
			request1: "/update/gauge/",
			request2: "",
			want: want{
				statusCode: http.StatusNotFound,
				body:       "{\"message\":\"Not Found\"}\n",
				value1:     "",
				value2:     "",
			},
		},
		{
			name:   "TestIteration2/TestGaugeHandlers/invalid_value",
			fields: fields{s: storage.NewStorage()},
			// content:  "text/plain",
			method:   http.MethodPost,
			request1: "/update/gauge/testGauge/none",
			request2: "",
			want: want{
				statusCode: http.StatusBadRequest,
				body:       "неверное значение метрики",
				value1:     "",
				value2:     "",
			},
		},
		{
			name:   "TestIteration2/TestUnknownHandlers/update_invalid_type",
			fields: fields{s: storage.NewStorage()},
			// content:  "text/plain",
			method:   http.MethodPost,
			request1: "/update/unknown/testCounter/100",
			request2: "",
			want: want{
				statusCode: http.StatusNotImplemented,
				body:       "нет метрики такого типа",
				value1:     "",
				value2:     "",
			},
		},
		{
			name:   "TestIteration2/TestUnknownHandlers/update_invalid_method",
			fields: fields{s: storage.NewStorage()},
			// content:  "text/plain",
			method:   http.MethodGet,
			request1: "/updater/counter/testCounter/100",
			request2: "",
			want: want{
				statusCode: http.StatusNotFound,
				body:       "{\"message\":\"Not Found\"}\n",
				value1:     "",
				value2:     "",
			},
		},
		// {
		// 	name:     "Неправильный Content-Type",
		// 	fields:   fields{s: storage.NewStorage()},
		// 	content:  "",
		// 	method:   http.MethodPost,
		// 	request1: "/update/counter/PollCount/2",
		// 	request2: "/update/counter/PollCount/3",
		// 	want: want{
		// 		statusCode: http.StatusBadRequest,
		// 		body:       "Only text/plain content are allowed!\n",
		// 		value1:     nil,
		// 		value2:     nil,
		// 	},
		// },
		{
			name:     "Неправильный http метод",
			fields:   fields{s: storage.NewStorage()},
			method:   http.MethodGet,
			request1: "/update/counter/PollCount/2",
			request2: "/update/counter/PollCount/3",
			want: want{
				statusCode: http.StatusMethodNotAllowed,
				body:       "{\"message\":\"Method Not Allowed\"}\n",
				value1:     nil,
				value2:     nil,
			},
		},
		{
			name:     "Сохранение неправильного counter",
			fields:   fields{s: storage.NewStorage()},
			content:  "text/plain",
			method:   http.MethodPost,
			request1: "/update/counter/PollCount/2.2",
			request2: "/update/counter/PollCount/3.1",
			want: want{
				statusCode: http.StatusBadRequest,
				body:       "неверное значение метрики",
				value1:     "",
				value2:     "",
			},
		},
	}
	var rMetricURL = regexp.MustCompile(`^/update/(\w+)\/(\w+)\/(-?\S+)$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewEchoHandler(tt.fields.s)
			request := httptest.NewRequest(tt.method, tt.request1, nil)
			w := httptest.NewRecorder()
			handler.Echo().ServeHTTP(w, handler.Echo().NewContext(request, w).Request())
			result := w.Result()

			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			body, err := ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.body, string(body))

			if result.StatusCode != http.StatusOK {
				return
			}

			match := rMetricURL.FindStringSubmatch(tt.request1)
			require.Equal(t, len(match), 4)
			assert.Contains(t, tt.fields.s.List()["192.0.2.1"], fmt.Sprintf("%s - %s - %v", match[1], match[2], tt.want.value1))
			fmt.Println(tt.name, tt.fields.s.List())

			request = httptest.NewRequest(tt.method, tt.request2, nil)
			request.Header.Add("Content-Type", tt.content)
			w = httptest.NewRecorder()
			handler.Echo().ServeHTTP(w, request)

			result = w.Result()

			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			body, err = ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.body, string(body))

			if result.StatusCode == http.StatusOK {
				match := rMetricURL.FindStringSubmatch(tt.request1)
				assert.Equal(t, len(match), 4)
				assert.Contains(t, tt.fields.s.List()["192.0.2.1"], fmt.Sprintf("%s - %s - %v", match[1], match[2], tt.want.value2))
				fmt.Println(tt.name, tt.fields.s.List())
			}
		})
	}
}

func TestEchoHandlerJSON(t *testing.T) {
	type fields struct {
		s repositories.Repository
	}
	type want struct {
		statusCode1 int
		statusCode2 int
		resp1       interface{}
		resp2       interface{}
	}
	tests := []struct {
		name     string
		fields   fields
		content  string
		request1 string
		request2 string
		body1    string
		body2    string
		method   string
		want     want
	}{
		{
			name:     "TestIteration2/TestCounterHandlersJSON",
			fields:   fields{s: storage.NewStorage()},
			content:  "application/json",
			method:   http.MethodPost,
			request1: "/update/",
			body1:    `{"id":"PollCount","type":"counter","delta":1}`,
			request2: "/value/",
			body2:    `{"id":"PollCount","type":"counter"}`,
			want: want{
				statusCode1: http.StatusOK,
				resp1:       ``,
				statusCode2: http.StatusOK,
				resp2:       `{"id":"PollCount","type":"counter","delta":1}` + "\n",
			},
		},
		{
			name:     "TestIteration2/TestGaugeHandlersJSON",
			fields:   fields{s: storage.NewStorage()},
			content:  "application/json",
			method:   http.MethodPost,
			request1: "/update/",
			body1:    `{"id":"RandomValue","type":"gauge","value":1.1}`,
			request2: "/value/",
			body2:    `{"id":"RandomValue","type":"gauge"}`,
			want: want{
				statusCode1: http.StatusOK,
				resp1:       ``,
				statusCode2: http.StatusOK,
				resp2:       `{"id":"RandomValue","type":"gauge","value":1.1}` + "\n",
			},
		},
		// {
		// 	name: "TestIteration2/TestGaugeHandlersJSONupdate",
		// 	// fields: fields{s: &storage.Storage{
		// 	// 	v: map[metrics.MetricType]map[string]map[string]interface{}{
		// 	// 		metrics.GaugeType: map[string]map[string]interface{}{
		// 	// 			"192.0.2.1": map[string]interface{}{
		// 	// 				"RandomValue": metrics.RandomValue{v: 2.2},
		// 	// 			},
		// 	// 		},
		// 	// 	},
		// 	// }},
		// 	fields:   fields{s: storage.NewStorage()},
		// 	content:  "application/json",
		// 	method:   http.MethodPost,
		// 	request1: "/update/",
		// 	body1:    `{"id":"RandomValue","type":"gauge","value":2.2}`,
		// 	request2: "/value/",
		// 	body2:    `{"id":"RandomValue","type":"gauge"}`,
		// 	want: want{
		// 		statusCode1: http.StatusOK,
		// 		resp1:       ``,
		// 		statusCode2: http.StatusOK,
		// 		resp2:       `{"id":"RandomValue","type":"gauge","value":2.2}`,
		// 	},
		// },
	}
	loger := logrus.New()
	loger.SetReportCaller(true)
	loger.SetLevel(logrus.DebugLevel)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewEchoHandler(tt.fields.s)
			handler.SetLoger(loger)

			buf := bytes.NewBufferString(tt.body1)
			request := httptest.NewRequest(tt.method, tt.request1, buf)
			request.Header.Add("Content-Type", tt.content)
			w := httptest.NewRecorder()
			handler.Echo().ServeHTTP(w, handler.Echo().NewContext(request, w).Request())
			result := w.Result()

			assert.Equal(t, tt.want.statusCode1, result.StatusCode)
			body, err := ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.resp1, string(body))

			if result.StatusCode != http.StatusOK {
				return
			}

			// match := rMetricURL.FindStringSubmatch(tt.request1)
			// require.Equal(t, len(match), 4)
			// assert.Contains(t, tt.fields.s.List()["192.0.2.1"], fmt.Sprintf("%s - %s - %v", match[1], match[2], tt.want.value1))
			// fmt.Println(tt.name, tt.fields.s.List())
			buf = bytes.NewBufferString(tt.body2)
			request = httptest.NewRequest(tt.method, tt.request2, buf)
			request.Header.Add("Content-Type", tt.content)
			w = httptest.NewRecorder()
			handler.Echo().ServeHTTP(w, request)

			result = w.Result()

			require.Equal(t, tt.want.statusCode2, result.StatusCode)
			body, err = ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.resp2, string(body))

			if result.StatusCode == http.StatusOK {
				// match := rMetricURL.FindStringSubmatch(tt.request1)
				// assert.Equal(t, len(match), 4)
				// assert.Contains(t, tt.fields.s.List()["192.0.2.1"], fmt.Sprintf("%s - %s - %v", match[1], match[2], tt.want.value2))
				fmt.Println(tt.name, tt.fields.s.List())
			}
		})
	}
}
