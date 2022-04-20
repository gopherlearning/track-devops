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
	Config        string        `help:"Config"`
	ServerAddr    string        `help:"Server address" name:"address" env:"ADDRESS" default:"127.0.0.1:8080"`
	StoreInterval time.Duration `help:"интервал времени в секундах, по истечении которого текущие показания сервера сбрасываются на диск (значение 0 — делает запись синхронной)" env:"STORE_INTERVAL" default:"300s"`
	StoreFile     string        `help:"строка, имя файла, где хранятся значения (пустое значение — отключает функцию записи на диск)" env:"STORE_FILE" default:"/tmp/devops-metrics-db.json"`
	Restore       bool          `help:"булево значение (true/false), определяющее, загружать или нет начальные значения из указанного файла при старте сервера" env:"RESTORE" default:"true"`
	Format        string        `help:"Report format" env:"FORMAT"`
}

func main() {
	err := env.Parse(&args)
	if err != nil {
		logrus.Fatal(err)
	}
	kong.Parse(&args)
	logrus.Info(args)
	store, err := storage.NewStorage(args.Restore, &args.StoreInterval, args.StoreFile)
	if err != nil {
		logrus.Error(err)
		return
	}
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
			logrus.Error(err)
		}
	}()
	sig := <-terminate
	err = s.Stop()
	if err != nil {
		logrus.Error(err)
	}
	err = store.Save()
	if err != nil {
		logrus.Error(err)
	}
	logrus.Info("Server stoped by signal \"%v\"\n", sig)
}
