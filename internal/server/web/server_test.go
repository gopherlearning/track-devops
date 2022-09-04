package web

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			s, err := NewEchoServer(newStorage(t), WithKey([]byte("test")), WithPprof(true))
			require.NoError(t, err)
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
