package db

import (
	"metricscollector/metrics"
)

type DB interface {
	RetrieveMetrics(appid string, name string, start int64, end int64) ([]*metrics.Metric, error)
	SaveMetric(metric *metrics.Metric) error
	PruneMetrics(before int64) error
	GetAppIds() (map[string]bool, error)
	Close() error
}
