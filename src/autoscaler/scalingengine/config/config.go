package config

import (
	"fmt"
	"io"
	"strings"

	"github.com/cloudfoundry-incubator/candiedyaml"

	"autoscaler/cf"
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
	Level: "info",
}

type DbConfig struct {
	PolicyDbUrl  string `yaml:"policy_db_url"`
	HistoryDbUrl string `yaml:"history_db_url"`
}

type Config struct {
	Cf      cf.CfConfig   `yaml:"cf"`
	Logging LoggingConfig `yaml:"logging"`
	Server  ServerConfig  `yaml:"server"`
	Db      DbConfig      `yaml:"db"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Cf:      defaultCfConfig,
		Logging: defaultLoggingConfig,
		Server:  defaultServerConfig,
	}

	decoder := candiedyaml.NewDecoder(reader)
	err := decoder.Decode(conf)
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

	if c.Db.HistoryDbUrl == "" {
		return fmt.Errorf("Configuration error: History DB url is empty")
	}

	return nil

}
