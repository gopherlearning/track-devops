package agent

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gopherlearning/track-devops/internal"
	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/gopherlearning/track-devops/internal/server/rpc"
	"github.com/gopherlearning/track-devops/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

// Client клиент с шифрованием запросов
type Client struct {
	transport     string
	selfAddress   string
	serverAddress string
	conn          grpc.ClientConnInterface
	grpcopts      []grpc.DialOption
	http          *http.Client
	key           *rsa.PublicKey
}

var emulatedError string

// emulateError используется для Эмуляции ошибок в тесте, для тех функций, в которых невозможно замокать интерфейс
func emulateError(err error, pos int) error {
	if err != nil {
		return err
	}
	if len(emulatedError) != 0 && strings.Contains(emulatedError, fmt.Sprint(pos)) {
		return errors.New(emulatedError)
	}
	return nil
}

var ErrMetricsCountIsNull = errors.New("metric count is 0")
var ErrStreamError = errors.New("stream error")

// MonitoringClient returns grpc client interface
func (c *Client) MonitoringClient() proto.MonitoringClient { return proto.NewMonitoringClient(c.conn) }

// Type returns type of client
func (c *Client) Type() string { return c.transport }

// SendMetrics ...
func (c *Client) SendMetrics(ctx context.Context, metrics []metrics.Metrics) error {
	if len(metrics) == 0 {
		return ErrMetricsCountIsNull
	}
	resp := make([]*proto.Metric, 0)
	for _, m := range metrics {
		msg := convertToProto(m)
		if msg == nil {
			return repositories.ErrWrongMetricType
		}
		resp = append(resp, msg)
	}
	_, err := c.MonitoringClient().Update(ctx, &proto.UpdateRequest{Metrics: resp})
	if err != nil {
		return err
	}
	return nil
}

// Do для клиента
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// здесь имитация установки собственного адреса,
	// для реальной установки адреса отправки можно было бы реализовать функцию
	// Dial() для транспорта http клиента
	req.Header.Add("X-Real-IP", c.selfAddress)
	if req.Method != http.MethodPost || c.key == nil {
		return c.http.Do(req)
	}
	hash := sha512.New()
	encrypted := make([]byte, 0)
	bufEncrypted := bytes.NewBuffer(encrypted)
	for {
		b := make([]byte, 0)
		buf := bytes.NewBuffer(b)
		v, _ := io.CopyN(buf, req.Body, int64(c.key.Size()-2*hash.Size()-2))
		if v == 0 {
			break
		}
		ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, c.key, buf.Bytes(), nil)
		if err != nil {
			zap.L().Info(err.Error())
			return nil, err
		}
		_, err = bufEncrypted.Write(ciphertext)
		if err = emulateError(err, 1); err != nil {
			zap.L().Info(err.Error())
			return nil, err
		}
	}
	req.ContentLength = int64(bufEncrypted.Len())
	req.Body = io.NopCloser(bufEncrypted)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type ClientOpt func(c *Client)

func WithGRPCOpts(opts ...grpc.DialOption) func(c *Client) {
	return func(c *Client) {
		c.grpcopts = opts
	}
}

// NewClient конструктор для клиента
func NewClient(ctx context.Context, args *internal.AgentArgs, opts ...ClientOpt) (*Client, error) {
	c := &Client{
		transport:     args.Transport,
		selfAddress:   args.SelfAddress,
		serverAddress: args.ServerAddr,
		grpcopts:      []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.DefaultConfig})},
	}
	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("option error: %v", opt)
		}
		opt(c)
	}
	switch args.Transport {
	case "http":
		c.http = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxConnsPerHost:     10,
				MaxIdleConnsPerHost: 10,
			},
		}
	case "grpc":
		conn, err := grpc.Dial(c.serverAddress, c.grpcopts...)
		if err != nil {
			return nil, err
		}
		go func() {
			<-ctx.Done()
			conn.Close()
		}()
		c.conn = conn
	default:
		return nil, fmt.Errorf("транспорт не поддерживается %s", args.Transport)
	}

	if len(args.CryptoKey) == 0 {
		return c, nil
	}
	keyPEM, err := os.ReadFile(args.CryptoKey)
	if err != nil {
		return nil, err
	}
	cryptoKey, _ := pem.Decode(keyPEM)
	if cryptoKey == nil {
		return nil, errors.New("bad PEM signature")
	}
	pubKey, err := x509.ParsePKCS1PublicKey(cryptoKey.Bytes)
	if err != nil {
		return nil, err
	}
	c.key = pubKey
	return c, nil
}

func convertToProto(m metrics.Metrics) *proto.Metric {
	metric := &proto.Metric{Id: m.ID, Hash: m.Hash, Type: rpc.GetMetricProtoType(&m)}
	switch metric.Type {
	case proto.Type_COUNTER:
		metric.Value = &proto.Metric_Counter{Counter: *m.Delta}
	case proto.Type_GAUGE:
		metric.Value = &proto.Metric_Gauge{Gauge: *m.Value}
	default:
		return nil
	}
	return metric
}
