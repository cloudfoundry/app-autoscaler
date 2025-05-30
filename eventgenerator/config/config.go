package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

var (
	ErrReadYaml                     = helpers.ErrReadYaml
	ErrReadJson                     = errors.New("failed to read vcap_services json")
	ErrEventgeneratorConfigNotFound = errors.New("eventgenerator config service not found")
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
	DefaultMetricCacheSizePerApp          = 100
)

var DefaultHttpClientTimeout = 5 * time.Second

var defaultCFServerConfig = helpers.ServerConfig{
	Port: 8082,
}

type PoolConfig struct {
	TotalInstances int `yaml:"total_instances" json:"total_instances"`
	InstanceIndex  int `yaml:"instance_index" json:"instance_index"`
}

type AggregatorConfig struct {
	MetricPollerCount         int           `yaml:"metric_poller_count" json:"metric_poller_count"`
	AppMonitorChannelSize     int           `yaml:"app_monitor_channel_size" json:"app_monitor_channel_size"`
	AppMetricChannelSize      int           `yaml:"app_metric_channel_size" json:"app_metric_channel_size"`
	AggregatorExecuteInterval time.Duration `yaml:"aggregator_execute_interval" json:"aggregator_execute_interval"`
	PolicyPollerInterval      time.Duration `yaml:"policy_poller_interval" json:"policy_poller_interval"`
	SaveInterval              time.Duration `yaml:"save_interval" json:"save_interval"`
	MetricCacheSizePerApp     int           `yaml:"metric_cache_size_per_app" json:"metric_cache_size_per_app"`
}

type EvaluatorConfig struct {
	EvaluatorCount            int           `yaml:"evaluator_count" json:"evaluator_count"`
	TriggerArrayChannelSize   int           `yaml:"trigger_array_channel_size" json:"trigger_array_channel_size"`
	EvaluationManagerInterval time.Duration `yaml:"evaluation_manager_execute_interval" json:"evaluation_manager_execute_interval"`
}

type ScalingEngineConfig struct {
	ScalingEngineURL string          `yaml:"scaling_engine_url" json:"scaling_engine_url"`
	TLSClientCerts   models.TLSCerts `yaml:"tls" json:"tls"`
}

type MetricCollectorConfig struct {
	MetricCollectorURL string          `yaml:"metric_collector_url" json:"metric_collector_url"`
	TLSClientCerts     models.TLSCerts `yaml:"tls" json:"tls"`
	UAACreds           models.UAACreds `yaml:"uaa" json:"uaa"`
}

type CircuitBreakerConfig struct {
	BackOffInitialInterval  time.Duration `yaml:"back_off_initial_interval" json:"back_off_initial_interval"`
	BackOffMaxInterval      time.Duration `yaml:"back_off_max_interval" json:"back_off_max_interval"`
	ConsecutiveFailureCount int64         `yaml:"consecutive_failure_count" json:"consecutive_failure_count"`
}

type Config struct {
	Logging                   helpers.LoggingConfig        `yaml:"logging" json:"logging"`
	Server                    helpers.ServerConfig         `yaml:"server" json:"server"`
	CFServer                  helpers.ServerConfig         `yaml:"cf_server" json:"cf_server"`
	Pool                      *PoolConfig                  `yaml:"pool" json:"pool"`
	Health                    helpers.HealthConfig         `yaml:"health" json:"health"`
	Db                        map[string]db.DatabaseConfig `yaml:"db" json:"db"`
	Aggregator                *AggregatorConfig            `yaml:"aggregator" json:"aggregator,omitempty"`
	Evaluator                 *EvaluatorConfig             `yaml:"evaluator" json:"evaluator,omitempty"`
	ScalingEngine             ScalingEngineConfig          `yaml:"scalingEngine" json:"scalingEngine"`
	MetricCollector           MetricCollectorConfig        `yaml:"metricCollector" json:"metricCollector"`
	DefaultStatWindowSecs     int                          `yaml:"defaultStatWindowSecs" json:"defaultStatWindowSecs"`
	DefaultBreachDurationSecs int                          `yaml:"defaultBreachDurationSecs" json:"defaultBreachDurationSecs"`
	CircuitBreaker            *CircuitBreakerConfig        `yaml:"circuitBreaker,omitempty" json:"circuitBreaker,omitempty"`
	HttpClientTimeout         *time.Duration               `yaml:"http_client_timeout,omitempty" json:"http_client_timeout,omitempty"`
}

func LoadConfig(filepath string, vcapReader configutil.VCAPConfigurationReader) (*Config, error) {
	conf := defaultConfig()

	if err := helpers.LoadYamlFile(filepath, &conf); err != nil {
		return nil, err
	}

	if err := loadVcapConfig(&conf, vcapReader); err != nil {
		return nil, err
	}

	return &conf, nil
}

func loadVcapConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	if !vcapReader.IsRunningOnCF() {
		return nil
	}
	// enable plain text logging. See src/autoscaler/helpers/logger.go
	conf.Logging.PlainTextSink = true

	// Avoid port conflict: assign actual port to CF server, set BOSH server port to 0 (unused)
	conf.CFServer.Port = vcapReader.GetPort()
	conf.Server.Port = 0

	if err := loadEventgeneratorConfig(conf, vcapReader); err != nil {
		return err
	}

	if err := vcapReader.ConfigureDatabases(&conf.Db, nil, ""); err != nil {
		return err
	}

	if err := configureInstanceIndex(conf, vcapReader); err != nil {
		return err
	}

	if err := configureXfccSpaceAndOrg(conf, vcapReader); err != nil {
		return err
	}

	conf.ScalingEngine.TLSClientCerts = vcapReader.GetInstanceTLSCerts()

	return nil
}

func loadEventgeneratorConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	var raw string
	data, err := vcapReader.GetServiceCredentialContent("eventgenerator-config", "eventgenerator-config")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEventgeneratorConfigNotFound, err)
	}

	// removes the first and last double quotes if they exist
	if json.Unmarshal(data, &raw) == nil {
		return yaml.Unmarshal([]byte(raw), conf)
	} else {
		return yaml.Unmarshal(data, conf)
	}
}

func configureXfccSpaceAndOrg(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	conf.CFServer.XFCC.ValidSpaceGuid = vcapReader.GetSpaceGuid()
	conf.CFServer.XFCC.ValidOrgGuid = vcapReader.GetOrgGuid()

	return nil
}

func configureInstanceIndex(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	conf.Pool.InstanceIndex = vcapReader.GetInstanceIndex()
	return nil
}

func defaultConfig() Config {
	return Config{
		Logging: helpers.LoggingConfig{
			Level: DefaultLoggingLevel,
		},
		Server: helpers.ServerConfig{
			Port: DefaultServerPort,
		},
		Pool: &PoolConfig{},
		Db:   make(map[string]db.DatabaseConfig),
		CircuitBreaker: &CircuitBreakerConfig{
			BackOffInitialInterval:  DefaultBackOffInitialInterval,
			BackOffMaxInterval:      DefaultBackOffMaxInterval,
			ConsecutiveFailureCount: DefaultBreakerConsecutiveFailureCount,
		},
		CFServer: defaultCFServerConfig,
		Health: helpers.HealthConfig{
			ServerConfig: helpers.ServerConfig{
				Port: DefaultHealthServerPort,
			},
		},
		Aggregator: &AggregatorConfig{
			AggregatorExecuteInterval: DefaultAggregatorExecuteInterval,
			PolicyPollerInterval:      DefaultPolicyPollerInterval,
			SaveInterval:              DefaultSaveInterval,
			MetricPollerCount:         DefaultMetricPollerCount,
			AppMonitorChannelSize:     DefaultAppMonitorChannelSize,
			AppMetricChannelSize:      DefaultAppMetricChannelSize,
			MetricCacheSizePerApp:     DefaultMetricCacheSizePerApp,
		},
		Evaluator: &EvaluatorConfig{
			EvaluationManagerInterval: DefaultEvaluationExecuteInterval,
			EvaluatorCount:            DefaultEvaluatorCount,
			TriggerArrayChannelSize:   DefaultTriggerArrayChannelSize,
		},
		HttpClientTimeout: &DefaultHttpClientTimeout,
	}
}

func (c *Config) Validate() error {
	if err := c.validateDb(); err != nil {
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
	if err := c.validatePool(); err != nil {
		return err
	}
	if err := c.validateHealth(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateDb() error {
	if c.Db[db.PolicyDb].URL == "" {
		return fmt.Errorf("Configuration error: db.policy_db.url is empty")
	}
	if c.Db[db.AppMetricsDb].URL == "" {
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
	if *c.HttpClientTimeout <= 0 {
		return fmt.Errorf("Configuration error: http_client_timeout is less-equal than 0")
	}
	return nil
}

func (c *Config) validatePool() error {
	if c.Pool.InstanceIndex < 0 || c.Pool.InstanceIndex >= c.Pool.TotalInstances {
		return fmt.Errorf("Configuration error: pool.instance_index out of range")
	}
	return nil
}

func (c *Config) validateHealth() error {
	return c.Health.Validate()
}
