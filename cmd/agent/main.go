package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gopherlearning/track-devops/internal/metrics"
)

const (
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
	serverAddr     = "127.0.0.1"
	serverPort     = "8080"
)

func main() {
	httpClient := http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxConnsPerHost:     10,
			MaxIdleConnsPerHost: 10,
		},
	}
	tickerPoll := time.NewTicker(pollInterval)
	tickerReport := time.NewTicker(reportInterval)
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
			baseURL := fmt.Sprintf("http://%s:%s", serverAddr, serverPort)
			err := metricStore.Save(&httpClient, &baseURL)
			if err != nil {
				fmt.Println(fmt.Errorf("metric store Save() failed: %v", err))
			}
		}
	}
}
