package web

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/gopherlearning/track-devops/internal/server/storage/local"
)

func TestEchoHandler_Get(t *testing.T) {
	tests := []struct {
		value   map[string]interface{}
		name    string
		request string
		target  string
		want    string
		status  int
	}{
		{
			name:    "Существующее значение",
			request: "/value/counter/PollCount",
			status:  http.StatusOK,
			value:   map[string]interface{}{"counter": int64(123), "gauge": float64(0)},
			target:  "192.0.2.1",
			want:    "123",
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
			s := newStorage(t)
			if len(tt.want) != 0 {
				m := strings.Split(tt.request, "/")
				require.NoError(t, s.UpdateMetric(context.TODO(), tt.target, metrics.Metrics{MType: metrics.MetricType(m[2]), ID: m[3], Delta: metrics.GetInt64Pointer(tt.value["counter"].(int64)), Value: metrics.GetFloat64Pointer(tt.value["gauge"].(float64))}))
			}
			handler, err := NewEchoServer(s, "", false)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodGet, tt.request, nil)
			w := httptest.NewRecorder()
			handler.e.ServeHTTP(w, handler.e.NewContext(request, w).Request())
			result := w.Result()

			assert.Equal(t, tt.status, result.StatusCode)
			body, err := io.ReadAll(result.Body)
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
		value1     interface{}
		value2     interface{}
		body       string
		statusCode int
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
			name:     "TestIteration2/TestCounterHandlers/invalid_value",
			fields:   fields{s: newStorage(t)},
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
			name:     "TestIteration2/TestCounterHandlers/update",
			fields:   fields{s: newStorage(t)},
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
			name:     "TestIteration2/TestCounterHandlers/without_id",
			fields:   fields{s: newStorage(t)},
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
			fields: fields{s: newStorage(t)},
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
			fields: fields{s: newStorage(t)},
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
			fields: fields{s: newStorage(t)},
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
			fields: fields{s: newStorage(t)},
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
			fields: fields{s: newStorage(t)},
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
		{
			name:     "Неправильный http метод",
			fields:   fields{s: newStorage(t)},
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
			fields:   fields{s: newStorage(t)},
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
			handler, err := NewEchoServer(tt.fields.s, "", false)
			require.NoError(t, err)
			request := httptest.NewRequest(tt.method, tt.request1, nil)
			w := httptest.NewRecorder()
			handler.e.ServeHTTP(w, handler.e.NewContext(request, w).Request())
			result := w.Result()

			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.body, string(body))

			if result.StatusCode != http.StatusOK {
				return
			}

			match := rMetricURL.FindStringSubmatch(tt.request1)
			require.Equal(t, len(match), 4)
			list, _ := tt.fields.s.List(context.TODO())
			assert.Contains(t, list["192.0.2.1"], fmt.Sprintf("%s - %s - %v", match[1], match[2], tt.want.value1))
			fmt.Println(tt.name, list)

			request = httptest.NewRequest(tt.method, tt.request2, nil)
			request.Header.Add("Content-Type", tt.content)
			w = httptest.NewRecorder()
			handler.e.ServeHTTP(w, request)

			result = w.Result()

			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			body, err = io.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.body, string(body))

			if result.StatusCode == http.StatusOK {
				match := rMetricURL.FindStringSubmatch(tt.request1)
				assert.Equal(t, len(match), 4)
				list, _ := tt.fields.s.List(context.TODO())
				assert.Contains(t, list["192.0.2.1"], fmt.Sprintf("%s - %s - %v", match[1], match[2], tt.want.value2))
				fmt.Println(tt.name, list)
			}
		})
	}
}
func newStorage(t *testing.T) *local.Storage {
	s, err := local.NewStorage(false, nil, zap.L())
	require.NoError(t, err)
	return s
}
func TestEchoHandlerJSON(t *testing.T) {
	type fields struct {
		s repositories.Repository
	}
	type want struct {
		resp1       interface{}
		resp2       interface{}
		statusCode1 int
		statusCode2 int
	}
	tests := []struct {
		name     string
		content  string
		request1 string
		request2 string
		body1    string
		body2    string
		method   string
		fields   fields
		want     want
	}{
		{
			name:     "TestIteration2/TestCounterHandlersJSON",
			fields:   fields{s: newStorage(t)},
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
			name:     "TestErrorMethod",
			fields:   fields{s: newStorage(t)},
			content:  "application/json",
			method:   http.MethodGet,
			request1: "/update/",
			body1:    ``,
			want: want{
				statusCode1: http.StatusMethodNotAllowed,
				resp1:       `{"message":"Method Not Allowed"}` + "\n",
			},
		},
		{
			name:     "TestIteration2/TestGaugeHandlersJSON",
			fields:   fields{s: newStorage(t)},
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
	}
	logger, _ := zap.NewDevelopment()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewEchoServer(tt.fields.s, "", false, WithLogger(logger))
			require.NoError(t, err)
			buf := bytes.NewBufferString(tt.body1)
			request := httptest.NewRequest(tt.method, tt.request1, buf)
			request.Header.Add("Content-Type", tt.content)
			w := httptest.NewRecorder()
			handler.e.ServeHTTP(w, handler.e.NewContext(request, w).Request())
			result := w.Result()

			assert.Equal(t, tt.want.statusCode1, result.StatusCode)
			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.resp1, string(body))

			if result.StatusCode != http.StatusOK {
				return
			}
			buf = bytes.NewBufferString(tt.body2)
			request = httptest.NewRequest(tt.method, tt.request2, buf)
			request.Header.Add("Content-Type", tt.content)
			w = httptest.NewRecorder()
			handler.e.ServeHTTP(w, request)

			result = w.Result()

			require.Equal(t, tt.want.statusCode2, result.StatusCode)
			body, err = io.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.resp2, string(body))

			if result.StatusCode == http.StatusOK {
				list, _ := tt.fields.s.List(context.TODO())
				fmt.Println(tt.name, list)
			}
		})
	}
}
