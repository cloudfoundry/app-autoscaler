package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"

	"autoscaler/db"
	"autoscaler/helpers"
)

const (
	DefaultLoggingLevel = "info"
)

type ServerConfig struct {
	Port int `yaml:"port"`
}

var defaultServerConfig = ServerConfig{
	Port: 8080,
}

var defaultLoggingConfig = helpers.LoggingConfig{
	Level: DefaultLoggingLevel,
}

type DBConfig struct {
	BindingDB db.DatabaseConfig `yaml:"binding_db"`
}

type Config struct {
	Logging              helpers.LoggingConfig `yaml:"logging"`
	Server               ServerConfig          `yaml:"server"`
	DB                   DBConfig              `yaml:"db"`
	BrokerUsername       string                `yaml:"broker_username"`
	BrokerPassword       string                `yaml:"broker_password"`
	Catalog              string                `yaml:"catalog"`
	DashboardRedirectURI string                `yaml:"dashboard_redirect_uri"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Logging: defaultLoggingConfig,
		Server:  defaultServerConfig,
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
	if c.DB.BindingDB.URL == "" {
		return fmt.Errorf("Configuration error: BindingDB URL is empty")
	}
	if c.BrokerUsername == "" {
		return fmt.Errorf("Configuration error: BrokerUsername is empty")
	}
	if c.BrokerPassword == "" {
		return fmt.Errorf("Configuration error: BrokerPassword is empty")
	}
	if c.Catalog == "" {
		return fmt.Errorf("Configuration error: Catalog is empty")
	}
	return nil
}
