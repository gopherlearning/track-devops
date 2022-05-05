package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env/v6"

	"github.com/alecthomas/kong"
	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/sirupsen/logrus"
)

var args struct {
	ServerAddr     string        `short:"a" help:"Server address" name:"address" env:"ADDRESS" default:"127.0.0.1:8080"`
	PollInterval   time.Duration `short:"p" help:"Poll interval" env:"POLL_INTERVAL" default:"2s"`
	ReportInterval time.Duration `short:"r" help:"Report interval" env:"REPORT_INTERVAL" default:"10s"`
	Key            string        `short:"k" help:"Ключ подписи" env:"KEY"`
	Format         string        `short:"f" help:"Report format" env:"FORMAT"`
	Batch          bool          `short:"b" help:"Send batch mrtrics" env:"BATCH" default:"true"`
}

func init() {
	// только для прохождения теста
	for i := 0; i < len(os.Args); i++ {
		if strings.Contains(os.Args[i], "=") {
			a := strings.Split(os.Args[i], "=")
			os.Args[i] = a[1]
			os.Args = append(os.Args[:i], append(a, os.Args[i+1:]...)...)
		}
	}
}

func main() {
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
			baseURL := fmt.Sprintf("http://%s", args.ServerAddr)
			err := metricStore.Save(&httpClient, &baseURL, args.Format == "json", args.Batch)
			if err != nil {
				fmt.Println(fmt.Errorf("metric store Save() failed: %v", err))
			}
		}
	}
}
