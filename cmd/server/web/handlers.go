package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/labstack/echo/v4"
)

// GetMetric ...
func (h *EchoServer) GetMetric(c echo.Context) error {
	if v := h.s.GetMetric(c.RealIP(), c.Param("type"), c.Param("name")); v != nil {
		return c.HTML(http.StatusOK, v.String())
	}
	return c.NoContent(http.StatusNotFound)
}

func (h *EchoServer) Ping(c echo.Context) error {
	if err := h.s.Ping(c.Request().Context()); err != nil {
		return c.HTML(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

// ListMetrics ...
func (h *EchoServer) ListMetrics(c echo.Context) error {
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
func (h *EchoServer) UpdateMetric(c echo.Context) error {
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
		case repositories.ErrWrongValueInStorage:
			return c.HTML(http.StatusNotImplemented, err.Error())
		default:
			return c.HTML(http.StatusInternalServerError, err.Error())
		}
	}
	return c.NoContent(http.StatusOK)
}

// UpdatesMetricJSON ...
func (h *EchoServer) UpdatesMetricJSON(c echo.Context) error {
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

	if err := h.s.UpdateMetric(c.RealIP(), mm...); err != nil {
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
func (h *EchoServer) UpdateMetricJSON(c echo.Context) error {
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

	if err := h.s.UpdateMetric(c.RealIP(), m); err != nil {
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
func (h *EchoServer) GetMetricJSON(c echo.Context) error {
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
	if v := h.s.GetMetric(c.RealIP(), m.MType, m.ID); v != nil {

		if len(h.key) != 0 {
			err = v.Sign(h.key)
			if err != nil {
				h.loger.Error(err)
				return c.String(http.StatusBadRequest, err.Error())
			}
		}
		return c.JSON(http.StatusOK, v)
	}
	h.loger.Warn(m)
	return c.NoContent(http.StatusNotFound)
}
