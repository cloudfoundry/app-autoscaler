package db

import (
	"autoscaler/models"
	"fmt"
	"time"

	"database/sql"
)

const PostgresDriverName = "postgres"
const MysqlDriverName = "mysql"

type OrderType uint8

const (
	DESC OrderType = iota
	ASC
)
const (
	DESCSTR string = "DESC"
	ASCSTR  string = "ASC"
)

var ErrAlreadyExists = fmt.Errorf("already exists")
var ErrDoesNotExist = fmt.Errorf("doesn't exist")
var ErrConflict = fmt.Errorf("conflicting entry exists")

type DatabaseConfig struct {
	URL                   string        `yaml:"url"`
	MaxOpenConnections    int           `yaml:"max_open_connections"`
	MaxIdleConnections    int           `yaml:"max_idle_connections"`
	ConnectionMaxLifetime time.Duration `yaml:"connection_max_lifetime"`
}
type DatabaseStatus interface {
	GetDBStatus() sql.DBStats
}
type InstanceMetricsDB interface {
	DatabaseStatus
	RetrieveInstanceMetrics(appid string, instanceIndex int, name string, start int64, end int64, orderType OrderType) ([]*models.AppInstanceMetric, error)
	SaveMetric(metric *models.AppInstanceMetric) error
	SaveMetricsInBulk(metrics []*models.AppInstanceMetric) error
	PruneInstanceMetrics(before int64) error
	Close() error
}

type PolicyDB interface {
	DatabaseStatus
	GetAppIds() (map[string]bool, error)
	GetAppPolicy(appId string) (*models.ScalingPolicy, error)
	SaveAppPolicy(appId string, policy string, policyGuid string) error
	SetOrUpdateDefaultAppPolicy(appIds []string, oldPolicyGuid string, newPolicy string, newPolicyGuid string) ([]string, error)
	DeletePoliciesByPolicyGuid(policyGuid string) ([]string, error)
	RetrievePolicies() ([]*models.PolicyJson, error)
	Close() error
	DeletePolicy(appId string) error
	SaveCredential(appId string, cred models.Credential) error
	DeleteCredential(appId string) error
	GetCredential(appId string) (*models.Credential, error)
}

type BindingDB interface {
	DatabaseStatus
	CreateServiceInstance(serviceInstance models.ServiceInstance) error
	GetServiceInstance(serviceInstanceId string) (*models.ServiceInstance, error)
	GetServiceInstanceByAppId(appId string) (*models.ServiceInstance, error)
	UpdateServiceInstance(serviceInstance models.ServiceInstance) error
	DeleteServiceInstance(serviceInstanceId string) error
	CreateServiceBinding(bindingId string, serviceInstanceId string, appId string) error
	DeleteServiceBinding(bindingId string) error
	DeleteServiceBindingByAppId(appId string) error
	CheckServiceBinding(appId string) bool
	GetAppIdByBindingId(bindingId string) (string, error)
	GetAppIdsByInstanceId(instanceId string) ([]string, error)
	Close() error
}

type AppMetricDB interface {
	DatabaseStatus
	SaveAppMetric(appMetric *models.AppMetric) error
	SaveAppMetricsInBulk(metrics []*models.AppMetric) error
	RetrieveAppMetrics(appId string, metricType string, start int64, end int64, orderType OrderType) ([]*models.AppMetric, error)
	PruneAppMetrics(before int64) error
	Close() error
}

type ScalingEngineDB interface {
	DatabaseStatus
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
}

type SchedulerDB interface {
	DatabaseStatus
	GetActiveSchedules() (map[string]*models.ActiveSchedule, error)
	Close() error
}

type LockDB interface {
	Lock(lock *models.Lock) (bool, error)
	Release(owner string) error
	Close() error
}
