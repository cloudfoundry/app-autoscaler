package db

import (
	"eventgenerator/appmetric"
	"eventgenerator/policy"
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
	RetrievePolicies() ([]*policy.PolicyJson, error)
	GetAppPolicy(appId string) (*models.ScalingPolicy, error)
	Close() error
}

type AppMetricDB interface {
	SaveAppMetric(appMetric *appmetric.AppMetric) error
	Close() error
}
