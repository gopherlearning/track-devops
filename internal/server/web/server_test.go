package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testPem = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`
var testPup = `-----BEGIN RSA PUBLIC KEY-----
MIICCgKCAgEAvB5OF+O38HG/t2A/LapE/smRXpTSXjg3dnrmMunl4k7CX8NYeMDX
00aOLlEBIcqG+QiZXyPW5l00i3dorb0Q70yGSR4oQCIN4aBqwpRArFMOqQhiRIou
zXW0eJBb+DIg6M66fifuSKB3O/esna+lOk3qkYSc+8D39/8cD12ezlHthLZ+3KQ4
Y0q9qcko7HFizHmvLrI3q9MWyPTzyA9kClnFUKXhi33xczpNBPMZJoZMj3Zt5hNM
vhtK6HtKgn2/K17DXhiQWz5L8HJOHHxmwb/7kZXFJEgkFABFGpfNgA7NZkJCUMzj
UkthFU1d5UlSjfJttRrPrAMGjwQjmXVHbRFH+Ue5d3kGGKtrPIyBitHpusKsT/hd
6RwaZVeK5VmiF8IVTyzRqcetjcWN7j1jGzluQo3UVUEuE9oXwPVcDFF2KmBaAsrX
Erf9AX683aSw7TIgi9u5ZSI8DEcBeaQekkgVKxYjPOdvhJsrRjkCGKQebpt+0xsC
wgJw6r1m+eFi0mvID7HO1SmCKmD24TyU3A9Oa88XuBm6hvcQgbG+dLIGfom9k7KI
HW9pP86STu9lJKUECLOr6gqvQ9ChfO/72GsjfezOn/cySgeaT2jb/CqXHHz6Lqnm
HzDjUWT05lVaNnKLUTwxvBerQCt8TGLS7j7D7qgc+ksGk3noBOJdeT0CAwEAAQ==
-----END RSA PUBLIC KEY-----`

type failStore struct{}

func (s *failStore) List(ctx context.Context) (map[string][]string, error) {
	return nil, errors.New("test error")
}
func (s *failStore) GetMetric(ctx context.Context, target string, mType metrics.MetricType, name string) (*metrics.Metrics, error) {
	switch name {
	case "ErrWrongMetricURL":
		return nil, repositories.ErrWrongMetricURL
	case "ErrWrongMetricValue":
		return nil, repositories.ErrWrongMetricValue
	case "ErrWrongValueInStorage":
		return nil, repositories.ErrWrongValueInStorage
	case "success":
		return &metrics.Metrics{}, nil
	default:
		return nil, errors.New("Internal")
	}
}

func (s *failStore) UpdateMetric(ctx context.Context, target string, mm ...metrics.Metrics) error {
	switch mm[0].ID {
	case "ErrWrongMetricURL":
		return repositories.ErrWrongMetricURL
	case "ErrWrongMetricValue":
		return repositories.ErrWrongMetricValue
	case "ErrWrongValueInStorage":
		return repositories.ErrWrongValueInStorage
	case "success":
		return nil
	default:
		return errors.New("Internal")
	}
}

func (s *failStore) Metrics(ctx context.Context, target string) (map[string][]metrics.Metrics, error) {
	panic("not implemented") // TODO: Implement
}

func (s *failStore) Ping(_ context.Context) error {
	panic("not implemented") // TODO: Implement
}

func TestServer(t *testing.T) {
	t.Run("List storage", func(t *testing.T) {
		store := newStorage(t)
		s, err := NewEchoServer(store, "", false, WithKey([]byte("test")), WithPprof(true))
		require.NoError(t, err)
		s.s.UpdateMetric(context.TODO(), "127.0.0.1", metrics.Metrics{ID: "test", MType: metrics.CounterType, Delta: metrics.GetInt64Pointer(1)})
		require.NoError(t, s.ListMetrics(s.e.NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())))
		require.NoError(t, s.ListMetrics(s.e.NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())))
		s.s = &failStore{}
		require.ErrorContains(t, s.ListMetrics(s.e.NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())), "test error")
	})
	t.Run("With trusted subnet", func(t *testing.T) {
		store := newStorage(t)
		_, err := NewEchoServer(store, "", false, WithTrustedSubnet("10.0.0.0/24"))
		require.NoError(t, err)
		_, err = NewEchoServer(store, "", false, WithTrustedSubnet(""))
		require.NoError(t, err)
		_, err = NewEchoServer(store, "", false, WithTrustedSubnet("1.1.1.1.1"))
		require.NoError(t, err)
	})

	tests := []struct {
		name   string
		listen string
	}{
		{
			name:   "Создание сервера",
			listen: "127.0.0.1:31328",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newStorage(t)
			pemFile, err := os.CreateTemp(os.TempDir(), "testGo")
			require.NoError(t, err)
			require.NoError(t, pemFile.Close())
			defer os.Remove(pemFile.Name())
			pupFile, err := os.CreateTemp(os.TempDir(), "testGo")
			require.NoError(t, err)
			require.NoError(t, pupFile.Close())
			defer os.Remove(pupFile.Name())
			_, err = NewEchoServer(store, "", false, WithKey([]byte("test")), WithPprof(true), WithCryptoKey("bla"))
			require.Error(t, err)
			_, err = NewEchoServer(store, "", false, WithKey([]byte("test")), WithPprof(true), WithCryptoKey(pemFile.Name()))
			require.Error(t, err)
			require.NoError(t, os.WriteFile(pemFile.Name(), []byte(testPup), 0700))
			_, err = NewEchoServer(store, "", false, WithKey([]byte("test")), WithPprof(true), WithCryptoKey(pemFile.Name()))
			require.Error(t, err)
			require.NoError(t, os.WriteFile(pemFile.Name(), []byte(testPem), 0700))
			s, err := NewEchoServer(store, "", false, WithKey([]byte("test")), WithPprof(true), WithCryptoKey(pemFile.Name()))
			require.NoError(t, err)
			// go s.Start("127.0.0.1:31329")
			// resp, err := http.Get("http://127.0.0.1:31329/ping")
			// require.NoError(t, err)
			// require.Equal(t, resp.StatusCode, http.StatusOK)
			// require.NoError(t, resp.Body.Close())
			// resp, err = http.Post("http://127.0.0.1:31329/value", "aplication/html", nil)
			// require.NoError(t, err)
			// require.NoError(t, resp.Body.Close())
			// resp, err = http.Post("http://127.0.0.1:31329/update/", "application/json", bytes.NewBufferString(`{"value":123}`))
			// require.NoError(t, err)
			// require.Equal(t, resp.StatusCode, http.StatusNotAcceptable)
			// require.NoError(t, resp.Body.Close())
			// s.Stop()
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/ping", nil)
			require.NoError(t, err)
			s.Ping(s.e.NewContext(req, rr))
			require.Equal(t, rr.Code, http.StatusOK)
			////
			s, err = NewEchoServer(store, ":6655", false, WithKey([]byte("test")), WithPprof(true), WithCryptoKey(""))
			require.NoError(t, err)
			s, err = NewEchoServer(store, ":1", false, WithKey([]byte("test")), WithPprof(true), WithCryptoKey(""))
			require.NoError(t, err)
			s, err = NewEchoServer(store, "", false, WithKey([]byte("test")), WithPprof(true), WithCryptoKey(""))
			require.NoError(t, err)
			req, err = http.NewRequest("GET", "/ping", nil)
			require.NoError(t, err)
			rr = httptest.NewRecorder()
			s.Ping(s.e.NewContext(req, rr))
			require.Equal(t, rr.Code, http.StatusOK)
			rr = httptest.NewRecorder()
			store.PingError = true
			s.Ping(s.e.NewContext(req, rr))
			require.Equal(t, rr.Code, http.StatusInternalServerError)
			store.PingError = false
			require.NoError(t, s.s.Ping(context.TODO()))
			require.NotNil(t, s)
			wg := sync.WaitGroup{}
			wg.Add(3)
			time.AfterFunc(500*time.Millisecond, func() {
				t.Run("Test Stop()", func(t *testing.T) {
					defer wg.Done()
					conn, err := net.DialTimeout("tcp", tt.listen, time.Second)
					assert.NoError(t, err)
					assert.NotNil(t, conn)
					s.Stop()
				})
			})
			t.Run(fmt.Sprintf("Test Start(%s)", tt.listen), func(t *testing.T) {
				defer wg.Done()
				go s.Start(tt.listen)
			})
			t.Run(fmt.Sprintf("Test Start(%s) again", tt.listen), func(t *testing.T) {
				defer wg.Done()
				err := s.Start(tt.listen)
				require.NoError(t, err)
			})
			wg.Wait()
		})
	}

}

func Test_echoServer_UpdateMetric(t *testing.T) {
	store := &failStore{}
	s, err := NewEchoServer(store, "", false, WithKey([]byte("test")), WithPprof(true))
	require.NoError(t, err)

	ctx := s.e.NewContext(httptest.NewRequest("GET", "/update/bal/bla/bla", nil), httptest.NewRecorder())
	assert.ErrorContains(t, s.UpdateMetric(ctx), "method not allowed")

	tests := []struct {
		req    *http.Request
		status int
		s      *echoServer
	}{
		{
			req:    httptest.NewRequest("POST", "/update/"+string(metrics.CounterType)+"/ErrWrongMetricURL/1", nil),
			status: http.StatusNotFound,
			s:      s,
		},
		{
			req:    httptest.NewRequest("POST", "/update/"+string(metrics.CounterType)+"/ErrWrongMetricValue/1", nil),
			status: http.StatusBadRequest,
			s:      s,
		},
		{
			req:    httptest.NewRequest("POST", "/update/"+string(metrics.CounterType)+"/ErrWrongValueInStorage/1", nil),
			status: http.StatusNotImplemented,
			s:      s,
		},
		{
			req:    httptest.NewRequest("POST", "/update/"+string(metrics.CounterType)+"/blabla/1", nil),
			status: http.StatusInternalServerError,
			s:      s,
		},
		{
			req:    httptest.NewRequest("POST", "/update/"+string(metrics.CounterType)+"/success/1", nil),
			status: http.StatusOK,
			s:      s,
		},
		// {
		// 	req: ,
		// 	uri: ,
		// 	status: ,
		// 	s: s,
		// },

	}
	for _, v := range tests {
		t.Run(v.req.RequestURI, func(t *testing.T) {
			resp := httptest.NewRecorder()
			s.e.ServeHTTP(resp, v.req)
			assert.Equal(t, resp.Result().StatusCode, v.status)
			fmt.Println(resp.Result().Status)
		})
	}

}

func Test_echoServer_UpdatesMetricJSON(t *testing.T) {
	store := &failStore{}
	s, err := NewEchoServer(store, "", false)
	require.NoError(t, err)
	sHash, err := NewEchoServer(store, "", false, WithKey([]byte("bhygyg")))
	require.NoError(t, err)

	ctx := s.e.NewContext(httptest.NewRequest("GET", "/updates/", nil), httptest.NewRecorder())
	assert.ErrorContains(t, s.UpdatesMetricJSON(ctx), "method not allowed")

	tests := []struct {
		req     *http.Request
		content string
		status  int
		s       *echoServer
		err     string
	}{
		{
			req:     httptest.NewRequest("POST", "/updates/", nil),
			content: "application/text",
			status:  http.StatusBadRequest,
			s:       s,
			err:     "only application/json content are allowed!",
		},
		{
			req:     httptest.NewRequest("POST", "/updates/", bytes.NewBufferString("12333ddddd")),
			content: "application/json",
			status:  http.StatusBadRequest,
			s:       s,
			err:     "json: cannot unmarshal number ",
		},
		{
			req:     httptest.NewRequest("POST", "/updates/", bytes.NewBufferString(`[{"id":"Alloc","type":"gauge","hash":"5c6ea3afd33a3f42c74e64da490c55ac299955e31a6b77d62e9ada33d1112a1f","value":819632}]`)),
			content: "application/json",
			status:  http.StatusBadRequest,
			s:       sHash,
			err:     "подпись не соответствует ожиданиям",
		},
		{
			req:     httptest.NewRequest("POST", "/updates/", bytes.NewBufferString(`[{"id":"ErrWrongMetricURL","type":"counter","delta":1}]`)),
			content: "application/json",
			status:  http.StatusNotFound,
			s:       s,
			err:     repositories.ErrWrongMetricURL.Error(),
		},
		{
			req:     httptest.NewRequest("POST", "/updates/", bytes.NewBufferString(`[{"id":"ErrWrongMetricValue","type":"counter","delta":1}]`)),
			content: "application/json",
			status:  http.StatusBadRequest,
			s:       s,
			err:     repositories.ErrWrongMetricValue.Error(),
		},
		{
			req:     httptest.NewRequest("POST", "/updates/", bytes.NewBufferString(`[{"id":"ErrWrongValueInStorage","type":"counter","delta":1}]`)),
			content: "application/json",
			status:  http.StatusNotImplemented,
			s:       s,
			err:     repositories.ErrWrongValueInStorage.Error(),
		},
		{
			req:     httptest.NewRequest("POST", "/updates/", bytes.NewBufferString(`[{"id":"bla","type":"counter","delta":1}]`)),
			content: "application/json",
			status:  http.StatusInternalServerError,
			s:       s,
			err:     "Internal",
		},
		// metrics.Metrics

		{
			req:     httptest.NewRequest("POST", "/updates/", bytes.NewBufferString(`[{"id":"success","type":"counter","delta":1}]`)),
			content: "application/json",
			status:  http.StatusOK,
			s:       s,
			err:     "",
		},
		// {
		// 	req: ,
		// 	uri: ,
		// 	status: ,
		// 	s: s,
		// },

	}
	for _, v := range tests {
		t.Run(v.err, func(t *testing.T) {
			resp := httptest.NewRecorder()
			v.req.Header.Add("Content-Type", v.content)
			v.s.e.ServeHTTP(resp, v.req)
			assert.Equal(t, resp.Result().StatusCode, v.status)
			if len(v.err) != 0 {
				b, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(b), v.err)
			}
		})
	}

}

func Test_echoServer_UpdateMetricJSON(t *testing.T) {
	store := &failStore{}
	s, err := NewEchoServer(store, "", false)
	require.NoError(t, err)
	sHash, err := NewEchoServer(store, "", false, WithKey([]byte("bhygyg")), WithPprof(true))
	require.NoError(t, err)

	ctx := s.e.NewContext(httptest.NewRequest("GET", "/update/", nil), httptest.NewRecorder())
	assert.ErrorContains(t, s.UpdateMetricJSON(ctx), "method not allowed")

	tests := []struct {
		req     *http.Request
		content string
		status  int
		s       *echoServer
		err     string
	}{
		{
			req:     httptest.NewRequest("POST", "/update/", nil),
			content: "application/text",
			status:  http.StatusBadRequest,
			s:       s,
			err:     "only application/json content are allowed!",
		},
		{
			req:     httptest.NewRequest("POST", "/update/", bytes.NewBufferString("12333ddddd")),
			content: "application/json",
			status:  http.StatusBadRequest,
			s:       s,
			err:     "json: cannot unmarshal number ",
		},
		{
			req:     httptest.NewRequest("POST", "/update/", bytes.NewBufferString(`{"id":"d","type":"counter","delta":1}`)),
			content: "application/json",
			status:  http.StatusBadRequest,
			s:       sHash,
			err:     "подпись не соответствует ожиданиям",
		},
		{
			req:     httptest.NewRequest("POST", "/update/", bytes.NewBufferString(`{"id":"ErrWrongMetricURL","type":"counter","delta":1}`)),
			content: "application/json",
			status:  http.StatusNotFound,
			s:       s,
			err:     repositories.ErrWrongMetricURL.Error(),
		},
		{
			req:     httptest.NewRequest("POST", "/update/", bytes.NewBufferString(`{"id":"ErrWrongMetricValue","type":"counter","delta":1}`)),
			content: "application/json",
			status:  http.StatusBadRequest,
			s:       s,
			err:     repositories.ErrWrongMetricValue.Error(),
		},
		{
			req:     httptest.NewRequest("POST", "/update/", bytes.NewBufferString(`{"id":"ErrWrongValueInStorage","type":"counter","delta":1}`)),
			content: "application/json",
			status:  http.StatusNotImplemented,
			s:       s,
			err:     repositories.ErrWrongValueInStorage.Error(),
		},
		{
			req:     httptest.NewRequest("POST", "/update/", bytes.NewBufferString(`{"id":"bla","type":"counter","delta":1}`)),
			content: "application/json",
			status:  http.StatusInternalServerError,
			s:       s,
			err:     "Internal",
		},
		// metrics.Metrics

		{
			req:     httptest.NewRequest("POST", "/update/", bytes.NewBufferString(`{"id":"success","type":"counter","delta":1}`)),
			content: "application/json",
			status:  http.StatusOK,
			s:       s,
			err:     "",
		},
		// {
		// 	req: ,
		// 	uri: ,
		// 	status: ,
		// 	s: s,
		// },

	}
	for _, v := range tests {
		t.Run(v.err, func(t *testing.T) {
			resp := httptest.NewRecorder()
			v.req.Header.Add("Content-Type", v.content)
			v.s.e.ServeHTTP(resp, v.req)
			assert.Equal(t, resp.Result().StatusCode, v.status)
			if len(v.err) != 0 {
				b, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(b), v.err)
			}
		})
	}

}

// func Test_echoServer_GetMetricJSON(t *testing.T) {
// 	store := &failStore{}
// 	s, err := NewEchoServer(store, "", false)
// 	require.NoError(t, err)
// 	sHash, err := NewEchoServer(store, "", false, WithKey([]byte("bhygyg")), WithPprof(true))
// 	require.NoError(t, err)

// 	ctx := s.e.NewContext(httptest.NewRequest("GET", "/value/", nil), httptest.NewRecorder())
// 	assert.ErrorContains(t, s.GetMetricJSON(ctx), "method not allowed")

// 	tests := []struct {
// 		req     *http.Request
// 		content string
// 		status  int
// 		s       *echoServer
// 		err     string
// 	}{
// 		{
// 			req:     httptest.NewRequest("POST", "/value/", nil),
// 			content: "application/text",
// 			status:  http.StatusBadRequest,
// 			s:       s,
// 			err:     "only application/json content are allowed!",
// 		},
// 		{
// 			req:     httptest.NewRequest("POST", "/value/", bytes.NewBufferString("12333ddddd")),
// 			content: "application/json",
// 			status:  http.StatusBadRequest,
// 			s:       s,
// 			err:     "json: cannot unmarshal number ",
// 		},
// 		{
// 			req:     httptest.NewRequest("POST", "/value/", bytes.NewBufferString(`{"id":"d","type":"counter"}`)),
// 			content: "application/json",
// 			status:  http.StatusBadRequest,
// 			s:       sHash,
// 			err:     "подпись не соответствует ожиданиям",
// 		},
// 		{
// 			req:     httptest.NewRequest("POST", "/value/", bytes.NewBufferString(`{"id":"ErrWrongMetricURL","type":"counter"}`)),
// 			content: "application/json",
// 			status:  http.StatusNotFound,
// 			s:       s,
// 			err:     repositories.ErrWrongMetricURL.Error(),
// 		},
// 		{
// 			req:     httptest.NewRequest("POST", "/value/", bytes.NewBufferString(`{"id":"ErrWrongMetricValue","type":"counter"}`)),
// 			content: "application/json",
// 			status:  http.StatusBadRequest,
// 			s:       s,
// 			err:     repositories.ErrWrongMetricValue.Error(),
// 		},
// 		{
// 			req:     httptest.NewRequest("POST", "/value/", bytes.NewBufferString(`{"id":"ErrWrongValueInStorage","type":"counter"}`)),
// 			content: "application/json",
// 			status:  http.StatusNotImplemented,
// 			s:       s,
// 			err:     repositories.ErrWrongValueInStorage.Error(),
// 		},
// 		{
// 			req:     httptest.NewRequest("POST", "/value/", bytes.NewBufferString(`{"id":"bla","type":"counter"}`)),
// 			content: "application/json",
// 			status:  http.StatusInternalServerError,
// 			s:       s,
// 			err:     "Internal",
// 		},
// 		// metrics.Metrics

// 		{
// 			req:     httptest.NewRequest("POST", "/value/", bytes.NewBufferString(`{"id":"success","type":"counter"}`)),
// 			content: "application/json",
// 			status:  http.StatusOK,
// 			s:       s,
// 			err:     "",
// 		},
// 		// {
// 		// 	req: ,
// 		// 	uri: ,
// 		// 	status: ,
// 		// 	s: s,
// 		// },

// 	}
// 	for _, v := range tests {
// 		t.Run(v.err, func(t *testing.T) {
// 			resp := httptest.NewRecorder()
// 			v.req.Header.Add("Content-Type", v.content)
// 			v.s.e.ServeHTTP(resp, v.req)
// 			assert.Equal(t, resp.Result().StatusCode, v.status)
// 			if len(v.err) != 0 {
// 				b, err := io.ReadAll(resp.Body)
// 				require.NoError(t, err)
// 				assert.Contains(t, string(b), v.err)
// 			}
// 		})
// 	}

// }
