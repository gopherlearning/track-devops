package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
)

type echoServer struct {
	s      repositories.Repository
	e      *echo.Echo
	logger *zap.Logger
	key    []byte
}

// echoServerOptionFunc определяет тип функции для опций.
type echoServerOptionFunc func(*echoServer)

// WithKey задаёт ключ для подписи
func WithKey(key []byte) echoServerOptionFunc {
	return func(c *echoServer) {
		c.key = key
	}
}

// WithCryptoKey задаёт ключ шифрования соединения с агентом и задаёт middleware для дешифрования тела запроса
func WithCryptoKey(keyPath string) echoServerOptionFunc {
	if len(keyPath) == 0 {
		return func(c *echoServer) {}
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		zap.L().Error(err.Error())
		return nil
	}
	cryptoKey, rest := pem.Decode(keyPEM)
	if cryptoKey == nil {
		zap.L().Error("pem.Decode failed", zap.Any("error", rest))
		return nil
	}
	privKey, err := x509.ParsePKCS1PrivateKey(cryptoKey.Bytes)
	if err != nil {
		zap.L().Error("x509.ParsePKCS1PrivateKey", zap.Error(err))
		return nil
	}
	return func(c *echoServer) {
		c.e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				if c.Request().Method != echo.POST {
					return next(c)
				}
				if c.Request().Header.Get("Content-Type") != "application/json" {
					return next(c)
				}
				hash := sha512.New()
				r := c.Request()
				encrypted := make([]byte, 0)
				bufEncrypted := bytes.NewBuffer(encrypted)
				for {
					b := make([]byte, 0)
					buf := bytes.NewBuffer(b)
					v, _ := io.CopyN(buf, r.Body, int64(privKey.PublicKey.Size()))
					if v == 0 {
						break
					}
					plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, privKey, buf.Bytes(), nil)
					if err != nil {
						zap.L().Debug(err.Error())
						return c.HTML(http.StatusNotAcceptable, err.Error())
					}
					_, err = bufEncrypted.Write(plaintext)
					if err != nil {
						zap.L().Debug(err.Error())
						return c.HTML(http.StatusNotAcceptable, err.Error())
					}
				}
				r.ContentLength = int64(bufEncrypted.Len())
				r.Body = io.NopCloser(bufEncrypted)
				return next(c)
			}
		})
	}
}

// WithLogger set logger
func WithLogger(logger *zap.Logger) echoServerOptionFunc {
	return func(c *echoServer) {
		c.logger = logger
	}
}

// WithProf задаёт ключ для подписи
func WithPprof(usePprof bool) echoServerOptionFunc {
	return func(c *echoServer) {
		if usePprof {
			pprof.Register(c.e)
		}
	}
}

// NewechoServer returns http server
func NewEchoServer(s repositories.Repository, opts ...echoServerOptionFunc) (*echoServer, error) {
	e := echo.New()
	// e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	logger, _ := zap.NewDevelopment()
	serv := &echoServer{s: s, e: e, logger: logger}
	serv.e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	serv.e.POST("/update/", serv.UpdateMetricJSON)
	serv.e.POST("/updates/", serv.UpdatesMetricJSON)
	serv.e.POST("/value/", serv.GetMetricJSON)
	serv.e.POST("/update/:type/:name/:value", serv.UpdateMetric)
	serv.e.GET("/value/:type/:name", serv.GetMetric)
	serv.e.GET("/ping", serv.Ping)
	serv.e.GET("/", serv.ListMetrics)
	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("option error: %v", opt)
		}
		opt(serv)
	}
	return serv, nil
}

// GetMetric ...
func (h *echoServer) GetMetric(c echo.Context) error {
	if v, _ := h.s.GetMetric(c.Request().Context(), c.RealIP(), c.Param("type"), c.Param("name")); v != nil {
		return c.HTML(http.StatusOK, v.String())
	}
	return c.NoContent(http.StatusNotFound)
}

// Ping check storage connection
func (h *echoServer) Ping(c echo.Context) error {
	if err := h.s.Ping(c.Request().Context()); err != nil {
		return c.HTML(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

// ListMetrics ...
func (h *echoServer) ListMetrics(c echo.Context) error {
	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	list, err := h.s.List(c.Request().Context())
	if err != nil {
		return err
	}
	for target, values := range list {
		fmt.Fprintf(buf, `<b>Target "%s":</b></br>`, target)
		for _, v := range values {
			fmt.Fprintf(buf, "  %s<br>", v)
		}
	}
	return c.HTMLBlob(http.StatusOK, buf.Bytes())
}

// UpdateMetric ...
func (h *echoServer) UpdateMetric(c echo.Context) error {
	if c.Request().Method != http.MethodPost {
		return c.NoContent(http.StatusMethodNotAllowed)
	}
	m := metrics.Metrics{MType: c.Param("type"), ID: c.Param("name")}
	switch c.Param("type") {
	case string(metrics.CounterType):
		i, err := strconv.Atoi(c.Param("value"))
		if err != nil {
			return c.HTML(http.StatusBadRequest, repositories.ErrWrongMetricValue.Error())
		}
		m.Delta = metrics.GetInt64Pointer(int64(i))
	case string(metrics.GaugeType):
		i, err := strconv.ParseFloat(c.Param("value"), 64)
		if err != nil {
			return c.HTML(http.StatusBadRequest, repositories.ErrWrongMetricValue.Error())
		}
		m.Value = metrics.GetFloat64Pointer(float64(i))
	default:
		return c.HTML(http.StatusNotImplemented, repositories.ErrWrongMetricType.Error())
	}
	if err := h.s.UpdateMetric(context.TODO(), c.RealIP(), m); err != nil {
		switch err {
		case repositories.ErrWrongMetricURL:
			return c.HTML(http.StatusNotFound, err.Error())
		case repositories.ErrWrongMetricValue:
			return c.HTML(http.StatusBadRequest, err.Error())
		case repositories.ErrWrongValueInStorage:
			return c.HTML(http.StatusNotImplemented, err.Error())
		default:
			return c.HTML(http.StatusInternalServerError, err.Error())
		}
	}
	return c.NoContent(http.StatusOK)
}

// UpdatesMetricJSON ...
func (h *echoServer) UpdatesMetricJSON(c echo.Context) error {
	if c.Request().Method != http.MethodPost {
		return c.NoContent(http.StatusMethodNotAllowed)
	}
	if c.Request().Header["Content-Type"][0] != "application/json" {
		return c.String(http.StatusBadRequest, "only application/json content are allowed!")
	}
	decoder := json.NewDecoder(c.Request().Body)
	defer c.Request().Body.Close()
	mm := []metrics.Metrics{}
	err := decoder.Decode(&mm)
	if err != nil {
		h.logger.Error(err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}
	if len(h.key) != 0 {
		for _, v := range mm {
			recived := v.Hash
			err = v.Sign(h.key)
			if err != nil || recived != v.Hash {
				return c.HTML(http.StatusBadRequest, "подпись не соответствует ожиданиям")
			}
		}
	}

	if err := h.s.UpdateMetric(context.TODO(), c.RealIP(), mm...); err != nil {
		switch err {
		case repositories.ErrWrongMetricURL:
			return c.HTML(http.StatusNotFound, err.Error())
		case repositories.ErrWrongMetricValue:
			return c.HTML(http.StatusBadRequest, err.Error())
		case repositories.ErrWrongValueInStorage:
			return c.HTML(http.StatusNotImplemented, err.Error())
		default:
			return c.HTML(http.StatusInternalServerError, err.Error())
		}
	}
	return c.NoContent(http.StatusOK)
}

// UpdateMetricJSON ...
func (h *echoServer) UpdateMetricJSON(c echo.Context) error {
	if c.Request().Method != http.MethodPost {
		return c.NoContent(http.StatusMethodNotAllowed)
	}
	if c.Request().Header["Content-Type"][0] != "application/json" {
		return c.String(http.StatusBadRequest, "only application/json content are allowed!")
	}
	decoder := json.NewDecoder(c.Request().Body)
	defer c.Request().Body.Close()
	m := metrics.Metrics{}
	err := decoder.Decode(&m)
	if err != nil {
		h.logger.Error(err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}
	if len(h.key) != 0 {
		recived := m.Hash
		err = m.Sign(h.key)
		if err != nil || recived != m.Hash {
			return c.HTML(http.StatusBadRequest, "подпись не соответствует ожиданиям")
		}
	}

	if err := h.s.UpdateMetric(context.TODO(), c.RealIP(), m); err != nil {
		switch err {
		case repositories.ErrWrongMetricURL:
			return c.HTML(http.StatusNotFound, err.Error())
		case repositories.ErrWrongMetricValue:
			return c.HTML(http.StatusBadRequest, err.Error())
		case repositories.ErrWrongValueInStorage:
			return c.HTML(http.StatusNotImplemented, err.Error())
		default:
			return c.HTML(http.StatusInternalServerError, err.Error())
		}
	}
	return c.NoContent(http.StatusOK)
}

// GetMetricJSON ...
func (h *echoServer) GetMetricJSON(c echo.Context) error {
	if c.Request().Method != http.MethodPost {
		return c.NoContent(http.StatusMethodNotAllowed)
	}
	if c.Request().Header["Content-Type"][0] != "application/json" {
		return c.String(http.StatusBadRequest, "only application/json content are allowed!")
	}
	decoder := json.NewDecoder(c.Request().Body)
	defer c.Request().Body.Close()
	m := metrics.Metrics{}
	err := decoder.Decode(&m)
	if err != nil {
		h.logger.Error(err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}
	if v, _ := h.s.GetMetric(c.Request().Context(), c.RealIP(), m.MType, m.ID); v != nil {

		if len(h.key) != 0 {
			err = v.Sign(h.key)
			if err != nil {
				h.logger.Error(err.Error())
				return c.String(http.StatusBadRequest, err.Error())
			}
		}
		return c.JSON(http.StatusOK, v)
	}
	return c.NoContent(http.StatusNotFound)
}

// Start http server
func (h *echoServer) Start(listen string) error {
	h.e.Server.Addr = listen
	err := h.e.Server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Stop http server
func (h *echoServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	return h.e.Shutdown(ctx)
}
