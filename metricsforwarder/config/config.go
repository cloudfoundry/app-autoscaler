package config

import (
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	DefaultMetronAddress                      = "127.0.0.1:3458"
	DefaultCacheTTL             time.Duration = 15 * time.Minute
	DefaultCacheCleanupInterval time.Duration = 6 * time.Hour
	DefaultPolicyPollerInterval time.Duration = 40 * time.Second
)

type Config struct {
	Logging              helpers.LoggingConfig `yaml:"logging"`
	Server               ServerConfig          `yaml:"server"`
	LoggregatorConfig    LoggregatorConfig     `yaml:"loggregator"`
	Db                   DbConfig              `yaml:"db"`
	CacheTTL             time.Duration         `yaml:"cache_ttl"`
	CacheCleanupInterval time.Duration         `yaml:"cache_cleanup_interval"`
	PolicyPollerInterval time.Duration         `yaml:"policy_poller_interval"`
	Health               models.HealthConfig   `yaml:"health"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

var defaultServerConfig = ServerConfig{
	Port: 6110,
}

var defaultHealthConfig = models.HealthConfig{
	Port: 8081,
}

var defaultLoggingConfig = helpers.LoggingConfig{
	Level: "info",
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type LoggregatorConfig struct {
	MetronAddress string          `yaml:"metron_address"`
	TLS           models.TLSCerts `yaml:"tls"`
}

type DbConfig struct {
	PolicyDb db.DatabaseConfig `yaml:"policy_db"`
}

func LoadConfig(reader io.Reader) (*Config, error) {

	conf := &Config{
		Server:  defaultServerConfig,
		Logging: defaultLoggingConfig,
		LoggregatorConfig: LoggregatorConfig{
			MetronAddress: DefaultMetronAddress,
		},
		Health:               defaultHealthConfig,
		CacheTTL:             DefaultCacheTTL,
		CacheCleanupInterval: DefaultCacheCleanupInterval,
		PolicyPollerInterval: DefaultPolicyPollerInterval,
	}

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (c *Config) Validate() error {

	if c.Db.PolicyDb.URL == "" {
		return fmt.Errorf("Configuration error: Policy DB url is empty")
	}
	if c.LoggregatorConfig.TLS.CACertFile == "" {
		return fmt.Errorf("Configuration error: Loggregator CACert is empty")
	}
	if c.LoggregatorConfig.TLS.CertFile == "" {
		return fmt.Errorf("Configuration error: Loggregator ClientCert is empty")
	}
	if c.LoggregatorConfig.TLS.KeyFile == "" {
		return fmt.Errorf("Configuration error: Loggregator ClientKey is empty")
	}
	return nil

}
