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
	DefaultCutoffDuration      time.Duration = 30 * 24 * time.Hour
	DefaultSyncInterval        time.Duration = 24 * time.Hour
	DefaultRetryInterval       time.Duration = locket.RetryInterval
	DefaultDBLockRetryInterval time.Duration = 5 * time.Second
	DefaultDBLockTTL           time.Duration = 15 * time.Second
	DefaultHttpClientTimeout   time.Duration = 5 * time.Second
)

type InstanceMetricsDbPrunerConfig struct {
	DB              db.DatabaseConfig `yaml:"db"`
	RefreshInterval time.Duration     `yaml:"refresh_interval"`
	CutoffDuration  time.Duration     `yaml:"cutoff_duration"`
}

type AppMetricsDBPrunerConfig struct {
	DB              db.DatabaseConfig `yaml:"db"`
	RefreshInterval time.Duration     `yaml:"refresh_interval"`
	CutoffDuration  time.Duration     `yaml:"cutoff_duration"`
}

type ScalingEngineDBPrunerConfig struct {
	DB              db.DatabaseConfig `yaml:"db"`
	RefreshInterval time.Duration     `yaml:"refresh_interval"`
	CutoffDuration  time.Duration     `yaml:"cutoff_duration"`
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

var defaultHealthConfig = models.HealthConfig{
	Port: 8081,
}

type Config struct {
	CF                cf.CFConfig                   `yaml:"cf"`
	Health            models.HealthConfig           `yaml:"health"`
	Logging           helpers.LoggingConfig         `yaml:"logging"`
	InstanceMetricsDB InstanceMetricsDbPrunerConfig `yaml:"instance_metrics_db"`
	AppMetricsDB      AppMetricsDBPrunerConfig      `yaml:"app_metrics_db"`
	ScalingEngineDB   ScalingEngineDBPrunerConfig   `yaml:"scaling_engine_db"`
	ScalingEngine     ScalingEngineConfig           `yaml:"scaling_engine"`
	Scheduler         SchedulerConfig               `yaml:"scheduler"`
	AppSyncer         AppSyncerConfig               `yaml:"app_syncer"`
	DBLock            DBLockConfig                  `yaml:"db_lock"`
	HttpClientTimeout time.Duration                 `yaml:"http_client_timeout"`
}

var defaultConfig = Config{
	CF: cf.CFConfig{
		SkipSSLValidation: false,
	},
	Health:  defaultHealthConfig,
	Logging: helpers.LoggingConfig{Level: DefaultLoggingLevel},
	InstanceMetricsDB: InstanceMetricsDbPrunerConfig{
		RefreshInterval: DefaultRefreshInterval,
		CutoffDuration:  DefaultCutoffDuration,
	},
	AppMetricsDB: AppMetricsDBPrunerConfig{
		RefreshInterval: DefaultRefreshInterval,
		CutoffDuration:  DefaultCutoffDuration,
	},
	ScalingEngineDB: ScalingEngineDBPrunerConfig{
		RefreshInterval: DefaultRefreshInterval,
		CutoffDuration:  DefaultCutoffDuration,
	},
	ScalingEngine: ScalingEngineConfig{
		SyncInterval: DefaultSyncInterval,
	},
	Scheduler: SchedulerConfig{
		SyncInterval: DefaultSyncInterval,
	},
	AppSyncer: AppSyncerConfig{
		SyncInterval: DefaultSyncInterval,
	},
	DBLock:            defaultDBLockConfig,
	HttpClientTimeout: DefaultHttpClientTimeout,
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

	if c.InstanceMetricsDB.CutoffDuration <= 0 {
		return fmt.Errorf("Configuration error: instance_metrics_db.cutoff_duration is less than or equal to 0")
	}

	if c.AppMetricsDB.DB.URL == "" {
		return fmt.Errorf("Configuration error: app_metrics_db.db.url is empty")
	}

	if c.AppMetricsDB.RefreshInterval <= 0 {
		return fmt.Errorf("Configuration error: app_metrics_db.refresh_interval is less than or equal to 0")
	}

	if c.AppMetricsDB.CutoffDuration <= 0 {
		return fmt.Errorf("Configuration error: app_metrics_db.cutoff_duration is less than or equal to 0")
	}

	if c.ScalingEngineDB.DB.URL == "" {
		return fmt.Errorf("Configuration error: scaling_engine_db.db.url is empty")
	}

	if c.ScalingEngineDB.RefreshInterval <= 0 {
		return fmt.Errorf("Configuration error: scaling_engine_db.refresh_interval is less than or equal to 0")
	}

	if c.ScalingEngineDB.CutoffDuration <= 0 {
		return fmt.Errorf("Configuration error: scaling_engine_db.cutoff_duration is less than or equal to 0")
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

	if c.DBLock.DB.URL == "" {
		return fmt.Errorf("Configuration error: db_lock.db.url is empty")
	}

	if c.AppSyncer.DB.URL == "" {
		return fmt.Errorf("Configuration error: appSyncer.db.url is empty")
	}
	if c.AppSyncer.SyncInterval <= 0 {
		return fmt.Errorf("Configuration error: appSyncer.sync_interval is less than or equal to 0")
	}

	if c.HttpClientTimeout <= time.Duration(0) {
		return fmt.Errorf("Configuration error: http_client_timeout is less-equal than 0")
	}

	if err := c.Health.Validate("operator"); err != nil {
		return err
	}

	return nil

}
