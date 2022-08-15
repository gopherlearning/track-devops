package repositories

import (
	"context"

	"github.com/gopherlearning/track-devops/internal/metrics"
)

// Repository storage interface
type Repository interface {
	GetMetric(ctx context.Context, target, mType, name string) (*metrics.Metrics, error)
	UpdateMetric(ctx context.Context, target string, mm ...metrics.Metrics) error
	Metrics(ctx context.Context, target string) (map[string][]metrics.Metrics, error)
	List(ctx context.Context) (map[string][]string, error)
}
