package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env/v6"
	"github.com/sirupsen/logrus"

	"github.com/gopherlearning/track-devops/internal/metrics"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)
var args struct {
	ServerAddr     string        `short:"a" help:"Server address" name:"address" env:"ADDRESS" default:"127.0.0.1:8080"`
	Key            string        `short:"k" help:"Ключ подписи" env:"KEY"`
	Format         string        `short:"f" help:"Report format" env:"FORMAT"`
	Batch          bool          `short:"b" help:"Send batch mrtrics" env:"BATCH" default:"true"`
	PollInterval   time.Duration `short:"p" help:"Poll interval" env:"POLL_INTERVAL" default:"2s"`
	ReportInterval time.Duration `short:"r" help:"Report interval" env:"REPORT_INTERVAL" default:"10s"`
	CryptoKey      string        `help:"Путь к файлу, где хранятся публийчный ключ шифрования" env:"CRYPTO_KEY" default:"key.pub"`
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
