package db

import (
	"autoscaler/models"
)

const PostgresDriverName = "postgres"

type OrderType uint8

const (
	DESC OrderType = iota
	ASC
)
const (
	DESCSTR string = "DESC"
	ASCSTR  string = "ASC"
)

type InstanceMetricsDB interface {
	RetrieveInstanceMetrics(appid string, name string, start int64, end int64) ([]*models.AppInstanceMetric, error)
	SaveMetric(metric *models.AppInstanceMetric) error
	PruneInstanceMetrics(before int64) error
	Close() error
}

type PolicyDB interface {
	GetAppIds() (map[string]struct{}, error)
	GetAppPolicy(appId string) (*models.ScalingPolicy, error)
	RetrievePolicies() ([]*models.PolicyJson, error)
	Close() error
}

type AppMetricDB interface {
	SaveAppMetric(appMetric *models.AppMetric) error
	RetrieveAppMetrics(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error)
	PruneAppMetrics(before int64) error
	Close() error
}

type ScalingEngineDB interface {
	SaveScalingHistory(history *models.AppScalingHistory) error
	RetrieveScalingHistories(appId string, start int64, end int64, orderType OrderType) ([]*models.AppScalingHistory, error)
	PruneScalingHistories(before int64) error
	UpdateScalingCooldownExpireTime(appId string, expireAt int64) error
	CanScaleApp(appId string) (bool, error)
	GetActiveSchedule(appId string) (*models.ActiveSchedule, error)
	GetActiveSchedules() (map[string]string, error)
	SetActiveSchedule(appId string, schedule *models.ActiveSchedule) error
	RemoveActiveSchedule(appId string) error
	Close() error
}

type SchedulerDB interface {
	GetActiveSchedules() (map[string]*models.ActiveSchedule, error)
	SynchronizeActiveSchedules(map[string]struct{}) error
	Close() error
}
