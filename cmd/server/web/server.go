package web

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

type EchoServer struct {
	s     repositories.Repository
	e     *echo.Echo
	loger logrus.FieldLogger
	key   []byte
}

// EchoServerOptionFunc определяет тип функции для опций.
type EchoServerOptionFunc func(*EchoServer)

// WithKey задаёт ключ для подписи
func WithKey(key []byte) EchoServerOptionFunc {
	return func(c *EchoServer) {
		c.key = key
	}
}

func WithLoger(loger logrus.FieldLogger) EchoServerOptionFunc {
	return func(c *EchoServer) {
		c.loger = loger
	}
}

// WithProf задаёт ключ для подписи
func WithPprof(usePprof bool) EchoServerOptionFunc {
	return func(c *EchoServer) {
		if usePprof {
			pprof.Register(c.e)
		}
	}
}
func NewEchoServer(s repositories.Repository, opts ...EchoServerOptionFunc) *EchoServer {
	e := echo.New()
	// e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	serv := &EchoServer{s: s, e: e, loger: logrus.StandardLogger()}
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
		opt(serv)
	}
	return serv
}
func (h *EchoServer) Start(listen string) error {
	h.e.Server.Addr = listen
	err := h.e.Server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (h *EchoServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	return h.e.Shutdown(ctx)
}
