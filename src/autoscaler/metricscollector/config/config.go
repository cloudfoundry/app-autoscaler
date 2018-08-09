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
	"autoscaler/helpers"
	"autoscaler/models"
)

const (
	DefaultLoggingLevel                  = "info"
	DefaultRefreshInterval time.Duration = 60 * time.Second
	DefaultCollectInterval time.Duration = 30 * time.Second
	DefaultSaveInterval    time.Duration = 5 * time.Second

	CollectMethodPolling   = "polling"
	CollectMethodStreaming = "streaming"
)

var defaultCfConfig = cf.CfConfig{
	GrantType:         cf.GrantTypePassword,
	SkipSSLValidation: false,
}

type ServerConfig struct {
	Port      int             `yaml:"port"`
	TLS       models.TLSCerts `yaml:"tls"`
	NodeAddrs []string        `yaml:"node_addrs"`
	NodeIndex int             `yaml:"node_index"`
}

var defaultServerConfig = ServerConfig{
	Port: 8080,
}

var defaultLoggingConfig = helpers.LoggingConfig{
	Level: DefaultLoggingLevel,
}

type DbConfig struct {
	PolicyDb          db.DatabaseConfig `yaml:"policy_db"`
	InstanceMetricsDb db.DatabaseConfig `yaml:"instance_metrics_db"`
}

type CollectorConfig struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	CollectInterval time.Duration `yaml:"collect_interval"`
	CollectMethod   string        `yaml:"collect_method"`
	SaveInterval    time.Duration `yaml:"save_interval"`
}

var defaultCollectorConfig = CollectorConfig{
	RefreshInterval: DefaultRefreshInterval,
	CollectInterval: DefaultCollectInterval,
	CollectMethod:   CollectMethodStreaming,
	SaveInterval:    DefaultSaveInterval,
}

type Config struct {
	Cf        cf.CfConfig           `yaml:"cf"`
	Logging   helpers.LoggingConfig `yaml:"logging"`
	Server    ServerConfig          `yaml:"server"`
	Db        DbConfig              `yaml:"db"`
	Collector CollectorConfig       `yaml:"collector"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Cf:        defaultCfConfig,
		Logging:   defaultLoggingConfig,
		Server:    defaultServerConfig,
		Collector: defaultCollectorConfig,
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
	conf.Collector.CollectMethod = strings.ToLower(conf.Collector.CollectMethod)

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

	if c.Db.InstanceMetricsDb.Url == "" {
		return fmt.Errorf("Configuration error: InstanceMetrics DB url is empty")
	}

	if c.Collector.CollectInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: CollectInterval is 0")
	}

	if c.Collector.RefreshInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: RefreshInterval is 0")
	}

	if c.Collector.SaveInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: SaveInterval is 0")
	}

	if (c.Collector.CollectMethod != CollectMethodPolling) && (c.Collector.CollectMethod != CollectMethodStreaming) {
		return fmt.Errorf("Configuration error: invalid collecting method")
	}

	if (c.Server.NodeIndex >= len(c.Server.NodeAddrs)) || (c.Server.NodeIndex < 0) {
		return fmt.Errorf("Configuration error: node_index out of range")
	}
	return nil

}
