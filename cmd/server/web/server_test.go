package web

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gopherlearning/track-devops/cmd/server/handlers"
	"github.com/gopherlearning/track-devops/cmd/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStorage(t *testing.T) *storage.Storage {
	s, err := storage.NewStorage(false, nil)
	require.NoError(t, err)
	return s
}
func TestServer(t *testing.T) {
	tests := []struct {
		name   string
		listen string
	}{
		{
			name:   "Создание сервера",
			listen: "127.0.0.1:12328",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewEchoServer(handlers.NewEchoHandler(newStorage(t)))
			require.NotNil(t, s)
			wg := sync.WaitGroup{}
			wg.Add(2)
			time.AfterFunc(time.Second, func() {
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
