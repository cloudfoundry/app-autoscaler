package db

import (
	"eventgenerator/model"
	"metricscollector/metrics"
	"models"
)

const PostgresDriverName = "postgres"

type MetricsDB interface {
	RetrieveMetrics(appid string, name string, start int64, end int64) ([]*models.Metric, error)
	SaveMetric(metric *models.Metric) error
	PruneMetrics(before int64) error
	Close() error
}

type PolicyDB interface {
	GetAppIds() (map[string]bool, error)
	GetAppPolicy(appId string) (*models.ScalingPolicy, error)
	RetrievePolicies() ([]*model.PolicyJson, error)
	Close() error
}

type AppMetricDB interface {
	SaveAppMetric(appMetric *model.AppMetric) error
	Close() error
}
