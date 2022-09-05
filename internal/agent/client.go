package agent

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"os"

	"go.uber.org/zap"
)

type Sender interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client клиент с шифрованием запросов
type Client struct {
	client *http.Client
	key    *rsa.PublicKey
}

// Do для клиента
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodPost {
		return c.client.Do(req)
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
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// NewClient конструктор для клиента
func NewClient(keyPath string) (Sender, error) {
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxConnsPerHost:     10,
			MaxIdleConnsPerHost: 10,
		},
	}
	if len(keyPath) == 0 {
		return c, nil
	}
	keyPEM, err := os.ReadFile(keyPath)
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
	return &Client{
		client: c,
		key:    pubKey,
	}, nil
}