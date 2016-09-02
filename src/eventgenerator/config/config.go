package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"strings"
	"time"
)

const (
	DefaultLoggingLevel               = "info"
	DefaultPollInterval time.Duration = 30 * time.Second
)

type Config struct {
	LogLevel       string        `yaml:"log_level"`
	PolicyDbUrl    string        `yaml:"policy_db_url"`
	AppMetricDbUrl string        `yaml:"appmetric_db_url"`
	PollInterval   time.Duration `yaml:"poll_interval"`
}

func LoadConfig(bytes []byte) (*Config, error) {
	conf := &Config{
		LogLevel:     DefaultLoggingLevel,
		PollInterval: DefaultPollInterval,
	}
	err := yaml.Unmarshal(bytes, &conf)
	if err != nil {
		return nil, err
	}
	conf.LogLevel = strings.ToLower(conf.LogLevel)

	return conf, nil
}

func (c *Config) Validate() error {

	if c.PolicyDbUrl == "" {
		return fmt.Errorf("Configuration error: Policy DB url is empty")
	}
	if c.AppMetricDbUrl == "" {
		return fmt.Errorf("Configuration error: AppMetric DB url is empty")
	}
	if c.PollInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: Poll Interval is le than 0")
	}
	return nil

}
