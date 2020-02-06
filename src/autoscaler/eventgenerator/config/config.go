package config

import (
	"fmt"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
)

const (
	DefaultLoggingLevel                   string        = "info"
	DefaultServerPort                     int           = 8080
	DefaultHealthServerPort               int           = 8081
	DefaultPolicyPollerInterval           time.Duration = 40 * time.Second
	DefaultAggregatorExecuteInterval      time.Duration = 40 * time.Second
	DefaultSaveInterval                   time.Duration = 5 * time.Second
	DefaultMetricPollerCount              int           = 20
	DefaultAppMonitorChannelSize          int           = 200
	DefaultAppMetricChannelSize           int           = 200
	DefaultEvaluationExecuteInterval      time.Duration = 40 * time.Second
	DefaultEvaluatorCount                 int           = 20
	DefaultTriggerArrayChannelSize        int           = 200
	DefaultBackOffInitialInterval         time.Duration = 5 * time.Minute
	DefaultBackOffMaxInterval             time.Duration = 2 * time.Hour
	DefaultBreakerConsecutiveFailureCount int64         = 3
	DefaultHttpClientTimeout              time.Duration = 5 * time.Second
	DefaultMetricCacheSizePerApp                        = 100
)

type ServerConfig struct {
	Port      int             `yaml:"port"`
	TLS       models.TLSCerts `yaml:"tls"`
	NodeAddrs []string        `yaml:"node_addrs"`
	NodeIndex int             `yaml:"node_index"`
}
type DBConfig struct {
	PolicyDB    db.DatabaseConfig `yaml:"policy_db"`
	AppMetricDB db.DatabaseConfig `yaml:"app_metrics_db"`
}

type AggregatorConfig struct {
	MetricPollerCount         int           `yaml:"metric_poller_count"`
	AppMonitorChannelSize     int           `yaml:"app_monitor_channel_size"`
	AppMetricChannelSize      int           `yaml:"app_metric_channel_size"`
	AggregatorExecuteInterval time.Duration `yaml:"aggregator_execute_interval"`
	PolicyPollerInterval      time.Duration `yaml:"policy_poller_interval"`
	SaveInterval              time.Duration `yaml:"save_interval"`
	MetricCacheSizePerApp     int           `yaml:"metric_cache_size_per_app"`
}

type EvaluatorConfig struct {
	EvaluatorCount            int           `yaml:"evaluator_count"`
	TriggerArrayChannelSize   int           `yaml:"trigger_array_channel_size"`
	EvaluationManagerInterval time.Duration `yaml:"evaluation_manager_execute_interval"`
}

type ScalingEngineConfig struct {
	ScalingEngineURL string          `yaml:"scaling_engine_url"`
	TLSClientCerts   models.TLSCerts `yaml:"tls"`
}

type MetricCollectorConfig struct {
	MetricCollectorURL string          `yaml:"metric_collector_url"`
	TLSClientCerts     models.TLSCerts `yaml:"tls"`
}

type CircuitBreakerConfig struct {
	BackOffInitialInterval  time.Duration `yaml:"back_off_initial_interval"`
	BackOffMaxInterval      time.Duration `yaml:"back_off_max_interval"`
	ConsecutiveFailureCount int64         `yaml:"consecutive_failure_count"`
}
type Config struct {
	Logging                   helpers.LoggingConfig `yaml:"logging"`
	Server                    ServerConfig          `yaml:"server"`
	Health                    models.HealthConfig   `yaml:"health"`
	DB                        DBConfig              `yaml:"db"`
	Aggregator                AggregatorConfig      `yaml:"aggregator"`
	Evaluator                 EvaluatorConfig       `yaml:"evaluator"`
	ScalingEngine             ScalingEngineConfig   `yaml:"scalingEngine"`
	MetricCollector           MetricCollectorConfig `yaml:"metricCollector"`
	DefaultStatWindowSecs     int                   `yaml:"defaultStatWindowSecs"`
	DefaultBreachDurationSecs int                   `yaml:"defaultBreachDurationSecs"`
	CircuitBreaker            CircuitBreakerConfig  `yaml:"circuitBreaker"`
	HttpClientTimeout         time.Duration         `yaml:"http_client_timeout"`
}

func LoadConfig(bytes []byte) (*Config, error) {
	conf := &Config{
		Logging: helpers.LoggingConfig{
			Level: DefaultLoggingLevel,
		},
		Server: ServerConfig{
			Port: DefaultServerPort,
		},
		Health: models.HealthConfig{
			Port: DefaultHealthServerPort,
		},
		Aggregator: AggregatorConfig{
			AggregatorExecuteInterval: DefaultAggregatorExecuteInterval,
			PolicyPollerInterval:      DefaultPolicyPollerInterval,
			SaveInterval:              DefaultSaveInterval,
			MetricPollerCount:         DefaultMetricPollerCount,
			AppMonitorChannelSize:     DefaultAppMonitorChannelSize,
			AppMetricChannelSize:      DefaultAppMetricChannelSize,
			MetricCacheSizePerApp:     DefaultMetricCacheSizePerApp,
		},
		Evaluator: EvaluatorConfig{
			EvaluationManagerInterval: DefaultEvaluationExecuteInterval,
			EvaluatorCount:            DefaultEvaluatorCount,
			TriggerArrayChannelSize:   DefaultTriggerArrayChannelSize,
		},
		HttpClientTimeout: DefaultHttpClientTimeout,
	}
	err := yaml.Unmarshal(bytes, &conf)
	if err != nil {
		return nil, err
	}

	conf.Logging.Level = strings.ToLower(conf.Logging.Level)
	if conf.CircuitBreaker.ConsecutiveFailureCount == 0 {
		conf.CircuitBreaker.ConsecutiveFailureCount = DefaultBreakerConsecutiveFailureCount
	}
	if conf.CircuitBreaker.BackOffInitialInterval == 0 {
		conf.CircuitBreaker.BackOffInitialInterval = DefaultBackOffInitialInterval
	}
	if conf.CircuitBreaker.BackOffMaxInterval == 0 {
		conf.CircuitBreaker.BackOffMaxInterval = DefaultBackOffMaxInterval
	}
	return conf, nil
}

func (c *Config) Validate() error {
	if c.DB.PolicyDB.URL == "" {
		return fmt.Errorf("Configuration error: db.policy_db.url is empty")
	}
	if c.DB.AppMetricDB.URL == "" {
		return fmt.Errorf("Configuration error: db.app_metrics_db.url is empty")
	}
	if c.ScalingEngine.ScalingEngineURL == "" {
		return fmt.Errorf("Configuration error: scalingEngine.scaling_engine_url is empty")
	}
	if c.MetricCollector.MetricCollectorURL == "" {
		return fmt.Errorf("Configuration error: metricCollector.metric_collector_url is empty")
	}
	if c.Aggregator.AggregatorExecuteInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: aggregator.aggregator_execute_interval is less-equal than 0")
	}
	if c.Aggregator.PolicyPollerInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: aggregator.policy_poller_interval is less-equal than 0")
	}
	if c.Aggregator.SaveInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: aggregator.save_interval is less-equal than 0")
	}
	if c.Aggregator.MetricPollerCount <= 0 {
		return fmt.Errorf("Configuration error: aggregator.metric_poller_count is less-equal than 0")
	}
	if c.Aggregator.AppMonitorChannelSize <= 0 {
		return fmt.Errorf("Configuration error: aggregator.app_monitor_channel_size is less-equal than 0")
	}
	if c.Aggregator.AppMetricChannelSize <= 0 {
		return fmt.Errorf("Configuration error: aggregator.app_metric_channel_size is less-equal than 0")
	}

	if c.Aggregator.MetricCacheSizePerApp <= 0 {
		return fmt.Errorf("Configuration error: aggregator.metric_cache_size_per_app is less-equal than 0")
	}

	if c.Evaluator.EvaluationManagerInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: evaluator.evaluation_manager_execute_interval is less-equal than 0")
	}
	if c.Evaluator.EvaluatorCount <= 0 {
		return fmt.Errorf("Configuration error: evaluator.evaluator_count is less-equal than 0")
	}
	if c.Evaluator.TriggerArrayChannelSize <= 0 {
		return fmt.Errorf("Configuration error: evaluator.trigger_array_channel_size is less-equal than 0")
	}
	if c.DefaultBreachDurationSecs < 60 || c.DefaultBreachDurationSecs > 3600 {
		return fmt.Errorf("Configuration error: defaultBreachDurationSecs should be between 60 and 3600")
	}
	if c.DefaultStatWindowSecs < 60 || c.DefaultStatWindowSecs > 3600 {
		return fmt.Errorf("Configuration error: defaultStatWindowSecs should be between 60 and 3600")
	}

	if (c.Server.NodeIndex >= len(c.Server.NodeAddrs)) || (c.Server.NodeIndex < 0) {
		return fmt.Errorf("Configuration error: server.node_index out of range")
	}

	if c.HttpClientTimeout <= time.Duration(0) {
		return fmt.Errorf("Configuration error: http_client_timeout is less-equal than 0")
	}

	if err := c.Health.Validate("eventgenerator"); err != nil {
		return err
	}

	return nil
}
