package web

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	case "NotFound":
		return nil, repositories.ErrWrongMetricID
	case "success":
		return &metrics.Metrics{ID: "success", MType: mType, Delta: metrics.GetInt64Pointer(1)}, nil
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
			req:    httptest.NewRequest("POST", "/update/"+string(metrics.CounterType)+"/internal/1", nil),
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
			resp.Result().Body.Close()
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
		{
			req:     httptest.NewRequest("POST", "/updates/", bytes.NewBufferString(`[{"id":"success","type":"counter","delta":1}]`)),
			content: "application/json",
			status:  http.StatusOK,
			s:       s,
			err:     "",
		},
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
			resp.Result().Body.Close()
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
		{
			req:     httptest.NewRequest("POST", "/update/", bytes.NewBufferString(`{"id":"success","type":"counter","delta":1}`)),
			content: "application/json",
			status:  http.StatusOK,
			s:       s,
			err:     "",
		},
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
			resp.Result().Body.Close()
		})
	}

}

func Test_echoServer_GetMetricJSON(t *testing.T) {
	store := &failStore{}
	s, err := NewEchoServer(store, "", false)
	require.NoError(t, err)
	ctx := s.e.NewContext(httptest.NewRequest("GET", "/value/", nil), httptest.NewRecorder())
	assert.ErrorContains(t, s.GetMetricJSON(ctx), "method not allowed")

	tests := []struct {
		req     *http.Request
		content string
		status  int
		s       *echoServer
		err     string
	}{
		{
			req:     httptest.NewRequest("POST", "/value/", nil),
			content: "application/text",
			status:  http.StatusBadRequest,
			s:       s,
			err:     "only application/json content are allowed!",
		},
		{
			req:     httptest.NewRequest("POST", "/value/", bytes.NewBufferString("12333ddddd")),
			content: "application/json",
			status:  http.StatusBadRequest,
			s:       s,
			err:     "json: cannot unmarshal number ",
		},

		{
			req:     httptest.NewRequest("POST", "/value/", bytes.NewBufferString(`{"id":"NotFound","type":"counter"}`)),
			content: "application/json",
			status:  http.StatusNotFound,
			s:       s,
			err:     "",
		},

		{
			req:     httptest.NewRequest("POST", "/value/", bytes.NewBufferString(`{"id":"success","type":"counter"}`)),
			content: "application/json",
			status:  http.StatusOK,
			s:       s,
			err:     "",
		},
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

func Test_echoServer_checkTrusted(t *testing.T) {
	_, network, _ := net.ParseCIDR("10.10.10.0/24")
	tests := []struct {
		name    string
		trusted *net.IPNet
		realip  string
	}{
		{
			name:    "success",
			realip:  "10.10.10.1",
			trusted: network,
		},
		{
			name:    "access denied, no header",
			realip:  "",
			trusted: network,
		},
		{
			name:    "access denied, bad ip",
			realip:  "256.1.1.1",
			trusted: network,
		},
		{
			name:    "access denied",
			realip:  "1.1.1.1",
			trusted: network,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &echoServer{
				trusted: tt.trusted,
				s:       newStorage(t),
			}
			e := echo.New()
			e.Use(h.checkTrusted)
			e.GET("/", func(c echo.Context) error {
				fmt.Println("I'm next")
				return c.NoContent(http.StatusOK)
			})
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Add("X-Real-IP", tt.realip)
			resp := httptest.NewRecorder()
			e.ServeHTTP(resp, req)
			b, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if !strings.Contains(tt.name, "success") {
				assert.Equal(t, string(b), tt.name)
			}
			resp.Result().Body.Close()
		})
	}
}

func Test_echoServer_cryptoMiddleware(t *testing.T) {
	b, err := base64.RawStdEncoding.DecodeString(`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`)
	require.NoError(t, err)
	encrypted := bytes.NewBuffer(b)
	tmp, err := os.CreateTemp(os.TempDir(), "go_test")
	require.NoError(t, err)
	require.FileExists(t, tmp.Name())
	_, err = tmp.WriteString(testPem)
	require.NoError(t, err)

	defer os.Remove(tmp.Name())
	tests := []struct {
		name string
		body io.Reader
	}{
		{
			name: "success with nil",
			body: nil,
		},
		{
			name: "decryption error",
			body: bytes.NewBufferString("clear text"),
		},
		{
			name: "success",
			body: encrypted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &echoServer{
				s: newStorage(t),
				e: echo.New(),
			}

			WithCryptoKey(tmp.Name())(h)
			h.e.GET("/", func(c echo.Context) error {
				fmt.Println("I'm next")
				return c.NoContent(http.StatusOK)
			})
			req := httptest.NewRequest("GET", "/", tt.body)
			resp := httptest.NewRecorder()
			h.e.ServeHTTP(resp, req)
			b, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if !strings.Contains(tt.name, "success") {
				assert.Contains(t, string(b), tt.name)
			}
			resp.Result().Body.Close()
		})
	}
}

var testPem = `-----BEGIN RSA PRIVATE KEY-----
MIIJKAIBAAKCAgEA0K5yFNBFej4HyCLOF7hHYB98ZywAZkSgWJ+gP0fqN+orJ6cO
PakpDY9204hZJ9jIJ+mOI/LKeWlcFA0y6ouN7+p5SL73Yfla+oafIwVjdapCysSi
80+OkeJAG/MtIT3wldx8NUG0OUQ7D73RRNeQhjyzTn29el8I1oKfGBzOzdSp5iMf
vzcJebRlUmip9qYOcwa1mY8gZ6sakEWGJeRLcFia7pj0cThVg1yTr6eha2uOmmCj
CHXIyeVRO4kP4v70OgcPK1hujAdZvNlujPpoxJ5UhDLO2jD3vtH6rA/ltN0C0FhQ
XvZVJh6DWrz5wuz7mf5W9zV8thyTTkq57/JtSQex+hQcxcWoVXQeKap3gkqQMnN3
gv2E08UlyKijmhypCG0rk/49oCke0bVBx9c3ct7ud0Iyg31Mnr6nJs+Pk+RZS32t
jAWbKPerkRyoT/igccuDpHyzmzrQo5Kdth9gFu71Oez3qyHbzYAPn4xDced5aGEy
jEJlvfdp6BwXYLGbY3fUZdB++VfnM+S7CqTIMX0qCOCJpvs3uR7mIPuoIJUusRhl
VV7NDqBocOg1I94/u+6QiVCAE4a34o8JdbW/sYhx7WVRSX3x9I9VEGaA2XpyLL6o
6IQ5vjSEK3GK9vOa9wo+NV+gSSumhMV+f5uFeSvwsSZ6yZKE/enMd8bZ6wMCAwEA
AQKCAgBz4OqodB6giuF3WRxoP4Qi9Fj4UY7HO0Ru8fTiLuT4t6fPFQzrYIvTY97w
B766Sb6bqy2q+J9GXCMtX0buxx/CIcnHg4OLfBPxGeA0akGYjTsZraduxLa+e4xt
+NMOqn52OUEfsaSKSEEjtJwIRkuSvxIye9BDq5IUy+PcV+LemUDe8pImdEFmu62n
3UbEF+HeLdOZucicyH6vrmuXjvX1JL6jz3utg0K49ydrWwJfzBIb152wjPc6ZYR+
MtYHjtu/fTwHLcv1Jf/GxlaFImgbBnCYGD6VGqv34lH4Sbucez2cw+2dTdxqlncK
Y/WtMDtmf716+NCPr758szIc89mKuyajCDtzS5BJWAWTHiKADX7W/diXtCOPPCSm
UPV12O72kKHj/wZTxSD/Nj+u5iedTS+Xy/V1h0yvuYmzBIPgs81Oh2Fuqmzg3Sba
r2VmtfFgk3xKuqiHZloFd3sCrm9CmuoF+lhNiCtqsQx759cK4EjLR0pguGpVPZQv
mWasvSxjQOZkgVYJh2vU73tCVUjwAHtoVm4WIuAEnOtmkx9ob6VKAfHraUSrwMKi
Lat40wr7Jtk/8cxGANlF7L1VDDGb+OESdXyRoIHM1OwrYUtCisIBy0gf4JvyaxaB
KxkkZvVuxeuDMBjb0CWDDXW0VI/xtCBXSoWB+kcm4CMj/PmgwQKCAQEA8mi9r609
nEsKWXAEp8hJPt0psA5ATYxEvpcpM6fxEOmVxYak4T8+sNeFdBp/sEWbqjjZJQ6a
w3q+DqaUAoniL8RmJCiQN6+3t/kpAOJVdFHd4hFiTmIZsM6UTXBYe1xDSFaaR69O
8tiz6100fbq9or8QoWFQ/srjMjK6sxpXnGTJJZaoif3xjTWuxN2iBl+8WxCRv9QS
YfTqQ1j9/nXfGJaRGzodM7dvO7sXyzS9DCH/5bIJpDFW7mwUtzoPAlc6VN5By/RX
N2SnlQW6TqFgUKtNQvKz6LXly+/TwYrimky3uu9xawumOGau2JAH29qQJRb1y4qN
ZUab56qe6q89lQKCAQEA3GGdwgiT/IsftpKafSh0TfGxuj24U9LxSSkjx2amTLaA
5UNuZ0MElwcrOcn/AiaMCySr7sm+iqUWuZ2NPfnlUBYlTXoGBP0z1lmaqCAXBnkT
n9v1JXXKJKUQjdOIroZhSFMYRVgP3rMOIBlBjQWeNoU/1NMlEleCMUFwghmGNr70
8eKItyahf+vKmibHqlDIzkfGWgJr539rxuXcCQdKx2az8beXMjsfm7n9UyfcnfpL
E1yfOPfZCNp4/hx8o0Qql5mrGkxoTqGw/UkWaBNyB7KjSXGVLA+5+E1ZMhgrTwzt
PFOY71/vQOar+pntr/4noih+jpDtj35qvHK+PZjwNwKCAQAZaS1pOwnYVm1xTrLO
O8qh0mFKWVQYTPnv2Lyy84nrsfDHUgP6sLyLoSwWLajw+3sD7w2kOtGyaC2AL6oY
Ugfp5fanF7F2hO8HVBEeTJuUo/hUeGoLuXDj/ePB8mL0G4naDWoC1be68Uh7Bbw4
6dhzNQAzSpZI/0+ttW+o0rwYYuBLFm34eSxXFyeI74rKjEKccTI2H68Fobzk7nFB
uW13kGEJr5/cCgCZDFXEMXUXzoCavX0RPzLTr3TEeEuWfTpaJypSjPyPi/edQYp0
L6p/ClYBDJwbauX56NwTz9FNR3mDGRKUnBYCl5EAlqicPV8a5DtD6PRFh49US6h5
BYG5AoIBAQCJJq8ZXGFM4ABijSZcEdsfzvT+pP6cHEFReKrto7KHN2VMSQTietDW
dP2vv0hWvEqXfMELoL5WZpuX9Lc8BNNzXfTlHLW2USX7llQroZzyyFMwP6F3KLEe
0SNWQHlls/fDHQOT1FQ8Ek8OJummrAJkh9TLzIPbwF4j7Ufpj5z5YSnrh0HySbZk
eAfkm5HTKudtiTmmNq+UqLYYWGDxtXoSUpZWLh2Ig0cOkVdYcwxXvLcQW6ozt/t4
CQ8Xhf8DVJ71LgtQGJEprnMJjnzFVKS4qbH5ORjPDRJ9txV1mZkKX08dJiGdh6TM
TUJmeXl098UOpAjvDL7reI9QrFA84XtNAoIBAFFhN/2Ff21HiuH3zski+ohwYBCX
F4M+P/zUiiARPsGz4q/Gs632wlervvdMA8EfHu9IpiZUA+C5k+NXmCC5keHwa7QR
c+8NhI7i1XmtExWNkejPO9spag+XotHQT8b5OAoG/pcJWaLb70O1nA8KCr5E0nFS
no+iwbvhgddVE4LzO3MjOjtvwpThIowAePKp6TdW72ZeNgC7+4F9S8vQ4h4+ynQ7
VeZgsZPfhNZHc7phxiYKsmDeEX0l4z2FtP0wveHMfT4s59rKKsDPidfT+Ca9b/Rk
T5FAGLOlsNsXT4ecd9qma2g7pj898EfwyY0U6FMvQTgBbfUHqfqc3xYL9uI=
-----END RSA PRIVATE KEY-----
`
var testPup = `-----BEGIN RSA PUBLIC KEY-----
MIICCgKCAgEA0K5yFNBFej4HyCLOF7hHYB98ZywAZkSgWJ+gP0fqN+orJ6cOPakp
DY9204hZJ9jIJ+mOI/LKeWlcFA0y6ouN7+p5SL73Yfla+oafIwVjdapCysSi80+O
keJAG/MtIT3wldx8NUG0OUQ7D73RRNeQhjyzTn29el8I1oKfGBzOzdSp5iMfvzcJ
ebRlUmip9qYOcwa1mY8gZ6sakEWGJeRLcFia7pj0cThVg1yTr6eha2uOmmCjCHXI
yeVRO4kP4v70OgcPK1hujAdZvNlujPpoxJ5UhDLO2jD3vtH6rA/ltN0C0FhQXvZV
Jh6DWrz5wuz7mf5W9zV8thyTTkq57/JtSQex+hQcxcWoVXQeKap3gkqQMnN3gv2E
08UlyKijmhypCG0rk/49oCke0bVBx9c3ct7ud0Iyg31Mnr6nJs+Pk+RZS32tjAWb
KPerkRyoT/igccuDpHyzmzrQo5Kdth9gFu71Oez3qyHbzYAPn4xDced5aGEyjEJl
vfdp6BwXYLGbY3fUZdB++VfnM+S7CqTIMX0qCOCJpvs3uR7mIPuoIJUusRhlVV7N
DqBocOg1I94/u+6QiVCAE4a34o8JdbW/sYhx7WVRSX3x9I9VEGaA2XpyLL6o6IQ5
vjSEK3GK9vOa9wo+NV+gSSumhMV+f5uFeSvwsSZ6yZKE/enMd8bZ6wMCAwEAAQ==
-----END RSA PUBLIC KEY-----`
