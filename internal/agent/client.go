package agent

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gopherlearning/track-devops/internal"
	"github.com/gopherlearning/track-devops/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client клиент с шифрованием запросов
type Client struct {
	transport     string
	selfAddress   string
	serverAddress string
	grpc          proto.MonitoringClient
	http          *http.Client
	key           *rsa.PublicKey
}

func (c *Client) Type() string                   { return c.transport }
func (c *Client) SendMetric(*proto.Metric) error { return nil }

func (c *Client) SendMetrics([]*proto.Metric) error { return nil }

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
		if err != nil {
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

// NewClient конструктор для клиента
func NewClient(ctx context.Context, args *internal.AgentArgs) (*Client, error) {
	c := &Client{
		transport:     args.Transport,
		selfAddress:   args.SelfAddress,
		serverAddress: args.ServerAddr,
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
		conn, err := grpc.Dial(c.selfAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		go func() {
			<-ctx.Done()
			conn.Close()
		}()
		c.grpc = proto.NewMonitoringClient(conn)
	default:
		return nil, fmt.Errorf("транспорт не поддерживаетсяЖ %s", args.Transport)
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
		return nil, err
	}
	pubKey, err := x509.ParsePKCS1PublicKey(cryptoKey.Bytes)
	if err != nil {
		return nil, err
	}
	c.key = pubKey
	return c, nil
}
