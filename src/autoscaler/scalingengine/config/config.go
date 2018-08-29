package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"code.cloudfoundry.org/locket"
	"gopkg.in/yaml.v2"

	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
)

const (
	DefaultActiveScheduleSyncInterval time.Duration = 10 * time.Minute
	DefaultLockTTL                    time.Duration = locket.DefaultSessionTTL
	DefaultRetryInterval              time.Duration = locket.RetryInterval
	DefaultDBLockRetryInterval        time.Duration = 5 * time.Second
	DefaultDBLockTTL                  time.Duration = 15 * time.Second
)

var defaultCFConfig = cf.CFConfig{
	GrantType:         cf.GrantTypePassword,
	SkipSSLValidation: false,
}

type ServerConfig struct {
	Port int             `yaml:"port"`
	TLS  models.TLSCerts `yaml:"tls"`
}

var defaultServerConfig = ServerConfig{
	Port: 8080,
}

type HealthConfig struct {
	Port         int           `yaml:"port"`
	EmitInterval time.Duration `yaml:"emit_interval"`
}

var defaultHealthConfig = HealthConfig{
	Port:         8081,
	EmitInterval: 15 * time.Second,
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

var defaultSynchronizerConfig = SynchronizerConfig{
	ActiveScheduleSyncInterval: DefaultActiveScheduleSyncInterval,
}

type Config struct {
	CF                  cf.CFConfig           `yaml:"cf"`
	Logging             helpers.LoggingConfig `yaml:"logging"`
	Server              ServerConfig          `yaml:"server"`
	Health              HealthConfig          `yaml:"health"`
	DB                  DBConfig              `yaml:"db"`
	DefaultCoolDownSecs int                   `yaml:"defaultCoolDownSecs"`
	LockSize            int                   `yaml:"lockSize"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		CF:      defaultCFConfig,
		Logging: defaultLoggingConfig,
		Server:  defaultServerConfig,
		Health:  defaultHealthConfig,
	}

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, conf)
	if err != nil {
		return nil, err
	}

	conf.CF.GrantType = strings.ToLower(conf.CF.GrantType)
	conf.Logging.Level = strings.ToLower(conf.Logging.Level)
	if conf.Health.EmitInterval <= 0*time.Second {
		conf.Health.EmitInterval = defaultHealthConfig.EmitInterval
	}

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
	return nil

}
