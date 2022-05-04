package repositories

import (
	"context"

	"github.com/gopherlearning/track-devops/internal/metrics"
)

type Repository interface {
	// Get(target, metric, name string) string
	GetMetric(target, mType, name string) *metrics.Metrics
	Ping(context.Context) error
	UpdateMetric(target string, m metrics.Metrics) error
	Metrics() map[string][]metrics.Metrics
	List(targets ...string) map[string][]string
	ListProm(targets ...string) []byte
}
