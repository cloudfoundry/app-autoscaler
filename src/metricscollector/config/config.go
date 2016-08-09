package config

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/candiedyaml"
)

const (
	GrantTypePassword                        = "password"
	GrantTypeClientCredentials               = "client_credentials"
	GrantTypeRefreshToken                    = "refresh_token"
	DefaultLoggingLevel                      = "info"
	DefaultRefreshInterval     time.Duration = 60 * time.Second
	DefaultPollInterval        time.Duration = 30 * time.Second
)

type CfConfig struct {
	Api       string `yaml:"api"`
	GrantType string `yaml:"grant_type"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	ClientId  string `yaml:"client_id"`
	Secret    string `yaml:"secret"`
}

var defaultCfConfig = CfConfig{
	GrantType: GrantTypePassword,
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
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	PollInterval    time.Duration `yaml:"poll_interval"`
}

var defaultCollectorConfig = CollectorConfig{
	RefreshInterval: DefaultRefreshInterval,
	PollInterval:    DefaultPollInterval,
}

type Config struct {
	Cf        CfConfig        `yaml:"cf"`
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

	return conf, nil
}

func (c *Config) Validate() error {
	if c.Cf.Api == "" {
		return fmt.Errorf("Configuration error: cf api is empty")
	}

	if c.Cf.GrantType != GrantTypePassword && c.Cf.GrantType != GrantTypeClientCredentials {
		return fmt.Errorf("Configuration error: unsupported grant type [%s]", c.Cf.GrantType)
	}

	if c.Cf.GrantType == GrantTypePassword {
		if c.Cf.Username == "" {
			return fmt.Errorf("Configuration error: user name is empty")
		}
	}

	if c.Cf.GrantType == GrantTypeClientCredentials {
		if c.Cf.ClientId == "" {
			return fmt.Errorf("Configuration error: client id is empty")
		}
	}

	if c.Db.PolicyDbUrl == "" {
		return fmt.Errorf("Configuration error: Policy DB url is empty")
	}

	if c.Db.MetricsDbUrl == "" {
		return fmt.Errorf("Configuration error: Metrics DB url is empty")
	}

	return nil

}
