package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env/v6"
	"github.com/gopherlearning/track-devops/cmd/server/handlers"
	"github.com/gopherlearning/track-devops/cmd/server/storage"
	"github.com/gopherlearning/track-devops/cmd/server/web"
	"github.com/sirupsen/logrus"
)

var args struct {
	Config         string        `help:"Config"`
	ServerAddr     string        `help:"Server address" name:"address" env:"ADDRESS" default:"127.0.0.1:8080"`
	PollInterval   time.Duration `help:"Poll interval" env:"POLL_INTERVAL" default:"2s"`
	ReportInterval time.Duration `help:"Report interval" env:"REPORT_INTERVAL" default:"10s"`
	Format         string        `help:"Report format" env:"FORMAT"`
}

func main() {
	err := env.Parse(&args)
	if err != nil {
		logrus.Fatal(err)
	}
	kong.Parse(&args)
	store := storage.NewStorage()
	h := handlers.NewEchoHandler(store)
	h.SetLoger(logrus.StandardLogger())
	s := web.NewEchoServer(h)

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	// Периодический вывод содержимого хранилища
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			<-ticker.C

			fmt.Println("==============================")
			for target, values := range store.List() {
				fmt.Printf(`Target "%s":%s`, target, "\n")
				for _, v := range values {
					fmt.Printf("\t%s\n", v)
				}
			}
		}
	}()
	go func() {
		err := s.Start(args.ServerAddr)
		if err != nil {
			fmt.Println(222, err)
		}
	}()
	sig := <-terminate
	err = s.Stop()
	if err != nil {
		fmt.Println(222, err)
	}
	fmt.Printf("Server stoped by signal \"%v\"\n", sig)
}
