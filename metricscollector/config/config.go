package config

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/candiedyaml"

	"autoscaler/cf"
)

const (
	DefaultLoggingLevel                  = "info"
	DefaultRefreshInterval time.Duration = 60 * time.Second
	DefaultPollInterval    time.Duration = 30 * time.Second
)

var defaultCfConfig = cf.CfConfig{
	GrantType: cf.GrantTypePassword,
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

var defaultServerConfig = ServerConfig{
	Port: 8080,
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

var defaultLoggingConfig = LoggingConfig{
	Level: DefaultLoggingLevel,
}

type DbConfig struct {
	PolicyDbUrl  string `yaml:"policy_db_url"`
	MetricsDbUrl string `yaml:"metrics_db_url"`
}

type CollectorConfig struct {
	RefreshIntervalInSeconds int `yaml:"refresh_interval_in_seconds"`
	PollIntervalInSeconds    int `yaml:"poll_interval_in_seconds"`
	RefreshInterval          time.Duration
	PollInterval             time.Duration
}

var defaultCollectorConfig = CollectorConfig{
	RefreshInterval: DefaultRefreshInterval,
	PollInterval:    DefaultPollInterval,
}

type Config struct {
	Cf        cf.CfConfig     `yaml:"cf"`
	Logging   LoggingConfig   `yaml:"logging"`
	Server    ServerConfig    `yaml:"server"`
	Db        DbConfig        `yaml:"db"`
	Collector CollectorConfig `yaml:"collector"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Cf:        defaultCfConfig,
		Logging:   defaultLoggingConfig,
		Server:    defaultServerConfig,
		Collector: defaultCollectorConfig,
	}

	decoder := candiedyaml.NewDecoder(reader)
	err := decoder.Decode(conf)
	if err != nil {
		return nil, err
	}

	conf.Cf.GrantType = strings.ToLower(conf.Cf.GrantType)
	conf.Logging.Level = strings.ToLower(conf.Logging.Level)

	if conf.Collector.PollIntervalInSeconds != 0 {
		conf.Collector.PollInterval = time.Duration(conf.Collector.PollIntervalInSeconds) * time.Second
	}

	if conf.Collector.RefreshIntervalInSeconds != 0 {
		conf.Collector.RefreshInterval = time.Duration(conf.Collector.RefreshIntervalInSeconds) * time.Second
	}

	return conf, nil
}

func (c *Config) Validate() error {
	err := c.Cf.Validate()
	if err != nil {
		return err
	}

	if c.Db.PolicyDbUrl == "" {
		return fmt.Errorf("Configuration error: Policy DB url is empty")
	}

	if c.Db.MetricsDbUrl == "" {
		return fmt.Errorf("Configuration error: Metrics DB url is empty")
	}

	return nil

}
