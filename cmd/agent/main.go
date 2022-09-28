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
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	internal.ReadConfig(args)
	logger := internal.InitLogger(args.Verbose)
	logger.Info("Command arguments", zap.Any("agrs", args))
	client, err := agent.NewClient(ctx, args)
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
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer wg.Wait()
	for {
		select {
		case s := <-terminate:
			logger.Info(fmt.Sprintf("Agent stoped by signal \"%v\"", s))
			wg.Add(1)
			metricStore.Save(ctx, wg, client, args.ServerAddr, args.Format == "json", args.Batch)
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
			go metricStore.Save(ctx, wg, client, args.ServerAddr, args.Format == "json", args.Batch)
		}
	}

}
