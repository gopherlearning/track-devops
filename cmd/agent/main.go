package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/gopherlearning/track-devops/internal"
	"github.com/gopherlearning/track-devops/internal/agent"
	"github.com/gopherlearning/track-devops/internal/metrics"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
	args         = &internal.AgentArgs{}
)

func main() {
	var err error
	fmt.Printf("Build version: %s \nBuild date: %s \nBuild commit: %s \n", buildVersion, buildDate, buildCommit)

	internal.ReadConfig(args)
	logger := internal.InitLogger(args.Verbose)
	logger.Info("Command arguments", zap.Any("agrs", args))
	httpClient, err := agent.NewClient(args.CryptoKey, args.SelfAddress)
	if err != nil {
		logger.Fatal(err.Error())
	}
	tickerPoll := time.NewTicker(args.PollInterval)
	tickerReport := time.NewTicker(args.ReportInterval)
	metricStore := metrics.NewStore([]byte(args.Key), logger)
	metricStore.AddCustom(
		new(metrics.PollCount),
		new(metrics.RandomValue),
		new(metrics.TotalMemory),
		new(metrics.FreeMemory),
		new(metrics.CPUutilization1),
	)
	wg := &sync.WaitGroup{}
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	save := func() {
		wg.Done()
		baseURL := fmt.Sprintf("http://%s", args.ServerAddr)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := metricStore.Save(ctx, httpClient, &baseURL, args.Format == "json", args.Batch)
		if err != nil {
			logger.Error("metric store Save() failed", zap.Error(err))
		}
	}
	for {
		select {
		case s := <-terminate:
			logger.Info(fmt.Sprintf("Agent stoped by signal \"%v\"", s))
			wg.Add(1)
			save()
			return
		case <-tickerPoll.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := metricStore.Scrape()
				if err != nil {
					logger.Error("metric store Scrape() failed", zap.Error(err))
				}
			}()
		case <-tickerReport.C:
			wg.Add(1)
			go save()
		}
	}
}
