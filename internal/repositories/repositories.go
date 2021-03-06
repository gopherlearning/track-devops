package repositories

import (
	"context"

	"github.com/gopherlearning/track-devops/internal/metrics"
)

type Repository interface {
	GetMetric(target, mType, name string) *metrics.Metrics
	Ping(context.Context) error
	UpdateMetric(target string, mm ...metrics.Metrics) error
	Metrics() map[string][]metrics.Metrics
	List(targets ...string) map[string][]string
	ListProm(targets ...string) []byte
}
