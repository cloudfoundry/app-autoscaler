package config

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"code.cloudfoundry.org/locket"
	yaml "gopkg.in/yaml.v2"
)

const (
	DefaultLoggingLevel        string        = "info"
	DefaultRefreshInterval     time.Duration = 24 * time.Hour
	DefaultCutoffDays          int           = 30
	DefaultSyncInterval        time.Duration = 24 * time.Hour
	DefaultLockTTL             time.Duration = locket.DefaultSessionTTL
	DefaultRetryInterval       time.Duration = locket.RetryInterval
	DefaultDBLockRetryInterval time.Duration = 5 * time.Second
	DefaultDBLockTTL           time.Duration = 15 * time.Second
)

type InstanceMetricsDbPrunerConfig struct {
	DB              db.DatabaseConfig `yaml:"db"`
	RefreshInterval time.Duration     `yaml:"refresh_interval"`
	CutoffDays      int               `yaml:"cutoff_days"`
}

type AppMetricsDBPrunerConfig struct {
	DB              db.DatabaseConfig `yaml:"db"`
	RefreshInterval time.Duration     `yaml:"refresh_interval"`
	CutoffDays      int               `yaml:"cutoff_days"`
}

type ScalingEngineDBPrunerConfig struct {
	DB              db.DatabaseConfig `yaml:"db"`
	RefreshInterval time.Duration     `yaml:"refresh_interval"`
	CutoffDays      int               `yaml:"cutoff_days"`
}

type LockConfig struct {
	LockTTL             time.Duration `yaml:"lock_ttl"`
	LockRetryInterval   time.Duration `yaml:"lock_retry_interval"`
	ConsulClusterConfig string        `yaml:"consul_cluster_config"`
}

type DBLockConfig struct {
	LockTTL           time.Duration     `yaml:"ttl"`
	LockRetryInterval time.Duration     `yaml:"retry_interval"`
	DB                db.DatabaseConfig `yaml:"db"`
}

var defaultDBLockConfig = DBLockConfig{
	LockTTL:           DefaultDBLockTTL,
	LockRetryInterval: DefaultDBLockRetryInterval,
}

type ScalingEngineConfig struct {
	URL            string          `yaml:"scaling_engine_url"`
	SyncInterval   time.Duration   `yaml:"sync_interval"`
	TLSClientCerts models.TLSCerts `yaml:"tls"`
}

type SchedulerConfig struct {
	URL            string          `yaml:"scheduler_url"`
	SyncInterval   time.Duration   `yaml:"sync_interval"`
	TLSClientCerts models.TLSCerts `yaml:"tls"`
}

type AppSyncerConfig struct {
	DB           db.DatabaseConfig `yaml:"db"`
	SyncInterval time.Duration     `yaml:"sync_interval"`
}

type Config struct {
	CF                cf.CFConfig                   `yaml:"cf"`
	Logging           helpers.LoggingConfig         `yaml:"logging"`
	InstanceMetricsDB InstanceMetricsDbPrunerConfig `yaml:"instance_metrics_db"`
	AppMetricsDB      AppMetricsDBPrunerConfig      `yaml:"app_metrics_db"`
	ScalingEngineDB   ScalingEngineDBPrunerConfig   `yaml:"scaling_engine_db"`
	ScalingEngine     ScalingEngineConfig           `yaml:"scaling_engine"`
	Scheduler         SchedulerConfig               `yaml:"scheduler"`
	AppSyncer         AppSyncerConfig               `yaml:"app_syncer"`
	Lock              LockConfig                    `yaml:"lock"`
	DBLock            DBLockConfig                  `yaml:"db_lock"`
	EnableDBLock      bool                          `yaml:"enable_db_lock"`
}

var defaultConfig = Config{
	CF: cf.CFConfig{
		GrantType:         cf.GrantTypePassword,
		SkipSSLValidation: false,
	},
	Logging: helpers.LoggingConfig{Level: DefaultLoggingLevel},
	InstanceMetricsDB: InstanceMetricsDbPrunerConfig{
		RefreshInterval: DefaultRefreshInterval,
		CutoffDays:      DefaultCutoffDays,
	},
	AppMetricsDB: AppMetricsDBPrunerConfig{
		RefreshInterval: DefaultRefreshInterval,
		CutoffDays:      DefaultCutoffDays,
	},
	ScalingEngineDB: ScalingEngineDBPrunerConfig{
		RefreshInterval: DefaultRefreshInterval,
		CutoffDays:      DefaultCutoffDays,
	},
	ScalingEngine: ScalingEngineConfig{
		SyncInterval: DefaultSyncInterval,
	},
	Scheduler: SchedulerConfig{
		SyncInterval: DefaultSyncInterval,
	},
	Lock: LockConfig{
		LockRetryInterval: DefaultRetryInterval,
		LockTTL:           DefaultLockTTL,
	},
	AppSyncer: AppSyncerConfig{
		SyncInterval: DefaultSyncInterval,
	},
	DBLock:       defaultDBLockConfig,
	EnableDBLock: false,
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := defaultConfig

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, &conf)
	if err != nil {
		return nil, err
	}

	conf.Logging.Level = strings.ToLower(conf.Logging.Level)

	return &conf, nil
}

func (c *Config) Validate() error {

	if c.InstanceMetricsDB.DB.URL == "" {
		return fmt.Errorf("Configuration error: instance_metrics_db.db.url is empty")
	}

	if c.InstanceMetricsDB.RefreshInterval <= 0 {
		return fmt.Errorf("Configuration error: instance_metrics_db.refresh_interval is less than or equal to 0")
	}

	if c.InstanceMetricsDB.CutoffDays <= 0 {
		return fmt.Errorf("Configuration error: instance_metrics_db.cutoff_days is less than or equal to 0")
	}

	if c.AppMetricsDB.DB.URL == "" {
		return fmt.Errorf("Configuration error: app_metrics_db.db.url is empty")
	}

	if c.AppMetricsDB.RefreshInterval <= 0 {
		return fmt.Errorf("Configuration error: app_metrics_db.refresh_interval is less than or equal to 0")
	}

	if c.AppMetricsDB.CutoffDays <= 0 {
		return fmt.Errorf("Configuration error: app_metrics_db.cutoff_days is less than or equal to 0")
	}

	if c.ScalingEngineDB.DB.URL == "" {
		return fmt.Errorf("Configuration error: scaling_engine_db.db.url is empty")
	}

	if c.ScalingEngineDB.RefreshInterval <= 0 {
		return fmt.Errorf("Configuration error: scaling_engine_db.refresh_interval is less than or equal to 0")
	}

	if c.ScalingEngineDB.CutoffDays <= 0 {
		return fmt.Errorf("Configuration error: scaling_engine_db.cutoff_days is less than or equal to 0")
	}
	if c.ScalingEngine.URL == "" {
		return fmt.Errorf("Configuration error: scaling_engine.scaling_engine_url is empty")
	}
	if c.ScalingEngine.SyncInterval <= 0 {
		return fmt.Errorf("Configuration error: scaling_engine.sync_interval is less than or equal to 0")
	}
	if c.Scheduler.URL == "" {
		return fmt.Errorf("Configuration error: scheduler.scheduler_url is empty")
	}
	if c.Scheduler.SyncInterval <= 0 {
		return fmt.Errorf("Configuration error: scheduler.sync_interval is less than or equal to 0")
	}

	if c.Lock.LockRetryInterval <= 0 {
		return fmt.Errorf("Configuration error: lock.lock_retry_interval is less than or equal to 0")
	}

	if c.Lock.LockTTL <= 0 {
		return fmt.Errorf("Configuration error: lock.lock_ttl is less than or equal to 0")
	}

	if c.EnableDBLock && c.DBLock.DB.URL == "" {
		return fmt.Errorf("Configuration error: db_lock.db.url is empty")
	}

	if c.AppSyncer.DB.URL == "" {
		return fmt.Errorf("Configuration error: appSyncer.db.url is empty")
	}
	if c.AppSyncer.SyncInterval <= 0 {
		return fmt.Errorf("Configuration error: appSyncer.sync_interval is less than or equal to 0")
	}

	return nil

}
