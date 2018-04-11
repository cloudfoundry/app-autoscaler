package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/models"
)

const DefaultActiveScheduleSyncInterval time.Duration = 10 * time.Minute

var defaultCfConfig = cf.CfConfig{
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

type LoggingConfig struct {
	Level string `yaml:"level"`
}

var defaultLoggingConfig = LoggingConfig{
	Level: "info",
}

type DbConfig struct {
	PolicyDb        db.DatabaseConfig `yaml:"policy_db"`
	ScalingEngineDb db.DatabaseConfig `yaml:"scalingengine_db"`
	SchedulerDb     db.DatabaseConfig `yaml:"scheduler_db"`
}

type SynchronizerConfig struct {
	ActiveScheduleSyncInterval time.Duration `yaml:"active_schedule_sync_interval"`
}

var defaultSynchronizerConfig = SynchronizerConfig{
	ActiveScheduleSyncInterval: DefaultActiveScheduleSyncInterval,
}

type ConsulConfig struct {
	Cluster string `yaml:"cluster"`
}

var defaultConsulConfig = ConsulConfig{}

type Config struct {
	Cf                  cf.CfConfig        `yaml:"cf"`
	Logging             LoggingConfig      `yaml:"logging"`
	Server              ServerConfig       `yaml:"server"`
	Db                  DbConfig           `yaml:"db"`
	Synchronizer        SynchronizerConfig `yaml:"synchronizer"`
	Consul              ConsulConfig       `yaml:"consul"`
	DefaultCoolDownSecs int                `yaml:"defaultCoolDownSecs"`
	LockSize            int                `yaml:"lockSize"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Cf:           defaultCfConfig,
		Logging:      defaultLoggingConfig,
		Server:       defaultServerConfig,
		Synchronizer: defaultSynchronizerConfig,
		Consul:       defaultConsulConfig,
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

	if c.Db.PolicyDb.Url == "" {
		return fmt.Errorf("Configuration error: Policy DB url is empty")
	}

	if c.Db.ScalingEngineDb.Url == "" {
		return fmt.Errorf("Configuration error: ScalingEngine DB url is empty")
	}

	if c.Db.SchedulerDb.Url == "" {
		return fmt.Errorf("Configuration error: Scheduler DB url is empty")
	}

	if c.DefaultCoolDownSecs < 60 || c.DefaultCoolDownSecs > 3600 {
		return fmt.Errorf("Configuration error: DefaultCoolDownSecs should be between 60 and 3600")
	}

	if c.LockSize <= 0 {
		return fmt.Errorf("Configuration error: LockSize is less than or equal to 0")
	}

	return nil

}
