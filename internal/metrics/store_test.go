package metrics_test

import (
	"github.com/gopherlearning/track-devops/internal/metrics"
	"go.uber.org/zap"
)

func Example() {
	// Create store with sing key
	metricStore := metrics.NewStore([]byte("topSecret"), zap.L())

	// Add custom metrics for scrape
	metricStore.AddCustom(
		new(metrics.PollCount),
		new(metrics.RandomValue),
		new(metrics.TotalMemory),
		new(metrics.FreeMemory),
		new(metrics.CPUutilization1),
	)

}
