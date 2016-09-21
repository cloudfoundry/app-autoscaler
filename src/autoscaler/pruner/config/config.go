package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	DefaultLoggingLevel               = "info"
	DefaultRefreshIntervalInHours int = 24
	DefaultCutoffDays             int = 30
)

type LoggingConfig struct {
	Level string `yaml:"level"`
}

var defaultLoggingConfig = LoggingConfig{
	Level: DefaultLoggingLevel,
}

type DbConfig struct {
	MetricsDbUrl    string `yaml:"metrics_db_url"`
	AppMetricsDbUrl string `yaml:"app_metrics_db_url"`
}

type PrunerConfig struct {
	MetricsDbPruner    MetricsDbPrunerConfig    `yaml:"metrics_db"`
	AppMetricsDbPruner AppMetricsDbPrunerConfig `yaml:"app_metrics_db"`
}

type MetricsDbPrunerConfig struct {
	RefreshIntervalInHours int `yaml:"refresh_interval_in_hours"`
	CutoffDays             int `yaml:"cutoff_days"`
}

type AppMetricsDbPrunerConfig struct {
	RefreshIntervalInHours int `yaml:"refresh_interval_in_hours"`
	CutoffDays             int `yaml:"cutoff_days"`
}

var defaultPrunerConfig = PrunerConfig{
	MetricsDbPruner: MetricsDbPrunerConfig{
		RefreshIntervalInHours: DefaultRefreshIntervalInHours,
		CutoffDays:             DefaultCutoffDays,
	},
	AppMetricsDbPruner: AppMetricsDbPrunerConfig{
		RefreshIntervalInHours: DefaultRefreshIntervalInHours,
		CutoffDays:             DefaultCutoffDays,
	},
}

type Config struct {
	Logging LoggingConfig `yaml:"logging"`
	Db      DbConfig      `yaml:"db"`
	Pruner  PrunerConfig  `yaml:"pruner"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Logging: defaultLoggingConfig,
		Pruner:  defaultPrunerConfig,
	}

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, conf)
	if err != nil {
		return nil, err
	}

	conf.Logging.Level = strings.ToLower(conf.Logging.Level)

	return conf, nil
}

func (c *Config) Validate() error {

	if c.Db.MetricsDbUrl == "" {
		return fmt.Errorf("Configuration error: Metrics DB url is empty")
	}

	if c.Db.AppMetricsDbUrl == "" {
		return fmt.Errorf("Configuration error: App Metrics DB url is empty")
	}

	if c.Pruner.MetricsDbPruner.RefreshIntervalInHours < 0 {
		return fmt.Errorf("Configuration error: Metrics DB refresh interval in hours is negative")
	}

	if c.Pruner.MetricsDbPruner.CutoffDays < 0 {
		return fmt.Errorf("Configuration error: Metrics DB cutoff days is negative")
	}

	if c.Pruner.AppMetricsDbPruner.RefreshIntervalInHours < 0 {
		return fmt.Errorf("Configuration error: App Metrics DB refresh interval in hours is negative")
	}

	if c.Pruner.AppMetricsDbPruner.CutoffDays < 0 {
		return fmt.Errorf("Configuration error: App Metrics DB cutoff days is negative")
	}

	return nil

}
