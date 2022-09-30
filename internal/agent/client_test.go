package agent

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gopherlearning/track-devops/internal"
	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/gopherlearning/track-devops/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type mockMonitoringServer struct {
	proto.UnimplementedMonitoringServer
}

// var _ proto.MonitoringServer = (*mockMonitoringServer)(nil)

func (*mockMonitoringServer) Update(ctx context.Context, req *proto.UpdateRequest) (*proto.Empty, error) {
	for _, v := range req.Metrics {
		if v.Type == proto.Type_COUNTER && v.GetCounter() == 1 {
			return &proto.Empty{}, nil
		}
		if v.Type == proto.Type_GAUGE && v.GetGauge() == float64(1) {
			return &proto.Empty{}, nil
		}

	}
	return nil, status.Error(codes.InvalidArgument, repositories.ErrWrongMetricType.Error())
}

func dialer() func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)

	server := grpc.NewServer()

	proto.RegisterMonitoringServer(server, &mockMonitoringServer{})

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestMonitoringClientUpdate(t *testing.T) {
	ctx := context.TODO()
	t.Run("grpc dial error", func(t *testing.T) {
		_, err := NewClient(ctx, &internal.AgentArgs{Transport: "grpc", ServerAddr: "bla://bla"}, WithGRPCOpts(grpc.WithBlock()))
		assert.Error(t, err)
	})
	t.Run("nil option", func(t *testing.T) {
		_, err := NewClient(ctx, &internal.AgentArgs{Transport: "grpc", ServerAddr: "bla://bla"}, nil)
		assert.ErrorContains(t, err, "option error")
	})
	t.Run("bad key path", func(t *testing.T) {
		_, err := NewClient(ctx, &internal.AgentArgs{Transport: "grpc", ServerAddr: "bla://bla", CryptoKey: "12345"})
		assert.ErrorContains(t, err, "no such file or directory")
	})
	t.Run("bad key pem data", func(t *testing.T) {
		f, err := os.CreateTemp(os.TempDir(), "test")
		require.NoError(t, err)
		f.Close()
		require.NoError(t, os.WriteFile(f.Name(), []byte("bla bla bla"), 0644))
		defer os.Remove(f.Name())
		_, err = NewClient(ctx, &internal.AgentArgs{Transport: "grpc", ServerAddr: "bla://bla", CryptoKey: f.Name()})
		assert.ErrorContains(t, err, "bad PEM signature")
	})
	t.Run("bad key public key", func(t *testing.T) {
		f, err := os.CreateTemp(os.TempDir(), "test")
		require.NoError(t, err)
		f.Close()
		require.NoError(t, os.WriteFile(f.Name(), testPrivKey, 0644))
		defer os.Remove(f.Name())
		_, err = NewClient(ctx, &internal.AgentArgs{Transport: "grpc", ServerAddr: "bla://bla", CryptoKey: f.Name()})
		assert.ErrorContains(t, err, "structure error")
	})
	t.Run("good key public key", func(t *testing.T) {
		f, err := os.CreateTemp(os.TempDir(), "test")
		require.NoError(t, err)
		f.Close()
		require.NoError(t, os.WriteFile(f.Name(), testPubKey, 0644))
		defer os.Remove(f.Name())
		_, err = NewClient(ctx, &internal.AgentArgs{Transport: "grpc", ServerAddr: "bla://bla", CryptoKey: f.Name()})
		assert.NoError(t, err)
	})
	client, err := NewClient(ctx, &internal.AgentArgs{Transport: "grpc"}, WithGRPCOpts(grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(dialer())))
	require.NoError(t, err)

	require.Equal(t, "grpc", client.Type())
	testsUpdates := []struct {
		name string
		req  []metrics.Metrics
		err  error
	}{
		{
			"metric count is 0",
			make([]metrics.Metrics, 0),
			ErrMetricsCountIsNull,
		},
		{
			"invalid metric value",
			make([]metrics.Metrics, 1),
			repositories.ErrWrongMetricType,
		},
		{
			"invalid metric type",
			[]metrics.Metrics{{}},
			repositories.ErrWrongMetricType,
		},
		{
			"invalid metric value",
			[]metrics.Metrics{
				{MType: metrics.CounterType, Delta: metrics.GetInt64Pointer(2)},
			},
			status.Error(codes.InvalidArgument, repositories.ErrWrongMetricType.Error()),
		},
		{
			"success counter",
			[]metrics.Metrics{
				{MType: metrics.CounterType, Delta: metrics.GetInt64Pointer(1)},
			},
			nil,
		},
		{
			"success gauge",
			[]metrics.Metrics{
				{MType: metrics.GaugeType, Value: metrics.GetFloat64Pointer(1)},
			},
			nil,
		},
	}

	for _, tt := range testsUpdates {
		t.Run(tt.name, func(t *testing.T) {
			err := client.SendMetrics(ctx, tt.req)
			if tt.err != nil {
				assert.ErrorIs(t, err, tt.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

}

func TestClientDo(t *testing.T) {
	require.Error(t, emulateError(errors.New(""), 0))
	t.Run("транспорт не поддерживается", func(t *testing.T) {
		_, err := NewClient(context.TODO(), &internal.AgentArgs{Transport: "wrong"})
		assert.ErrorContains(t, err, "транспорт не поддерживается")
	})
	f, err := os.CreateTemp(os.TempDir(), "test")
	require.NoError(t, err)
	f.Close()
	require.NoError(t, os.WriteFile(f.Name(), testPubKey, 0644))
	defer os.Remove(f.Name())
	t.Run("http.MethodGet", func(t *testing.T) {
		client, err := NewClient(context.TODO(), &internal.AgentArgs{Transport: "http", CryptoKey: f.Name()})
		require.NoError(t, err)
		require.NotNil(t, client)
		req, err := http.NewRequest(http.MethodGet, "", nil)
		require.NoError(t, err)
		_, err = client.Do(req)
		assert.ErrorContains(t, err, "unsupported protocol scheme")
	})
	t.Run("c.key == nil", func(t *testing.T) {
		client, err := NewClient(context.TODO(), &internal.AgentArgs{Transport: "http"})
		require.NoError(t, err)
		require.NotNil(t, client)
		req, err := http.NewRequest(http.MethodPost, "", nil)
		require.NoError(t, err)
		_, err = client.Do(req)
		assert.ErrorContains(t, err, "unsupported protocol scheme")
	})
	t.Run("success request encrypted", func(t *testing.T) {
		// generate a test server so we can capture and inspect the request
		testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			b, err := io.ReadAll(req.Body)
			if err != nil {
				res.WriteHeader(http.StatusBadRequest)
				res.Write([]byte("read body error"))
				return
			}
			res.WriteHeader(http.StatusOK)
			res.Write(b)
		}))
		client, err := NewClient(context.TODO(), &internal.AgentArgs{Transport: "http", CryptoKey: f.Name()})
		require.NoError(t, err)
		require.NotNil(t, client)
		req, err := http.NewRequest(http.MethodPost, testServer.URL, bytes.NewBufferString("test message"))
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		_, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
	})
	t.Run("success request encrypted with error", func(t *testing.T) {
		client, err := NewClient(context.TODO(), &internal.AgentArgs{Transport: "http", CryptoKey: f.Name()})
		require.NoError(t, err)
		require.NotNil(t, client)
		req, err := http.NewRequest(http.MethodPost, "", bytes.NewBufferString("test message"))
		require.NoError(t, err)
		_, err = client.Do(req)
		require.Error(t, err)
	})
	t.Run("error EncryptOAEP", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusOK)
		}))
		client, err := NewClient(context.TODO(), &internal.AgentArgs{Transport: "http", CryptoKey: f.Name()})
		require.NoError(t, err)
		require.NotNil(t, client)
		req, err := http.NewRequest(http.MethodPost, testServer.URL, bytes.NewBufferString("test message"))
		require.NoError(t, err)
		client.key.E = 1
		_, err = client.Do(req)
		require.Error(t, err)
	})
	t.Run("error encrypt bufer write", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusOK)
		}))
		emulatedError = "buffer wryte 1"
		defer func() { emulatedError = "" }()
		client, err := NewClient(context.TODO(), &internal.AgentArgs{Transport: "http", CryptoKey: f.Name()})
		require.NoError(t, err)
		require.NotNil(t, client)
		req, err := http.NewRequest(http.MethodPost, testServer.URL, bytes.NewBufferString("test message"))
		require.NoError(t, err)
		_, err = client.Do(req)
		require.ErrorContains(t, err, "buffer wryte")
	})
}

var testPrivKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`)
var testPubKey = []byte(`-----BEGIN RSA PUBLIC KEY-----
MIICCgKCAgEAwM5szSRWOMnOAqFUn6VsZfVUyOUMHZRmbO+Fhp24DTx65BhBSPIc
usZw8LuHShyUsYeOUrqfBnKxWUW/pnNLRinLHB8D4OFHDw4y+MPNRGEIxLID7FHA
xwdd01Mc4CUGempg80Hg/zY2x2rmqIaeJDoxv3jRPMpGT7Mrr/8UWW+aebqrZRhI
E0N4LkAMZxuP/Sc8eAMP+K6smUEaOVTJFXNVI/UubCFiG8bUeR7HSPy7qR72ExBn
cI8PbIHzpSnAtqbTO4ZzjSoAI8UiyptY9m39/bZTagm0mHTTV04c7xrcdPpwwMFW
rd/ddyx2Fau0MvVThPevJliGLIxXWgo3xE5cLi+hti+u5iVw1FgtBaEDJ34WYqbV
cLiO+6KBOYzgW8wumUQ8CNl14iN9JuXDEQ9v8bl/IKz8xYdMyMXBezy5EDnWj0CS
Xs4liwBXwoqVv8iW+6L7XWA9xjNinsS7N2ubo2FsqkCJTVZuSFrSqc2aQm0VLUKu
RlnI67qcysbcR939r/Vln96E88xHUCH76eVqcPPDYx8+UkTXjYp9l1bN8VOmp0zL
dWVS9oINKqq205Dpgef+q8W21G4/OebFyNkywptMzrxcrL5/SRT7KO+/feSivV7y
rN7w/yKNdTXqcS7Z92LriE7qWD7sAn4tsyp2M5lFy/CGT656wNpnDcsCAwEAAQ==
-----END RSA PUBLIC KEY-----
`)
