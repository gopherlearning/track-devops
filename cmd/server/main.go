package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"

	"github.com/gopherlearning/track-devops/internal"
	"github.com/gopherlearning/track-devops/internal/server/storage"
	"github.com/gopherlearning/track-devops/internal/server/web"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
	logger       *zap.Logger
)

var args internal.ServerArgs

func init() {
	internal.FixServerArgs()
	logger = internal.InitLogger(args.Verbose)
}

func main() {
	// Printing build options.
	fmt.Printf("Build version: %s \nBuild date: %s \nBuild commit: %s \n", buildVersion, buildDate, buildCommit)

	var err error
	kong.Parse(&args)
	err = env.Parse(&args)
	if err != nil {
		logger.Fatal(err.Error())
	}
	logger.Info("Command arguments", zap.Any("agrs", args))
	store, err := storage.InitStorage(args, logger)
	if err != nil {
		logger.Fatal(err.Error())
	}
	s := web.NewEchoServer(store, web.WithKey([]byte(args.Key)), web.WithPprof(args.UsePprof), web.WithLogger(logger))

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	if args.ShowStore {
		go internal.ShowStore(store, logger)
	}
	go func() {
		err = s.Start(args.ServerAddr)
		if err != nil {
			logger.Error(err.Error())
		}
	}()
	sig := <-terminate
	err = s.Stop()
	if err != nil {
		logger.Error(err.Error())
	}
	err = storage.CloseStorage(args, store)
	if err != nil {
		logger.Error(err.Error())
	}
	logger.Info(fmt.Sprintf("Server stoped by signal \"%v\"", sig))
}
