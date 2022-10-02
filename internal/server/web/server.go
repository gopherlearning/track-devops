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
	"net"
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
	trusted    *net.IPNet
	s          repositories.Repository
	e          *echo.Echo
	logger     *zap.Logger
	key        []byte
	privateKey *rsa.PrivateKey
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
		c.privateKey = privKey
		c.e.Use(c.cryptoMiddleware)
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

// WithTrustedSubnet задаёт сеть доверенных адресов агентов
func WithTrustedSubnet(trusted string) echoServerOptionFunc {
	return func(c *echoServer) {
		if len(trusted) == 0 {
			return
		}
		_, trusted, err := net.ParseCIDR(trusted)
		if err != nil {
			if c.logger != nil {
				c.logger.Error(err.Error())
			}
			return
		}
		c.trusted = trusted
		c.e.Use(c.checkTrusted)
	}
}

// NewechoServer returns http server
func NewEchoServer(store repositories.Repository, listen string, debug bool, opts ...echoServerOptionFunc) (*echoServer, error) {
	e := echo.New()
	serv := &echoServer{s: store, e: e, logger: zap.L()}
	serv.e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			serv.logger.Info("request",
				zap.String("URI", v.URI),
				zap.Int("status", v.Status),
			)
			return nil
		},
	}))
	if !debug {
		serv.e.Use(middleware.Recover())
	}
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
	if len(listen) != 0 {
		go func() {
			err := serv.e.Start(listen)
			if err != nil && err != http.ErrServerClosed {
				serv.logger.Error(err.Error())
			}
		}()

	}
	return serv, nil
}

// GetMetric ...
func (h *echoServer) GetMetric(c echo.Context) error {
	if v, _ := h.s.GetMetric(c.Request().Context(), c.RealIP(), metrics.MetricType(c.Param("type")), c.Param("name")); v != nil {
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
		c.Response().Status = http.StatusMethodNotAllowed
		return errors.New("method not allowed")
	}
	m := metrics.Metrics{MType: metrics.MetricType(c.Param("type")), ID: c.Param("name")}
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
	if err := h.s.UpdateMetric(c.Request().Context(), c.RealIP(), m); err != nil {
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
		c.Response().Status = http.StatusMethodNotAllowed
		return errors.New("method not allowed")
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
		fmt.Println(h.key)
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
		c.Response().Status = http.StatusMethodNotAllowed
		return errors.New("method not allowed")
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
		c.Response().Status = http.StatusMethodNotAllowed
		return errors.New("method not allowed")
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

		// if len(h.key) != 0 {
		// 	err = v.Sign(h.key)
		// 	if err != nil {
		// 		h.logger.Error(err.Error())
		// 		return c.String(http.StatusBadRequest, err.Error())
		// 	}
		// }
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

// CheckTrusted проверяет вазрешён ли доступ клиенту на основе адреса
func (h *echoServer) checkTrusted(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		realIP := c.Request().Header.Get("X-Real-IP")
		if len(realIP) == 0 {
			return c.HTML(http.StatusForbidden, "access denied, no header")
		}
		ip := net.ParseIP(realIP)
		if ip == nil {
			return c.HTML(http.StatusForbidden, "access denied, bad ip")
		}
		if !h.trusted.Contains(ip) {
			return c.HTML(http.StatusForbidden, "access denied")
		}
		return next(c)
	}
}

func (h *echoServer) cryptoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		hash := sha512.New()
		r := c.Request()
		encrypted := make([]byte, 0)
		bufEncrypted := bytes.NewBuffer(encrypted)
		for {
			b := make([]byte, 0)
			buf := bytes.NewBuffer(b)
			v, _ := io.CopyN(buf, r.Body, int64(h.privateKey.PublicKey.Size()))
			if v == 0 {
				break
			}
			plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, h.privateKey, buf.Bytes(), nil)
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
}
