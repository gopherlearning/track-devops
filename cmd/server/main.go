package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env/v6"
	"github.com/sirupsen/logrus"

	"github.com/gopherlearning/track-devops/internal"
	"github.com/gopherlearning/track-devops/internal/server/storage"
	"github.com/gopherlearning/track-devops/internal/server/web"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

var args internal.ServerArgs

func init() {
	internal.FixServerArgs()
}

func main() {
	// Printing build options.
	fmt.Printf("Build version:%s \nBuild date:%s \nBuild commit:%s \n", buildVersion, buildDate, buildCommit)

	logrus.SetReportCaller(true)
	var err error
	kong.Parse(&args)
	err = env.Parse(&args)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("%+v", args)
	store, err := storage.InitStorage(args)
	if err != nil {
		logrus.Fatal(err)
	}
	s := web.NewEchoServer(store, web.WithKey([]byte(args.Key)), web.WithPprof(args.UsePprof), web.WithLoger(logrus.StandardLogger()))

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	if args.ShowStore {
		go internal.ShowStore(store)
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
	err = storage.CloseStorage(args, store)
	if err != nil {
		logrus.Error(err)
	}
	logrus.Infof("Server stoped by signal \"%v\"\n", sig)
}
