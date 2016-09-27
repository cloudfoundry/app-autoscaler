package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	DefaultLoggingLevel                  = "info"
	DefaultRefreshInterval time.Duration = 24 * time.Hour
	DefaultCutoffDays      int           = 30
)

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type MetricsDbPrunerConfig struct {
	DbUrl           string        `yaml:"db_url"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	CutoffDays      int           `yaml:"cutoff_days"`
}

type AppMetricsDbPrunerConfig struct {
	DbUrl           string        `yaml:"db_url"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	CutoffDays      int           `yaml:"cutoff_days"`
}

type Config struct {
	Logging      LoggingConfig            `yaml:"logging"`
	MetricsDb    MetricsDbPrunerConfig    `yaml:"metrics_db"`
	AppMetricsDb AppMetricsDbPrunerConfig `yaml:"app_metrics_db"`
}

var defaultDbConfig = Config{
	Logging: LoggingConfig{Level: DefaultLoggingLevel},
	MetricsDb: MetricsDbPrunerConfig{
		RefreshInterval: DefaultRefreshInterval,
		CutoffDays:      DefaultCutoffDays,
	},
	AppMetricsDb: AppMetricsDbPrunerConfig{
		RefreshInterval: DefaultRefreshInterval,
		CutoffDays:      DefaultCutoffDays,
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

	if c.MetricsDb.DbUrl == "" {
		return fmt.Errorf("Configuration error: Metrics DB url is empty")
	}

	if c.AppMetricsDb.DbUrl == "" {
		return fmt.Errorf("Configuration error: App Metrics DB url is empty")
	}

	if c.MetricsDb.RefreshInterval < 0 {
		return fmt.Errorf("Configuration error: Metrics DB refresh interval is negative")
	}

	if c.MetricsDb.CutoffDays < 0 {
		return fmt.Errorf("Configuration error: Metrics DB cutoff days is negative")
	}

	if c.AppMetricsDb.RefreshInterval < 0 {
		return fmt.Errorf("Configuration error: App Metrics DB refresh interval is negative")
	}

	if c.AppMetricsDb.CutoffDays < 0 {
		return fmt.Errorf("Configuration error: App Metrics DB cutoff days is negative")
	}

	return nil

}
