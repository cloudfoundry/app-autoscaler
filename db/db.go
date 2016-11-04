package db

import (
	"autoscaler/eventgenerator/model"
	"autoscaler/models"
)

const PostgresDriverName = "postgres"

type MetricsDB interface {
	RetrieveMetrics(appid string, name string, start int64, end int64) ([]*models.Metric, error)
	SaveMetric(metric *models.Metric) error
	PruneMetrics(before int64) error
	Close() error
}

type InstanceMetricsDB interface {
	RetrieveInstanceMetrics(appid string, name string, start int64, end int64) ([]*models.AppInstanceMetric, error)
	SaveMetric(metric *models.AppInstanceMetric) error
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
	RetrieveAppMetrics(appId string, metricType string, start int64, end int64) ([]*model.AppMetric, error)
	PruneAppMetrics(before int64) error
	Close() error
}

type ScalingEngineDB interface {
	SaveScalingHistory(history *models.AppScalingHistory) error
	RetrieveScalingHistories(appId string, start int64, end int64) ([]*models.AppScalingHistory, error)
	UpdateScalingCooldownExpireTime(appId string, expireAt int64) error
	CanScaleApp(appId string) (bool, error)
	GetActiveSchedule(appId string) (*models.ActiveSchedule, error)
	SetActiveSchedule(appId string, schedule *models.ActiveSchedule) error
	RemoveActiveSchedule(appId string) error
	Close() error
}
