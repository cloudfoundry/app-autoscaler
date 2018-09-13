package config

import (
	"autoscaler/db"
	"fmt"
	"time"
	"io"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Logging           LoggingConfig     `yaml:"logging"`
	Server            ServerConfig      `yaml:"server"`
	MetronAddress     string            `yaml:"metron_address"`
	LoggregatorConfig LoggregatorConfig `yaml:"loggregator"`
	Db                DbConfig          `yaml:"db"`
	CacheTTL          time.Duration     `yaml:cache_ttl"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

var defaultServerConfig = ServerConfig{
	Port: 6110,
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

var defaultLoggingConfig = LoggingConfig{
	Level: "info",
}

type LoggregatorConfig struct {
	CACertFile     string `yaml:"ca_cert"`
	ClientCertFile string `yaml:"client_cert"`
	ClientKeyFile  string `yaml:"client_key"`
}

type DbConfig struct {
	PolicyDb db.DatabaseConfig `yaml:"policy_db"`
}

const (
	defaultMetronAddress = "127.0.0.1:3458"
	defaultCacheTTL      time.Duration = 15 * time.Minute
)

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Server:        defaultServerConfig,
		Logging:       defaultLoggingConfig,
		MetronAddress: defaultMetronAddress,
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

	return nil

}
