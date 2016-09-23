package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	DefaultLoggingLevel        = "info"
	DefaultIntervalInHours int = 24
	DefaultCutoffDays      int = 30
)

type LoggingConfig struct {
	Level string `yaml:"level"`
}

var defaultLoggingConfig = LoggingConfig{
	Level: DefaultLoggingLevel,
}

type DbConfig struct {
	MetricsDbUrl string `yaml:"metrics_db_url"`
}

type PrunerConfig struct {
	IntervalInHours int `yaml:"interval_in_hours"`
	CutoffDays      int `yaml:"cutoff_days"`
}

var defaultPrunerConfig = PrunerConfig{
	IntervalInHours: DefaultIntervalInHours,
	CutoffDays:      DefaultCutoffDays,
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

	if c.Pruner.IntervalInHours < 0 {
		return fmt.Errorf("Configuration error: Interval in hours is negative")
	}

	if c.Pruner.CutoffDays < 0 {
		return fmt.Errorf("Configuration error: Cutoff days is negative")
	}

	return nil

}
