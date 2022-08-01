package repositories

import (
	"context"

	"github.com/gopherlearning/track-devops/internal/metrics"
)

// Repository storage interface
type Repository interface {
	GetMetric(target, mType, name string) (*metrics.Metrics, error)
	Ping(context.Context) error
	UpdateMetric(ctx context.Context, target string, mm ...metrics.Metrics) error
	Metrics(target string) (map[string][]metrics.Metrics, error)
	List() (map[string][]string, error)
	ListProm(targets ...string) ([]byte, error)
}
