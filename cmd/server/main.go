package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env/v6"
	"github.com/sirupsen/logrus"

	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/gopherlearning/track-devops/internal/server/storage/local"
	"github.com/gopherlearning/track-devops/internal/server/storage/postgres"
	"github.com/gopherlearning/track-devops/internal/server/web"
)

type Args struct {
	ServerAddr    string        `short:"a" help:"Server address" name:"address" env:"ADDRESS" default:"127.0.0.1:8080"`
	StoreFile     string        `short:"f" help:"строка, имя файла, где хранятся значения (пустое значение — отключает функцию записи на диск)" env:"STORE_FILE" default:"/tmp/devops-metrics-db.json"`
	DatabaseDSN   string        `short:"d" help:"строка с адресом подключения к БД" env:"DATABASE_DSN"`
	Key           string        `short:"k" help:"Ключ подписи" env:"KEY"`
	Restore       bool          `short:"r" help:"булево значение (true/false), определяющее, загружать или нет начальные значения из указанного файла при старте сервера" env:"RESTORE" default:"true"`
	UsePprof      bool          `help:"Использовать Pprof" env:"PPROF"`
	ShowStore     bool          `help:"Переодически выводить содержимое в консоль"`
	StoreInterval time.Duration `short:"i" help:"интервал времени в секундах, по истечении которого текущие показания сервера сбрасываются на диск (значение 0 — делает запись синхронной)" env:"STORE_INTERVAL" default:"300s"`
}

// showContent период вывода содержимого хранилища
const showContent = 5 * time.Second

var args Args

func init() {
	// только для прохождения теста
	for i := 0; i < len(os.Args); i++ {
		if strings.Contains(os.Args[i], "=") {
			a := strings.Split(os.Args[i], "=")
			if a[0] == "-r" {
				os.Args[i] = fmt.Sprintf("--restore=%s", a[1])
				continue
			}
			if a[0] == "-d" {
				os.Args[i] = fmt.Sprintf("--database-dsn=%s", a[1])
				continue
			}
			os.Args = append(os.Args[:i], append(a, os.Args[i+1:]...)...)
		}
	}
}

func main() {
	logrus.SetReportCaller(true)
	var err error
	kong.Parse(&args)
	err = env.Parse(&args)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("%+v", args)
	var store repositories.Repository
	if len(args.DatabaseDSN) != 0 {
		store, err = postgres.NewStorage(args.DatabaseDSN, logrus.StandardLogger())
		if err != nil {
			logrus.Error(err)
			return

		}
	} else {
		store, err = local.NewStorage(args.Restore, &args.StoreInterval, args.StoreFile)
		if err != nil {
			logrus.Error(err)
			return
		}
	}
	s := web.NewEchoServer(store, web.WithKey([]byte(args.Key)), web.WithPprof(args.UsePprof), web.WithLoger(logrus.StandardLogger()))

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	// Периодический вывод содержимого хранилища
	if args.ShowStore {
		go func() {
			ticker := time.NewTicker(showContent)
			for {
				<-ticker.C

				fmt.Println("==============================")
				var list map[string][]string
				list, err = store.List(context.Background())
				if err != nil {
					logrus.Error(err)
					continue
				}
				for target, values := range list {
					fmt.Printf(`Target "%s":%s`, target, "\n")
					for _, v := range values {
						fmt.Printf("\t%s\n", v)
					}
				}
			}
		}()
	}
	go func() {
		err = s.Start(args.ServerAddr)
		if err != nil {
			logrus.Error(err)
		}
	}()
	sig := <-terminate
	err = s.Stop()
	if err != nil {
		logrus.Error(err)
	}
	if len(args.DatabaseDSN) != 0 {
		err = store.(*postgres.Storage).Close(context.Background())
		if err != nil {
			logrus.Error(err)
		}
	} else {
		err = store.(*local.Storage).Save()
		if err != nil {
			logrus.Error(err)
		}
	}
	logrus.Infof("Server stoped by signal \"%v\"\n", sig)
}
