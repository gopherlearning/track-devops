package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gopherlearning/track-devops/cmd/server/handlers"
	"github.com/labstack/echo/v4"
)

type EchoServer struct {
	serv *echo.Echo
}

func NewEchoServer(h handlers.Handler) Web {
	return &EchoServer{serv: h.Echo()}
}
func (s *EchoServer) Start(listen string) error {
	s.serv.Server.Addr = listen
	err := s.serv.Server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *EchoServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	return s.serv.Shutdown(ctx)
}
