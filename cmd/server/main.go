package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	lokihook "github.com/akkuman/logrus-loki-hook"
	"github.com/alecthomas/kong"
	"github.com/caarlos0/env/v6"
	"github.com/gopherlearning/track-devops/cmd/server/handlers"
	"github.com/gopherlearning/track-devops/cmd/server/storage"
	"github.com/gopherlearning/track-devops/cmd/server/web"

	"github.com/sirupsen/logrus"
)

type Args struct {
	ServerAddr    string        `short:"a" help:"Server address" name:"address" env:"ADDRESS" default:"127.0.0.1:8080"`
	StoreInterval time.Duration `short:"i" help:"интервал времени в секундах, по истечении которого текущие показания сервера сбрасываются на диск (значение 0 — делает запись синхронной)" env:"STORE_INTERVAL" default:"300s"`
	StoreFile     string        `short:"f" help:"строка, имя файла, где хранятся значения (пустое значение — отключает функцию записи на диск)" env:"STORE_FILE" default:"/tmp/devops-metrics-db.json"`
	Restore       bool          `short:"r" help:"булево значение (true/false), определяющее, загружать или нет начальные значения из указанного файла при старте сервера" env:"RESTORE" default:"true"`
}

// func (a Args) Validate() error {
// 	logrus.Warn(a)
// 	// r := reflect.ValueOf(a)

// 	ps := reflect.ValueOf(&a)
// 	// struct
// 	s := ps.Elem()
// 	for i := 0; i < s.NumField(); i++ {
// 		f := s.Field(i)
// 		if f.IsValid() {
// 			// A Value can be changed only if it is
// 			// addressable and was not obtained by
// 			// the use of unexported struct fields.
// 			if f.CanSet() {
// 				// change value of N
// 				if f.Kind() == reflect.String {
// 					if len(f.String()) != 0 && f.String()[0] == '=' {
// 						f.SetString(f.String()[1:])
// 						args = a
// 					}
// 				}
// 			}
// 		}
// 	}
// 	// if s.Kind() == reflect.Struct {
// 	// 	// exported field
// 	// }
// 	return nil
// }

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
			os.Args = append(os.Args[:i], append(a, os.Args[i+1:]...)...)
		}
	}
	lokiHookConfig := &lokihook.Config{
		URL: "https://logsremoteloki:efnd9DG510YnZQUjMlgMYVIN@loki.duduh.ru/api/prom/push",
		Labels: map[string]string{
			"app": "track-devops",
		},
	}
	hook, err := lokihook.NewHook(lokiHookConfig)
	if err != nil {
		logrus.Error(err)
	} else {
		logrus.AddHook(hook)
	}

}

func main() {
	kong.Parse(&args)
	err := env.Parse(&args)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("%+v", args)
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
	logrus.Infof("Server stoped by signal \"%v\"\n", sig)
}
