package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/sirupsen/logrus"
)

var args struct {
	// Run struct {
	Config         string        `help:"Config"`
	ServerAddr     string        `help:"Server address" default:"127.0.0.1"`
	ServerPort     string        `help:"Server port" default:"8080"`
	PollInterval   time.Duration `help:"Poll interval" default:"2s"`
	ReportInterval time.Duration `help:"Report interval" default:"10s"`
	Format         string        `help:"Report format"`
}

func main() {
	kong.Parse(&args)
	logrus.Info(args)
	httpClient := http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxConnsPerHost:     10,
			MaxIdleConnsPerHost: 10,
		},
	}
	tickerPoll := time.NewTicker(args.PollInterval)
	tickerReport := time.NewTicker(args.ReportInterval)
	metricStore := metrics.NewStore()
	metricStore.AddCustom(new(metrics.PollCount), new(metrics.RandomValue))
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	for {
		select {
		case s := <-terminate:
			fmt.Printf("Agent stoped by signal \"%v\"\n", s)
			return
		case <-tickerPoll.C:
			err := metricStore.Scrape()
			if err != nil {
				fmt.Println(fmt.Errorf("metric store Scrape() failed: %v", err))
			}
		case <-tickerReport.C:
			baseURL := fmt.Sprintf("http://%s:%s", args.ServerAddr, args.ServerPort)
			err := metricStore.Save(&httpClient, &baseURL, args.Format == "json")
			if err != nil {
				fmt.Println(fmt.Errorf("metric store Save() failed: %v", err))
			}
		}
	}
}
