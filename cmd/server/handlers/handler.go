package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Handler interface {
	GetMetric(c echo.Context) error
	GetMetricJSON(c echo.Context) error
	UpdateMetric(c echo.Context) error
	UpdateMetricJSON(c echo.Context) error
	ListMetrics(c echo.Context) error
	Echo() *echo.Echo
	Loger() logrus.FieldLogger
	SetLoger(logrus.FieldLogger)
}
