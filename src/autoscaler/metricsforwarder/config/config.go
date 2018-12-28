package config

import (
	"autoscaler/db"
	"autoscaler/helpers"
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
	MetronAddress        string                `yaml:"metron_address"`
	LoggregatorConfig    LoggregatorConfig     `yaml:"loggregator"`
	Db                   DbConfig              `yaml:"db"`
	CacheTTL             time.Duration         `yaml:"cache_ttl"`
	CacheCleanupInterval time.Duration         `yaml:"cache_cleanup_interval"`
	PolicyPollerInterval time.Duration         `yaml:"policy_poller_interval"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

var defaultServerConfig = ServerConfig{
	Port: 6110,
}

var defaultLoggingConfig = helpers.LoggingConfig{
	Level: "info",
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type LoggregatorConfig struct {
	CACertFile     string `yaml:"ca_cert"`
	ClientCertFile string `yaml:"client_cert"`
	ClientKeyFile  string `yaml:"client_key"`
}

type DbConfig struct {
	PolicyDb db.DatabaseConfig `yaml:"policy_db"`
}

func LoadConfig(reader io.Reader) (*Config, error) {

	conf := &Config{
		Server:               defaultServerConfig,
		Logging:              defaultLoggingConfig,
		MetronAddress:        DefaultMetronAddress,
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
	if c.LoggregatorConfig.CACertFile == "" {
		return fmt.Errorf("Configuration error: Loggregator CACert is empty")
	}
	if c.LoggregatorConfig.ClientCertFile == "" {
		return fmt.Errorf("Configuration error: Loggregator ClientCert is empty")
	}
	if c.LoggregatorConfig.ClientKeyFile == "" {
		return fmt.Errorf("Configuration error: Loggregator ClientKey is empty")
	}
	return nil

}
