package db

import (
	"dataaggregator/appmetric"
	"dataaggregator/policy"
	"metricscollector/metrics"
)

const PostgresDriverName = "postgres"

type MetricsDB interface {
	RetrieveMetrics(appid string, name string, start int64, end int64) ([]*metrics.Metric, error)
	SaveMetric(metric *metrics.Metric) error
	PruneMetrics(before int64) error
	Close() error
}

type PolicyDB interface {
	GetAppIds() (map[string]bool, error)
	RetrievePolicies() ([]*policy.PolicyJson, error)
	Close() error
}

type AppMetricDB interface {
	SaveAppMetric(appMetric *appmetric.AppMetric) error
	Close() error
}
