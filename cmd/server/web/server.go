package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gopherlearning/track-devops/cmd/server/handlers"
	"github.com/labstack/echo/v4"
)

type Server struct {
	serv *echo.Echo
}

func NewServer(listen string, h *handlers.ClassicHandler) *Server {
	e := echo.New()
	e.Server.Addr = listen
	e.POST("/update/:type/:name/:value", echo.WrapHandler(http.HandlerFunc(h.Update)))
	return &Server{serv: e}
}
func (s *Server) Start() error {
	err := s.serv.Server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	return s.serv.Shutdown(ctx)
}
