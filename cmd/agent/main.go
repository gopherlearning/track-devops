package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env/v6"
	"github.com/sirupsen/logrus"

	"github.com/gopherlearning/track-devops/internal"
	"github.com/gopherlearning/track-devops/internal/metrics"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
	args         internal.AgentArgs
)

func init() {
	internal.FixAgentArgs()
}

func main() {
	// Printing build options.
	fmt.Printf("Build version:%s \nBuild date:%s \nBuild commit:%s \n", buildVersion, buildDate, buildCommit)

	kong.Parse(&args)
	err := env.Parse(&args)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("%+v", args)
	httpClient := http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxConnsPerHost:     10,
			MaxIdleConnsPerHost: 10,
		},
	}
	tickerPoll := time.NewTicker(args.PollInterval)
	tickerReport := time.NewTicker(args.ReportInterval)
	metricStore := metrics.NewStore([]byte(args.Key))
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
	for {
		select {
		case s := <-terminate:
			fmt.Printf("Agent stoped by signal \"%v\"\n", s)
			return
		case <-tickerPoll.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := metricStore.Scrape()
				if err != nil {
					fmt.Println(fmt.Errorf("metric store Scrape() failed: %v", err))
				}
			}()
		case <-tickerReport.C:
			wg.Add(1)
			go func() {
				wg.Done()
				baseURL := fmt.Sprintf("http://%s", args.ServerAddr)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				err := metricStore.Save(ctx, &httpClient, &baseURL, args.Format == "json", args.Batch)
				if err != nil {
					fmt.Println(fmt.Errorf("metric store Save() failed: %v", err))
				}
			}()
		}
	}
}
