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
	"autoscaler/models"
)

const (
	DefaultLoggingLevel                  = "info"
	DefaultRefreshInterval time.Duration = 60 * time.Second
	DefaultCollectInterval time.Duration = 30 * time.Second
	DefaultLockTTL         time.Duration = locket.DefaultSessionTTL
	DefaultRetryInterval   time.Duration = locket.RetryInterval
)

var defaultCfConfig = cf.CfConfig{
	GrantType: cf.GrantTypePassword,
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
	Level: DefaultLoggingLevel,
}

type DbConfig struct {
	PolicyDbUrl          string `yaml:"policy_db_url"`
	InstanceMetricsDbUrl string `yaml:"instance_metrics_db_url"`
}

type CollectorConfig struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	CollectInterval time.Duration `yaml:"collect_interval"`
}

var defaultCollectorConfig = CollectorConfig{
	RefreshInterval: DefaultRefreshInterval,
	CollectInterval: DefaultCollectInterval,
}

type LockConfig struct {
	LockTTL             time.Duration `yaml:"lock_ttl"`
	LockRetryInterval   time.Duration `yaml:"lock_retry_interval"`
	ConsulClusterConfig string        `yaml:"consul_cluster_config"`
}

var defaultLockConfig = LockConfig{
	LockTTL:           DefaultLockTTL,
	LockRetryInterval: DefaultRetryInterval,
}

type Config struct {
	Cf        cf.CfConfig     `yaml:"cf"`
	Logging   LoggingConfig   `yaml:"logging"`
	Server    ServerConfig    `yaml:"server"`
	Db        DbConfig        `yaml:"db"`
	Collector CollectorConfig `yaml:"collector"`
	Lock      LockConfig      `yaml:"lock"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Cf:        defaultCfConfig,
		Logging:   defaultLoggingConfig,
		Server:    defaultServerConfig,
		Collector: defaultCollectorConfig,
		Lock:      defaultLockConfig,
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

	if c.Db.InstanceMetricsDbUrl == "" {
		return fmt.Errorf("Configuration error: InstanceMetrics DB url is empty")
	}

	if c.Collector.CollectInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: CollectInterval is 0")
	}

	if c.Collector.RefreshInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: RefreshInterval is 0")
	}

	return nil

}
