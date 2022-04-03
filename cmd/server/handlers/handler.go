package handlers

import "github.com/labstack/echo/v4"

type Handler interface {
	GetMetric(c echo.Context) error
	UpdateMetric(c echo.Context) error
	ListMetrics(c echo.Context) error
	Echo() *echo.Echo
}
