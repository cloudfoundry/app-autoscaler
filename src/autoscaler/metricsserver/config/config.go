package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
)

const (
	DefaultLoggingLevel                  = "info"
	DefaultHttpClientTimeout             = 5 * time.Second
	DefaultWSKeepAliveTime               = 1 * time.Minute
	DefaultWSPort                        = 8443
	DefaultRefreshInterval               = 60 * time.Second
	DefaultCollectInterval               = 30 * time.Second
	DefaultSaveInterval                  = 5 * time.Second
	DefaultMetricCacheSizePerApp         = 1000
	DefaultIsMetricsPersistencySupported = true
	DefaultEnvelopeProcessorCount        = 5
	DefaultEnvelopeChannelSize           = 1000
	DefaultMetricChannelSize             = 1000
	DefaultHTTPServerPort                = 8080
	DefaultHealthPort                    = 8081
)

type DBConfig struct {
	PolicyDB          db.DatabaseConfig `yaml:"policy_db"`
	InstanceMetricsDB db.DatabaseConfig `yaml:"instance_metrics_db"`
}

type CollectorConfig struct {
	WSPort                 int             `yaml:"port"`
	WSKeepAliveTime        time.Duration   `yaml:"keep_alive_time"`
	TLS                    models.TLSCerts `yaml:"tls"`
	RefreshInterval        time.Duration   `yaml:"refresh_interval"`
	CollectInterval        time.Duration   `yaml:"collect_interval"`
	SaveInterval           time.Duration   `yaml:"save_interval"`
	MetricCacheSizePerApp  int             `yaml:"metric_cache_size_per_app"`
	PersistMetrics         bool            `yaml:"persist_metrics"`
	EnvelopeProcessorCount int             `yaml:"envelope_processor_count"`
	EnvelopeChannelSize    int             `yaml:"envelope_channel_size"`
	MetricChannelSize      int             `yaml:"metric_channel_size"`
}

type ServerConfig struct {
	Port int             `yaml:"port"`
	TLS  models.TLSCerts `yaml:"tls"`
}

type Config struct {
	Logging           helpers.LoggingConfig `yaml:"logging"`
	HttpClientTimeout time.Duration         `yaml:"http_client_timeout"`
	NodeAddrs         []string              `yaml:"node_addrs"`
	NodeIndex         int                   `yaml:"node_index"`
	DB                DBConfig              `yaml:"db"`
	Collector         CollectorConfig       `yaml:"collector"`
	Server            ServerConfig          `yaml:"server"`
	Health            models.HealthConfig   `yaml:"health"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Logging: helpers.LoggingConfig{
			Level: DefaultLoggingLevel,
		},
		HttpClientTimeout: DefaultHttpClientTimeout,
		Health: models.HealthConfig{
			Port: DefaultHealthPort,
		},
		Collector: CollectorConfig{
			WSPort:                 DefaultWSPort,
			WSKeepAliveTime:        DefaultWSKeepAliveTime,
			RefreshInterval:        DefaultRefreshInterval,
			CollectInterval:        DefaultCollectInterval,
			SaveInterval:           DefaultSaveInterval,
			MetricCacheSizePerApp:  DefaultMetricCacheSizePerApp,
			PersistMetrics:         DefaultIsMetricsPersistencySupported,
			EnvelopeProcessorCount: DefaultEnvelopeProcessorCount,
			EnvelopeChannelSize:    DefaultEnvelopeChannelSize,
			MetricChannelSize:      DefaultMetricChannelSize,
		},
		Server: ServerConfig{
			Port: DefaultHTTPServerPort,
		},
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
	if c.HttpClientTimeout <= time.Duration(0) {
		return fmt.Errorf("Configuration error: http_client_timeout is less-equal than 0")
	}

	if (c.NodeIndex >= len(c.NodeAddrs)) || (c.NodeIndex < 0) {
		return fmt.Errorf("Configuration error: node_index out of range")
	}

	if c.DB.PolicyDB.URL == "" {
		return fmt.Errorf("Configuration error: db.policy_db.url is empty")
	}

	if c.DB.InstanceMetricsDB.URL == "" {
		return fmt.Errorf("Configuration error: db.instance_metrics_db.url is empty")
	}

	if c.Collector.WSKeepAliveTime == time.Duration(0) {
		return fmt.Errorf("Configuration error: keep_alive_time is less-equal than 0")
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

	if c.Collector.EnvelopeProcessorCount <= 0 {
		return fmt.Errorf("Configuration error: envelope_processor_count is less-equal than 0")
	}

	if c.Collector.EnvelopeChannelSize <= 0 {
		return fmt.Errorf("Configuration error: envelope_channel_size is less-equal than 0")
	}

	if c.Collector.MetricChannelSize <= 0 {
		return fmt.Errorf("Configuration error: metric_channel_size is less-equal than 0")
	}

	if err := c.Health.Validate("metricsserver"); err != nil {
		return err
	}

	return nil
}
