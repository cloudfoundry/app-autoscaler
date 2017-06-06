package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"strings"
	"time"
)

var defaultSynchronizeInterval time.Duration = 12 * time.Hour

type LoggingConfig struct {
	Level string `yaml:"level"`
}

var defaultLoggingConfig = LoggingConfig{
	Level: "info",
}

type DbConfig struct {
	PolicyDbUrl    string `yaml:"policy_db_url"`
	SchedulerDbUrl string `yaml:"scheduler_db_url"`
}

type Config struct {
	Logging             LoggingConfig `yaml:"logging"`
	Db                  DbConfig      `yaml:"db"`
	SynchronizeInterval time.Duration `yaml:"synchronize_interval"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Logging:             defaultLoggingConfig,
		SynchronizeInterval: defaultSynchronizeInterval,
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

	if c.Db.PolicyDbUrl == "" {
		return fmt.Errorf("Configuration error: Policy DB url is empty")
	}

	if c.Db.SchedulerDbUrl == "" {
		return fmt.Errorf("Configuration error: Scheduler DB url is empty")
	}

	if c.SynchronizeInterval < 0 {
		return fmt.Errorf("Configuration error: SynchronizeInterval is negative")
	}
	return nil

}
