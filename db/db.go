package db

import (
	"fmt"
	"io"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	PostgresDriverName = "postgres"
	MysqlDriverName    = "mysql"
	PolicyDb           = "policy_db"
	BindingDb          = "binding_db"
	StoredProcedureDb  = "storedprocedure_db"
)

type OrderType uint8
type Name = string

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
	ConnectionMaxIdleTime time.Duration `yaml:"connection_max_idletime"`
}

type InstanceMetricsDB interface {
	healthendpoint.DatabaseStatus
	RetrieveInstanceMetrics(appid string, instanceIndex int, name string, start int64, end int64, orderType OrderType) ([]*models.AppInstanceMetric, error)
	SaveMetric(metric *models.AppInstanceMetric) error
	SaveMetricsInBulk(metrics []*models.AppInstanceMetric) error
	PruneInstanceMetrics(before int64) error
	io.Closer
}

type PolicyDB interface {
	healthendpoint.DatabaseStatus
	healthendpoint.Pinger
	GetAppIds() (map[string]bool, error)
	GetAppPolicy(appId string) (*models.ScalingPolicy, error)
	SaveAppPolicy(appId string, policy string, policyGuid string) error
	SetOrUpdateDefaultAppPolicy(appIds []string, oldPolicyGuid string, newPolicy string, newPolicyGuid string) ([]string, error)
	DeletePoliciesByPolicyGuid(policyGuid string) ([]string, error)
	RetrievePolicies() ([]*models.PolicyJson, error)
	io.Closer
	DeletePolicy(appId string) error
	SaveCredential(appId string, cred models.Credential) error
	DeleteCredential(appId string) error
	GetCredential(appId string) (*models.Credential, error)
}

type BindingDB interface {
	healthendpoint.DatabaseStatus
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
	CountServiceInstancesInOrg(orgId string) (int, error)
	GetBindingIdsByInstanceId(instanceId string) ([]string, error)
	io.Closer
}

type AppMetricDB interface {
	healthendpoint.DatabaseStatus
	SaveAppMetric(appMetric *models.AppMetric) error
	SaveAppMetricsInBulk(metrics []*models.AppMetric) error
	RetrieveAppMetrics(appId string, metricType string, start int64, end int64, orderType OrderType) ([]*models.AppMetric, error)
	PruneAppMetrics(before int64) error
	io.Closer
}

type ScalingEngineDB interface {
	healthendpoint.DatabaseStatus
	SaveScalingHistory(history *models.AppScalingHistory) error
	RetrieveScalingHistories(appId string, start int64, end int64, orderType OrderType, includeAll bool) ([]*models.AppScalingHistory, error)
	PruneScalingHistories(before int64) error
	UpdateScalingCooldownExpireTime(appId string, expireAt int64) error
	CanScaleApp(appId string) (bool, int64, error)
	GetActiveSchedule(appId string) (*models.ActiveSchedule, error)
	GetActiveSchedules() (map[string]string, error)
	SetActiveSchedule(appId string, schedule *models.ActiveSchedule) error
	RemoveActiveSchedule(appId string) error
	io.Closer
}

type SchedulerDB interface {
	healthendpoint.DatabaseStatus
	GetActiveSchedules() (map[string]*models.ActiveSchedule, error)
	io.Closer
}

type LockDB interface {
	Lock(lock *models.Lock) (bool, error)
	Release(owner string) error
	io.Closer
}

type StoredProcedureDB interface {
	healthendpoint.Pinger
	io.Closer
	CreateCredentials(credOptions models.CredentialsOptions) (*models.Credential, error)
	DeleteCredentials(credOptions models.CredentialsOptions) error
	DeleteAllInstanceCredentials(instanceId string) error
	ValidateCredentials(creds models.Credential) (*models.CredentialsOptions, error)
}
