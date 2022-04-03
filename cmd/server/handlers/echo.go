package handlers

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Handler ...
type EchoHandler struct {
	s repositories.Repository
	e *echo.Echo
}

// NewHandler создаёт новый экземпляр обработчика запросов, привязанный к хранилищу
func NewEchoHandler(s repositories.Repository) Handler {
	e := echo.New()
	// e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	h := &EchoHandler{s: s, e: e}
	h.e.POST("/update/:type/:name/:value", h.UpdateMetric)
	h.e.GET("/value/:type/:name", h.GetMetric)
	h.e.GET("/", h.ListMetrics)
	return h
}

func (h *EchoHandler) Echo() *echo.Echo { return h.e }

func (h *EchoHandler) GetMetric(c echo.Context) error {
	if v := h.s.Get(c.RealIP(), c.Param("type"), c.Param("name")); len(v) != 0 {
		return c.HTML(http.StatusOK, v)
	}
	return c.NoContent(http.StatusNotFound)
}

func (h *EchoHandler) ListMetrics(c echo.Context) error {
	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	for target, values := range h.s.List() {
		fmt.Fprintf(buf, `Target "%s":%s`, target, "\n")
		for _, v := range values {
			fmt.Fprintf(buf, "\t%s\n", v)
		}
	}
	return c.HTMLBlob(http.StatusOK, buf.Bytes())
}

func (h *EchoHandler) UpdateMetric(c echo.Context) error {
	if c.Request().Method != http.MethodPost {
		return c.NoContent(http.StatusNotFound)
	}
	// if r.Header.Get("Content-Type") != "text/plain" {
	// 	http.Error(w, "Only text/plain content are allowed!", http.StatusBadRequest)
	// 	return
	// }

	if err := h.s.Update(c.RealIP(), c.Param("type"), c.Param("name"), c.Param("value")); err != nil {
		switch err {
		case repositories.ErrWrongMetricURL:
			return c.HTML(http.StatusNotFound, err.Error())
		case repositories.ErrWrongMetricValue:
			return c.HTML(http.StatusBadRequest, err.Error())
		case repositories.ErrWrongMetricType:
			return c.HTML(http.StatusNotImplemented, err.Error())
		case repositories.ErrWrongValueInStorage:
			return c.HTML(http.StatusNotImplemented, err.Error())
		default:
			return c.HTML(http.StatusInternalServerError, err.Error())
		}
	}
	return c.NoContent(http.StatusOK)
}
