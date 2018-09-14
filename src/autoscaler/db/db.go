package db

import (
	"autoscaler/healthendpoint"
	"autoscaler/models"
	"time"

	"code.cloudfoundry.org/clock"
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

type DatabaseConfig struct {
	URL                   string        `yaml:"url"`
	MaxOpenConnections    int           `yaml:"max_open_connections"`
	MaxIdleConnections    int           `yaml:"max_idle_connections"`
	ConnectionMaxLifetime time.Duration `yaml:"connection_max_lifetime"`
}

type InstanceMetricsDB interface {
	RetrieveInstanceMetrics(appid string, instanceIndex int, name string, start int64, end int64, orderType OrderType) ([]*models.AppInstanceMetric, error)
	SaveMetric(metric *models.AppInstanceMetric) error
	SaveMetricsInBulk(metrics []*models.AppInstanceMetric) error
	PruneInstanceMetrics(before int64) error
	Close() error
}

type PolicyDB interface {
	GetAppIds() (map[string]bool, error)
	GetAppPolicy(appId string) (*models.ScalingPolicy, error)
	RetrievePolicies() ([]*models.PolicyJson, error)
	Close() error
	EmitHealthMetrics(h healthendpoint.Health, cclock clock.Clock, interval time.Duration)
	DeletePolicy(appId string) error
}

type AppMetricDB interface {
	SaveAppMetric(appMetric *models.AppMetric) error
	SaveAppMetricsInBulk(metrics []*models.AppMetric) error
	RetrieveAppMetrics(appId string, metricType string, start int64, end int64, orderType OrderType) ([]*models.AppMetric, error)
	PruneAppMetrics(before int64) error
	Close() error
}

type ScalingEngineDB interface {
	SaveScalingHistory(history *models.AppScalingHistory) error
	RetrieveScalingHistories(appId string, start int64, end int64, orderType OrderType, includeAll bool) ([]*models.AppScalingHistory, error)
	PruneScalingHistories(before int64) error
	UpdateScalingCooldownExpireTime(appId string, expireAt int64) error
	CanScaleApp(appId string) (bool, int64, error)
	GetActiveSchedule(appId string) (*models.ActiveSchedule, error)
	GetActiveSchedules() (map[string]string, error)
	SetActiveSchedule(appId string, schedule *models.ActiveSchedule) error
	RemoveActiveSchedule(appId string) error
	Close() error
	EmitHealthMetrics(h healthendpoint.Health, cclock clock.Clock, interval time.Duration)
}

type SchedulerDB interface {
	GetActiveSchedules() (map[string]*models.ActiveSchedule, error)
	Close() error
	EmitHealthMetrics(h healthendpoint.Health, cclock clock.Clock, interval time.Duration)
}

type LockDB interface {
	Lock(lock *models.Lock) (bool, error)
	Release(owner string) error
	Close() error
}
