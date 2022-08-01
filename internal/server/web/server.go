package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
)

type echoServer struct {
	s     repositories.Repository
	e     *echo.Echo
	loger logrus.FieldLogger
	key   []byte
}

// echoServerOptionFunc определяет тип функции для опций.
type echoServerOptionFunc func(*echoServer)

// WithKey задаёт ключ для подписи
func WithKey(key []byte) echoServerOptionFunc {
	return func(c *echoServer) {
		c.key = key
	}
}

// WithLoger set loger
func WithLoger(loger logrus.FieldLogger) echoServerOptionFunc {
	return func(c *echoServer) {
		c.loger = loger
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
func NewEchoServer(s repositories.Repository, opts ...echoServerOptionFunc) *echoServer {
	e := echo.New()
	// e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	serv := &echoServer{s: s, e: e, loger: logrus.StandardLogger()}
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

// GetMetric ...
func (h *echoServer) GetMetric(c echo.Context) error {
	if v, _ := h.s.GetMetric(c.RealIP(), c.Param("type"), c.Param("name")); v != nil {
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
	list, err := h.s.List()
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
		return c.NoContent(http.StatusNotFound)
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
	if err := h.s.UpdateMetric(context.Background(), c.RealIP(), m); err != nil {
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
		return c.NoContent(http.StatusNotFound)
	}
	if c.Request().Header["Content-Type"][0] != "application/json" {
		return c.String(http.StatusBadRequest, "only application/json content are allowed!")
	}
	decoder := json.NewDecoder(c.Request().Body)
	defer c.Request().Body.Close()
	mm := []metrics.Metrics{}
	err := decoder.Decode(&mm)
	if err != nil {
		h.loger.Error(err)
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

	if err := h.s.UpdateMetric(context.Background(), c.RealIP(), mm...); err != nil {
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
		return c.NoContent(http.StatusNotFound)
	}
	if c.Request().Header["Content-Type"][0] != "application/json" {
		return c.String(http.StatusBadRequest, "only application/json content are allowed!")
	}
	decoder := json.NewDecoder(c.Request().Body)
	defer c.Request().Body.Close()
	m := metrics.Metrics{}
	err := decoder.Decode(&m)
	if err != nil {
		h.loger.Error(err)
		return c.String(http.StatusBadRequest, err.Error())
	}
	if len(h.key) != 0 {
		recived := m.Hash
		err = m.Sign(h.key)
		if err != nil || recived != m.Hash {
			return c.HTML(http.StatusBadRequest, "подпись не соответствует ожиданиям")
		}
	}

	if err := h.s.UpdateMetric(context.Background(), c.RealIP(), m); err != nil {
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
		return c.NoContent(http.StatusNotFound)
	}
	if c.Request().Header["Content-Type"][0] != "application/json" {
		return c.String(http.StatusBadRequest, "only application/json content are allowed!")
	}
	decoder := json.NewDecoder(c.Request().Body)
	defer c.Request().Body.Close()
	m := metrics.Metrics{}
	err := decoder.Decode(&m)
	if err != nil {
		h.loger.Error(err)
		return c.String(http.StatusBadRequest, err.Error())
	}
	if v, _ := h.s.GetMetric(c.RealIP(), m.MType, m.ID); v != nil {

		if len(h.key) != 0 {
			err = v.Sign(h.key)
			if err != nil {
				h.loger.Error(err)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	return h.e.Shutdown(ctx)
}
