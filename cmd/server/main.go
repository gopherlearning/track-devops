package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/gopherlearning/track-devops/internal"
	"github.com/gopherlearning/track-devops/internal/server"
	"github.com/gopherlearning/track-devops/internal/server/storage"
	"github.com/gopherlearning/track-devops/internal/server/web"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
	args         = &internal.ServerArgs{}
)

func main() {
	var err error
	fmt.Printf("Build version: %s \nBuild date: %s \nBuild commit: %s \n", buildVersion, buildDate, buildCommit)
	internal.ReadConfig(args)
	logger := internal.InitLogger(args.Verbose)
	logger.Info("Command arguments", zap.Any("agrs", args))
	if args.GenerateCryptoKeys {
		err = web.GenerateCryptoKeys(args.CryptoKey)
		if err != nil {
			logger.Fatal(err.Error())
		}
	}
	store, err := storage.InitStorage(args, logger)
	if err != nil {
		logger.Fatal(err.Error())
	}
	s, err := server.NewServer(args, store)
	if err != nil {
		logger.Fatal(err.Error())
	}
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	if args.ShowStore {
		go internal.ShowStore(store, logger)
	}

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
