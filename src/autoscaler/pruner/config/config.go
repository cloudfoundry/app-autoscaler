package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"code.cloudfoundry.org/locket"

	"gopkg.in/yaml.v2"
)

const (
	DefaultServerPort      int           = 8080
	DefaultLoggingLevel    string        = "info"
	DefaultRefreshInterval time.Duration = 24 * time.Hour
	DefaultCutoffDays      int           = 30
	DefaultLockTTL         time.Duration = locket.DefaultSessionTTL
	DefaultRetryInterval   time.Duration = locket.RetryInterval
)

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type InstanceMetricsDbPrunerConfig struct {
	DbUrl           string        `yaml:"db_url"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	CutoffDays      int           `yaml:"cutoff_days"`
}

type AppMetricsDbPrunerConfig struct {
	DbUrl           string        `yaml:"db_url"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	CutoffDays      int           `yaml:"cutoff_days"`
}

type ScalingEngineDbPrunerConfig struct {
	DbUrl           string        `yaml:"db_url"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	CutoffDays      int           `yaml:"cutoff_days"`
}

type LockConfig struct {
	LockTTL             time.Duration `yaml:"lock_ttl"`
	LockRetryInterval   time.Duration `yaml:"lock_retry_interval"`
	ConsulClusterConfig string        `yaml:"consul_cluster_config"`
}

type Config struct {
	Logging           LoggingConfig                 `yaml:"logging"`
	InstanceMetricsDb InstanceMetricsDbPrunerConfig `yaml:"instance_metrics_db"`
	AppMetricsDb      AppMetricsDbPrunerConfig      `yaml:"app_metrics_db"`
	ScalingEngineDb   ScalingEngineDbPrunerConfig   `yaml:"scaling_engine_db"`
	Lock              LockConfig                    `yaml:"lock"`
}

var defaultDbConfig = Config{
	Logging: LoggingConfig{Level: DefaultLoggingLevel},
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
	Lock: LockConfig{
		LockRetryInterval: DefaultRetryInterval,
		LockTTL:           DefaultLockTTL,
	},
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := defaultDbConfig

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

	if c.InstanceMetricsDb.DbUrl == "" {
		return fmt.Errorf("Configuration error: InstanceMetrics DB url is empty")
	}

	if c.InstanceMetricsDb.RefreshInterval < 0 {
		return fmt.Errorf("Configuration error: InstanceMetrics DB refresh interval is negative")
	}

	if c.InstanceMetricsDb.CutoffDays < 0 {
		return fmt.Errorf("Configuration error: InstanceMetrics DB cutoff days is negative")
	}

	if c.AppMetricsDb.DbUrl == "" {
		return fmt.Errorf("Configuration error: AppMetrics DB url is empty")
	}

	if c.AppMetricsDb.RefreshInterval < 0 {
		return fmt.Errorf("Configuration error: AppMetrics DB refresh interval is negative")
	}

	if c.AppMetricsDb.CutoffDays < 0 {
		return fmt.Errorf("Configuration error: AppMetrics DB cutoff days is negative")
	}

	if c.ScalingEngineDb.DbUrl == "" {
		return fmt.Errorf("Configuration error: ScalingEngine DB url is empty")
	}

	if c.ScalingEngineDb.RefreshInterval < 0 {
		return fmt.Errorf("Configuration error: ScalingEngine DB refresh interval is negative")
	}

	if c.ScalingEngineDb.CutoffDays < 0 {
		return fmt.Errorf("Configuration error: ScalingEngine DB cutoff days is negative")
	}

	if c.Lock.LockRetryInterval <= 0 {
		return fmt.Errorf("Configuration error: lock retry interval is less than or equal to 0")
	}

	if c.Lock.LockTTL <= 0 {
		return fmt.Errorf("Configuration error: lock ttl is less than or equal to 0")
	}

	if c.Lock.ConsulClusterConfig == "" {
		return fmt.Errorf("Configuration error: Consul Cluster Config is empty")
	}

	return nil

}
