package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"autoscaler/cf"
	"autoscaler/models"
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
	PolicyDbUrl          string `yaml:"policy_db_url"`
	InstanceMetricsDbUrl string `yaml:"instance_metrics_db_url"`
}

type CollectorConfig struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	PollInterval    time.Duration `yaml:"poll_interval"`
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
	SSL       models.SSLCerts `yaml:"ssl"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Cf:        defaultCfConfig,
		Logging:   defaultLoggingConfig,
		Server:    defaultServerConfig,
		Collector: defaultCollectorConfig,
	}

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, conf)
	if err != nil {
		return nil, err
	}

	conf.Cf.GrantType = strings.ToLower(conf.Cf.GrantType)
	conf.Logging.Level = strings.ToLower(conf.Logging.Level)

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

	if c.Db.InstanceMetricsDbUrl == "" {
		return fmt.Errorf("Configuration error: InstanceMetrics DB url is empty")
	}

	return nil

}
