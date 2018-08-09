package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"

	"code.cloudfoundry.org/locket"
	"gopkg.in/yaml.v2"
)

const (
	DefaultLoggingLevel        string        = "info"
	DefaultRefreshInterval     time.Duration = 24 * time.Hour
	DefaultCutoffDays          int           = 30
	DefaultSyncInterval        time.Duration = 10 * time.Minute
	DefaultLockTTL             time.Duration = locket.DefaultSessionTTL
	DefaultRetryInterval       time.Duration = locket.RetryInterval
	DefaultDBLockRetryInterval time.Duration = 5 * time.Second
	DefaultDBLockTTL           time.Duration = 15 * time.Second
)

type InstanceMetricsDbPrunerConfig struct {
	Db              db.DatabaseConfig `yaml:"db"`
	RefreshInterval time.Duration     `yaml:"refresh_interval"`
	CutoffDays      int               `yaml:"cutoff_days"`
}

type AppMetricsDbPrunerConfig struct {
	Db              db.DatabaseConfig `yaml:"db"`
	RefreshInterval time.Duration     `yaml:"refresh_interval"`
	CutoffDays      int               `yaml:"cutoff_days"`
}

type ScalingEngineDbPrunerConfig struct {
	Db              db.DatabaseConfig `yaml:"db"`
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
	LockDB            db.DatabaseConfig `yaml:"lock_db"`
}

var defaultDBLockConfig = DBLockConfig{
	LockTTL:           DefaultDBLockTTL,
	LockRetryInterval: DefaultDBLockRetryInterval,
}

type ScalingEngineConfig struct {
	Url            string          `yaml:"scaling_engine_url"`
	SyncInterval   time.Duration   `yaml:"sync_interval"`
	TLSClientCerts models.TLSCerts `yaml:"tls"`
}

type SchedulerConfig struct {
	Url            string          `yaml:"scheduler_url"`
	SyncInterval   time.Duration   `yaml:"sync_interval"`
	TLSClientCerts models.TLSCerts `yaml:"tls"`
}

type Config struct {
	Logging           helpers.LoggingConfig         `yaml:"logging"`
	InstanceMetricsDb InstanceMetricsDbPrunerConfig `yaml:"instance_metrics_db"`
	AppMetricsDb      AppMetricsDbPrunerConfig      `yaml:"app_metrics_db"`
	ScalingEngineDb   ScalingEngineDbPrunerConfig   `yaml:"scaling_engine_db"`
	ScalingEngine     ScalingEngineConfig           `yaml:"scaling_engine"`
	Scheduler         SchedulerConfig               `yaml:"scheduler"`
	Lock              LockConfig                    `yaml:"lock"`
	DBLock            DBLockConfig                  `yaml:"db_lock"`
	EnableDBLock      bool                          `yaml:"enable_db_lock"`
}

var defaultConfig = Config{
	Logging: helpers.LoggingConfig{Level: DefaultLoggingLevel},
	InstanceMetricsDb: InstanceMetricsDbPrunerConfig{
		RefreshInterval: DefaultRefreshInterval,
		CutoffDays:      DefaultCutoffDays,
	},
	AppMetricsDb: AppMetricsDbPrunerConfig{
		RefreshInterval: DefaultRefreshInterval,
		CutoffDays:      DefaultCutoffDays,
	},
	ScalingEngineDb: ScalingEngineDbPrunerConfig{
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

	if c.InstanceMetricsDb.Db.Url == "" {
		return fmt.Errorf("Configuration error: InstanceMetrics DB url is empty")
	}

	if c.InstanceMetricsDb.RefreshInterval <= 0 {
		return fmt.Errorf("Configuration error: InstanceMetrics DB refresh interval is less than or equal to 0")
	}

	if c.InstanceMetricsDb.CutoffDays <= 0 {
		return fmt.Errorf("Configuration error: InstanceMetrics DB cutoff days is less than or equal to 0")
	}

	if c.AppMetricsDb.Db.Url == "" {
		return fmt.Errorf("Configuration error: AppMetrics DB url is empty")
	}

	if c.AppMetricsDb.RefreshInterval <= 0 {
		return fmt.Errorf("Configuration error: AppMetrics DB refresh interval is less than or equal to 0")
	}

	if c.AppMetricsDb.CutoffDays <= 0 {
		return fmt.Errorf("Configuration error: AppMetrics DB cutoff days is less than or equal to 0")
	}

	if c.ScalingEngineDb.Db.Url == "" {
		return fmt.Errorf("Configuration error: ScalingEngine DB url is empty")
	}

	if c.ScalingEngineDb.RefreshInterval <= 0 {
		return fmt.Errorf("Configuration error: ScalingEngine DB refresh interval is less than or equal to 0")
	}

	if c.ScalingEngineDb.CutoffDays <= 0 {
		return fmt.Errorf("Configuration error: ScalingEngine DB cutoff days is less than or equal to 0")
	}
	if c.ScalingEngine.Url == "" {
		return fmt.Errorf("Configuration error: ScalingEngine url is empty")
	}
	if c.ScalingEngine.SyncInterval <= 0 {
		return fmt.Errorf("Configuration error: ScalingEngine sync interval is less than or equal to 0")
	}
	if c.Scheduler.Url == "" {
		return fmt.Errorf("Configuration error: Scheduler url is empty")
	}
	if c.Scheduler.SyncInterval <= 0 {
		return fmt.Errorf("Configuration error: Scheduler sync interval is less than or equal to 0")
	}

	if c.Lock.LockRetryInterval <= 0 {
		return fmt.Errorf("Configuration error: lock retry interval is less than or equal to 0")
	}

	if c.Lock.LockTTL <= 0 {
		return fmt.Errorf("Configuration error: lock ttl is less than or equal to 0")
	}

	if c.EnableDBLock && c.DBLock.LockDB.Url == "" {
		return fmt.Errorf("Configuration error: Lock DB Url is empty")
	}

	return nil

}
