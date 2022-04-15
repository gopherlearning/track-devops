package repositories

import "github.com/gopherlearning/track-devops/internal/metrics"

type Repository interface {
	// Get(target, metric, name string) string
	GetMetric(target, mType, name string) *metrics.Metrics
	// Update(target, metric, name, value string) error
	UpdateMetric(target string, m metrics.Metrics) error
	List(targets ...string) map[string][]string
	ListProm(targets ...string) []byte
}
