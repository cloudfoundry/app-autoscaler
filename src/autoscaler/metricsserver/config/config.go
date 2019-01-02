package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
)

const (
	DefaultLoggingLevel                                = "info"
	DefaultRefreshInterval               time.Duration = 60 * time.Second
	DefaultCollectInterval               time.Duration = 30 * time.Second
	DefaultSaveInterval                  time.Duration = 5 * time.Second
	DefaultHttpClientTimeout             time.Duration = 5 * time.Second
	DefaultMetricCacheSizePerApp                       = 1000
	DefaultIsMetricsPersistencySupported               = true
	DefaultKeepAliveTime                 time.Duration = 3 * time.Minute
	DefaultEnvelopeProcessorCount                      = 5
)

type ServerConfig struct {
	Port      int             `yaml:"port"`
	TLS       models.TLSCerts `yaml:"tls"`
	NodeAddrs []string        `yaml:"node_addrs"`
	NodeIndex int             `yaml:"node_index"`
}

var defaultServerConfig = ServerConfig{
	Port: 8080,
}
var defaultHealthConfig = models.HealthConfig{
	Port: 8081,
}
var defaultLoggingConfig = helpers.LoggingConfig{
	Level: DefaultLoggingLevel,
}

type DBConfig struct {
	PolicyDB          db.DatabaseConfig `yaml:"policy_db"`
	InstanceMetricsDB db.DatabaseConfig `yaml:"instance_metrics_db"`
}

type CollectorConfig struct {
	RefreshInterval               time.Duration `yaml:"refresh_interval"`
	CollectInterval               time.Duration `yaml:"collect_interval"`
	CollectMethod                 string        `yaml:"collect_method"`
	SaveInterval                  time.Duration `yaml:"save_interval"`
	MetricCacheSizePerApp         int           `yaml:"metric_cache_size_per_app"`
	IsMetricsPersistencySupported bool          `yaml:"metrics_persistency_support_flag"`
	KeepAliveTime                 time.Duration `yaml:"keep_alive_time"`
	EnvelopeProcessorCount        int           `yaml:"envelope_processor_count"`
}

var defaultCollectorConfig = CollectorConfig{
	RefreshInterval:               DefaultRefreshInterval,
	CollectInterval:               DefaultCollectInterval,
	SaveInterval:                  DefaultSaveInterval,
	MetricCacheSizePerApp:         DefaultMetricCacheSizePerApp,
	IsMetricsPersistencySupported: DefaultIsMetricsPersistencySupported,
	KeepAliveTime:                 DefaultKeepAliveTime,
	EnvelopeProcessorCount:        DefaultEnvelopeProcessorCount,
}

type Config struct {
	Logging           helpers.LoggingConfig `yaml:"logging"`
	Server            ServerConfig          `yaml:"server"`
	Health            models.HealthConfig   `yaml:"health"`
	DB                DBConfig              `yaml:"db"`
	Collector         CollectorConfig       `yaml:"collector"`
	HttpClientTimeout time.Duration         `yaml:"http_client_timeout"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Logging:           defaultLoggingConfig,
		Server:            defaultServerConfig,
		Health:            defaultHealthConfig,
		Collector:         defaultCollectorConfig,
		HttpClientTimeout: DefaultHttpClientTimeout,
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

	if c.DB.PolicyDB.URL == "" {
		return fmt.Errorf("Configuration error: db.policy_db.url is empty")
	}

	if c.DB.InstanceMetricsDB.URL == "" {
		return fmt.Errorf("Configuration error: db.instance_metrics_db.url is empty")
	}

	if c.Collector.CollectInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: collector.collect_interval is 0")
	}

	if c.Collector.RefreshInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: collector.refresh_interval is 0")
	}

	if c.Collector.SaveInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: collector.save_interval is 0")
	}

	if c.Collector.MetricCacheSizePerApp <= 0 {
		return fmt.Errorf("Configuration error: invalid collector.metric_cache_size_per_app")
	}

	if (c.Server.NodeIndex >= len(c.Server.NodeAddrs)) || (c.Server.NodeIndex < 0) {
		return fmt.Errorf("Configuration error: server.node_index out of range")
	}

	if c.HttpClientTimeout <= time.Duration(0) {
		return fmt.Errorf("Configuration error: http_client_timeout is less-equal than 0")
	}

	if c.Collector.KeepAliveTime <= time.Duration(0) {
		return fmt.Errorf("Configuration error: keep_alive_time is less-equal than 0")
	}

	if c.Collector.EnvelopeProcessorCount <= 0 {
		return fmt.Errorf("Configuration error: envelope_processor_count is less-equal than 0")
	}

	return nil

}
