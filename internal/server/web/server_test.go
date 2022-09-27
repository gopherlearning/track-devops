package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

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

func TestServer(t *testing.T) {
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
			// resp, err = http.Post("http://127.0.0.1:31329/update/", "application/json", bytes.NewBufferString(`{"value": 123}`))
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
			wg.Add(2)
			time.AfterFunc(500*time.Millisecond, func() {
				t.Run("Test Stop()", func(t *testing.T) {
					defer wg.Done()
					conn, err := net.DialTimeout("tcp", tt.listen, time.Second)
					assert.NoError(t, err)
					assert.NotNil(t, conn)
					assert.NoError(t, s.Stop())
				})
			})
			t.Run(fmt.Sprintf("Test Start(%s)", tt.listen), func(t *testing.T) {
				defer wg.Done()
				err := s.Start(tt.listen)
				require.NoError(t, err)
			})

		})
	}

}
