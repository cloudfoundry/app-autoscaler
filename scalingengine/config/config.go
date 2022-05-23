package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	DefaultHttpClientTimeout = 5 * time.Second
)

var defaultCFConfig = cf.CFConfig{
	SkipSSLValidation: false,
}

type ServerConfig struct {
	Port int             `yaml:"port"`
	TLS  models.TLSCerts `yaml:"tls"`
}

var defaultServerConfig = ServerConfig{
	Port: 8080,
}

var defaultHealthConfig = models.HealthConfig{
	Port: 8081,
}

var defaultLoggingConfig = helpers.LoggingConfig{
	Level: "info",
}

type DBConfig struct {
	PolicyDB        db.DatabaseConfig `yaml:"policy_db"`
	ScalingEngineDB db.DatabaseConfig `yaml:"scalingengine_db"`
	SchedulerDB     db.DatabaseConfig `yaml:"scheduler_db"`
}

type SynchronizerConfig struct {
	ActiveScheduleSyncInterval time.Duration `yaml:"active_schedule_sync_interval"`
}

type Config struct {
	CF                  cf.CFConfig           `yaml:"cf"`
	Logging             helpers.LoggingConfig `yaml:"logging"`
	Server              ServerConfig          `yaml:"server"`
	Health              models.HealthConfig   `yaml:"health"`
	DB                  DBConfig              `yaml:"db"`
	DefaultCoolDownSecs int                   `yaml:"defaultCoolDownSecs"`
	LockSize            int                   `yaml:"lockSize"`
	HttpClientTimeout   time.Duration         `yaml:"http_client_timeout"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		CF:                defaultCFConfig,
		Logging:           defaultLoggingConfig,
		Server:            defaultServerConfig,
		Health:            defaultHealthConfig,
		HttpClientTimeout: DefaultHttpClientTimeout,
	}

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = yaml.UnmarshalStrict(bytes, conf)
	if err != nil {
		return nil, err
	}

	conf.Logging.Level = strings.ToLower(conf.Logging.Level)

	return conf, nil
}

func (c *Config) Validate() error {
	err := c.CF.Validate()
	if err != nil {
		return err
	}

	if c.DB.PolicyDB.URL == "" {
		return fmt.Errorf("Configuration error: db.policy_db.url is empty")
	}

	if c.DB.ScalingEngineDB.URL == "" {
		return fmt.Errorf("Configuration error: db.scalingengine_db.url is empty")
	}

	if c.DB.SchedulerDB.URL == "" {
		return fmt.Errorf("Configuration error: db.scheduler_db.url is empty")
	}

	if c.DefaultCoolDownSecs < 60 || c.DefaultCoolDownSecs > 3600 {
		return fmt.Errorf("Configuration error: DefaultCoolDownSecs should be between 60 and 3600")
	}

	if c.LockSize <= 0 {
		return fmt.Errorf("Configuration error: LockSize is less than or equal to 0")
	}

	if c.HttpClientTimeout <= time.Duration(0) {
		return fmt.Errorf("Configuration error: http_client_timeout is less-equal than 0")
	}

	if err := c.Health.Validate(); err != nil {
		return err
	}

	return nil
}
