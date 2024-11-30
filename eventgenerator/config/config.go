package config

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	DefaultLoggingLevel                   = "info"
	DefaultServerPort                     = 8080
	DefaultHealthServerPort               = 8081
	DefaultPolicyPollerInterval           = 40 * time.Second
	DefaultAggregatorExecuteInterval      = 40 * time.Second
	DefaultSaveInterval                   = 5 * time.Second
	DefaultMetricPollerCount              = 20
	DefaultAppMonitorChannelSize          = 200
	DefaultAppMetricChannelSize           = 200
	DefaultEvaluationExecuteInterval      = 40 * time.Second
	DefaultEvaluatorCount                 = 20
	DefaultTriggerArrayChannelSize        = 200
	DefaultBackOffInitialInterval         = 5 * time.Minute
	DefaultBackOffMaxInterval             = 2 * time.Hour
	DefaultBreakerConsecutiveFailureCount = 3
	DefaultHttpClientTimeout              = 5 * time.Second
	DefaultMetricCacheSizePerApp          = 100
)

var defaultCFServerConfig = helpers.ServerConfig{
	Port: 8082,
}

type ServerConfig struct {
	helpers.ServerConfig `yaml:",inline"`
	NodeAddrs            []string `yaml:"node_addrs"`
	NodeIndex            int      `yaml:"node_index"`
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
	UAACreds           models.UAACreds `yaml:"uaa"`
}

type CircuitBreakerConfig struct {
	BackOffInitialInterval  time.Duration `yaml:"back_off_initial_interval"`
	BackOffMaxInterval      time.Duration `yaml:"back_off_max_interval"`
	ConsecutiveFailureCount int64         `yaml:"consecutive_failure_count"`
}

type Config struct {
	Logging                   helpers.LoggingConfig `yaml:"logging"`
	Server                    ServerConfig          `yaml:"server"`
	CFServer                  helpers.ServerConfig  `yaml:"cf_server"`
	Health                    helpers.HealthConfig  `yaml:"health"`
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

func LoadConfig(config []byte) (*Config, error) {
	conf := defaultConfig()
	dec := yaml.NewDecoder(bytes.NewBuffer(config))
	dec.KnownFields(true)
	if err := dec.Decode(conf); err != nil {
		return nil, err
	}
	setDefaults(conf)
	return conf, nil
}

func defaultConfig() *Config {
	return &Config{
		Logging: helpers.LoggingConfig{
			Level: DefaultLoggingLevel,
		},
		Server: ServerConfig{
			ServerConfig: helpers.ServerConfig{
				Port: DefaultServerPort,
			},
		},
		CFServer: defaultCFServerConfig,
		Health: helpers.HealthConfig{
			ServerConfig: helpers.ServerConfig{
				Port: DefaultHealthServerPort,
			},
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
}

func setDefaults(conf *Config) {
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
}

func (c *Config) Validate() error {
	if err := c.validateDB(); err != nil {
		return err
	}
	if err := c.validateScalingEngine(); err != nil {
		return err
	}
	if err := c.validateMetricCollector(); err != nil {
		return err
	}
	if err := c.validateAggregator(); err != nil {
		return err
	}
	if err := c.validateEvaluator(); err != nil {
		return err
	}
	if err := c.validateDefaults(); err != nil {
		return err
	}
	if err := c.validateServer(); err != nil {
		return err
	}
	if err := c.validateHealth(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateDB() error {
	if c.DB.PolicyDB.URL == "" {
		return fmt.Errorf("Configuration error: db.policy_db.url is empty")
	}
	if c.DB.AppMetricDB.URL == "" {
		return fmt.Errorf("Configuration error: db.app_metrics_db.url is empty")
	}
	return nil
}

func (c *Config) validateScalingEngine() error {
	if c.ScalingEngine.ScalingEngineURL == "" {
		return fmt.Errorf("Configuration error: scalingEngine.scaling_engine_url is empty")
	}
	return nil
}

func (c *Config) validateMetricCollector() error {
	if c.MetricCollector.MetricCollectorURL == "" {
		return fmt.Errorf("Configuration error: metricCollector.metric_collector_url is empty")
	}
	return nil
}

func (c *Config) validateAggregator() error {
	if c.Aggregator.AggregatorExecuteInterval <= 0 {
		return fmt.Errorf("Configuration error: aggregator.aggregator_execute_interval is less-equal than 0")
	}
	if c.Aggregator.PolicyPollerInterval <= 0 {
		return fmt.Errorf("Configuration error: aggregator.policy_poller_interval is less-equal than 0")
	}
	if c.Aggregator.SaveInterval <= 0 {
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
	return nil
}

func (c *Config) validateEvaluator() error {
	if c.Evaluator.EvaluationManagerInterval <= 0 {
		return fmt.Errorf("Configuration error: evaluator.evaluation_manager_execute_interval is less-equal than 0")
	}
	if c.Evaluator.EvaluatorCount <= 0 {
		return fmt.Errorf("Configuration error: evaluator.evaluator_count is less-equal than 0")
	}
	if c.Evaluator.TriggerArrayChannelSize <= 0 {
		return fmt.Errorf("Configuration error: evaluator.trigger_array_channel_size is less-equal than 0")
	}
	return nil
}

func (c *Config) validateDefaults() error {
	if c.DefaultBreachDurationSecs < 60 || c.DefaultBreachDurationSecs > 3600 {
		return fmt.Errorf("Configuration error: defaultBreachDurationSecs should be between 60 and 3600")
	}
	if c.DefaultStatWindowSecs < 60 || c.DefaultStatWindowSecs > 3600 {
		return fmt.Errorf("Configuration error: defaultStatWindowSecs should be between 60 and 3600")
	}
	if c.HttpClientTimeout <= 0 {
		return fmt.Errorf("Configuration error: http_client_timeout is less-equal than 0")
	}
	return nil
}

func (c *Config) validateServer() error {
	if c.Server.NodeIndex < 0 || c.Server.NodeIndex >= len(c.Server.NodeAddrs) {
		return fmt.Errorf("Configuration error: server.node_index out of range")
	}
	return nil
}

func (c *Config) validateHealth() error {
	return c.Health.Validate()
}
