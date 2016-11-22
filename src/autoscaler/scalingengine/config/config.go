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

const DefaultActiveScheduleSyncInterval time.Duration = 10 * time.Minute

var defaultCfConfig = cf.CfConfig{
	GrantType: cf.GrantTypePassword,
}

type ServerConfig struct {
	Port      int  `yaml:"port"`
	EnableSSL bool `yaml:"enable_ssl"`
}

var defaultServerConfig = ServerConfig{
	Port:      8080,
	EnableSSL: true,
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

var defaultLoggingConfig = LoggingConfig{
	Level: "info",
}

type DbConfig struct {
	PolicyDbUrl        string `yaml:"policy_db_url"`
	ScalingEngineDbUrl string `yaml:"scalingengine_db_url"`
	SchedulerDbUrl     string `yaml:"scheduler_db_url"`
}

type SynchronizerConfig struct {
	ActiveScheduleSyncInterval time.Duration `yaml:"active_schedule_sync_interval"`
}

var defaultSynchronizerConfig = SynchronizerConfig{
	ActiveScheduleSyncInterval: DefaultActiveScheduleSyncInterval,
}

type Config struct {
	Cf           cf.CfConfig        `yaml:"cf"`
	Logging      LoggingConfig      `yaml:"logging"`
	Server       ServerConfig       `yaml:"server"`
	Db           DbConfig           `yaml:"db"`
	Synchronizer SynchronizerConfig `yaml:"synchronizer"`
	SSL          models.SSLCerts    `yaml:"ssl"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Cf:           defaultCfConfig,
		Logging:      defaultLoggingConfig,
		Server:       defaultServerConfig,
		Synchronizer: defaultSynchronizerConfig,
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

	if c.Db.ScalingEngineDbUrl == "" {
		return fmt.Errorf("Configuration error: ScalingEngine DB url is empty")
	}

	if c.Db.SchedulerDbUrl == "" {
		return fmt.Errorf("Configuration error: Scheduler DB url is empty")
	}

	return nil

}
