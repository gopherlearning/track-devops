package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

// Handler ...
type EchoHandler struct {
	s     repositories.Repository
	e     *echo.Echo
	loger logrus.FieldLogger
	key   []byte
}

// NewHandler создаёт новый экземпляр обработчика запросов, привязанный к хранилищу
func NewEchoHandler(s repositories.Repository, key []byte) Handler {
	e := echo.New()
	// e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	h := &EchoHandler{s: s, e: e, key: key}
	h.e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	h.e.POST("/update/", h.UpdateMetricJSON)
	h.e.POST("/value/", h.GetMetricJSON)
	h.e.POST("/update/:type/:name/:value", h.UpdateMetric)
	h.e.GET("/value/:type/:name", h.GetMetric)
	h.e.GET("/ping", h.Ping)
	h.e.GET("/", h.ListMetrics)
	return h
}

// Echo ...
func (h *EchoHandler) Echo() *echo.Echo { return h.e }

// Loger ...
func (h *EchoHandler) Loger() logrus.FieldLogger { return h.loger }

// SetLoger ...
func (h *EchoHandler) SetLoger(l logrus.FieldLogger) { h.loger = l }

// GetMetric ...
func (h *EchoHandler) GetMetric(c echo.Context) error {
	if v := h.s.GetMetric(c.RealIP(), c.Param("type"), c.Param("name")); v != nil {
		return c.HTML(http.StatusOK, v.String())
	}
	return c.NoContent(http.StatusNotFound)
}

func (h *EchoHandler) Ping(c echo.Context) error {
	if err := h.s.Ping(c.Request().Context()); err != nil {
		return c.HTML(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

// ListMetrics ...
func (h *EchoHandler) ListMetrics(c echo.Context) error {
	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	for target, values := range h.s.List() {
		fmt.Fprintf(buf, `<b>Target "%s":</b></br>`, target)
		for _, v := range values {
			fmt.Fprintf(buf, "  %s<br>", v)
		}
	}
	return c.HTMLBlob(http.StatusOK, buf.Bytes())
}

// UpdateMetric ...
func (h *EchoHandler) UpdateMetric(c echo.Context) error {
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
	if err := h.s.UpdateMetric(c.RealIP(), m); err != nil {
		switch err {
		case repositories.ErrWrongMetricURL:
			return c.HTML(http.StatusNotFound, err.Error())
		case repositories.ErrWrongMetricValue:
			return c.HTML(http.StatusBadRequest, err.Error())
		// case repositories.ErrWrongMetricType:
		case repositories.ErrWrongValueInStorage:
			return c.HTML(http.StatusNotImplemented, err.Error())
		default:
			return c.HTML(http.StatusInternalServerError, err.Error())
		}
	}
	return c.NoContent(http.StatusOK)
}

// UpdateMetricJSON ...
func (h *EchoHandler) UpdateMetricJSON(c echo.Context) error {
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
		h.Loger().Error(err)
		return c.String(http.StatusBadRequest, err.Error())
	}
	if len(h.key) != 0 {
		recived := m.Hash
		err = m.Sign(h.key)
		if err != nil || recived != m.Hash {
			return c.HTML(http.StatusBadRequest, "подпись не соответствует ожиданиям")
		}
	}

	// h.Loger().Infof("%+v"q, &m)
	if err := h.s.UpdateMetric(c.RealIP(), m); err != nil {
		switch err {
		case repositories.ErrWrongMetricURL:
			return c.HTML(http.StatusNotFound, err.Error())
		case repositories.ErrWrongMetricValue:
			return c.HTML(http.StatusBadRequest, err.Error())
		// case repositories.ErrWrongMetricType:
		case repositories.ErrWrongValueInStorage:
			return c.HTML(http.StatusNotImplemented, err.Error())
		default:
			return c.HTML(http.StatusInternalServerError, err.Error())
		}
	}
	// return c.NoContent(http.StatusOK)
	return c.NoContent(http.StatusOK)
}

// GetMetricJSON ...
func (h *EchoHandler) GetMetricJSON(c echo.Context) error {
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
		h.Loger().Error(err)
		return c.String(http.StatusBadRequest, err.Error())
	}
	if v := h.s.GetMetric(c.RealIP(), m.MType, m.ID); v != nil {

		if len(h.key) != 0 {
			err = v.Sign(h.key)
			if err != nil {
				h.Loger().Error(err)
				return c.String(http.StatusBadRequest, err.Error())
			}
		}
		return c.JSON(http.StatusOK, v)
	}
	h.Loger().Warn(m)
	return c.NoContent(http.StatusNotFound)
}
