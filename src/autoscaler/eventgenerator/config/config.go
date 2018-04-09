package config

import (
	"fmt"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"code.cloudfoundry.org/locket"

	"autoscaler/models"
)

const (
	DefaultLoggingLevel                   string        = "info"
	DefaultPolicyPollerInterval           time.Duration = 40 * time.Second
	DefaultAggregatorExecuteInterval      time.Duration = 40 * time.Second
	DefaultSaveInterval                   time.Duration = 5 * time.Second
	DefaultMetricPollerCount              int           = 20
	DefaultAppMonitorChannelSize          int           = 200
	DefaultAppMetricChannelSize           int           = 200
	DefaultEvaluationExecuteInterval      time.Duration = 40 * time.Second
	DefaultEvaluatorCount                 int           = 20
	DefaultTriggerArrayChannelSize        int           = 200
	DefaultLockTTL                        time.Duration = locket.DefaultSessionTTL
	DefaultRetryInterval                  time.Duration = locket.RetryInterval
	DefaultDBLockRetryInterval            time.Duration = 5 * time.Second
	DefaultDBLockTTL                      time.Duration = 15 * time.Second
	DefaultBackOffInitialInterval         time.Duration = 5 * time.Minute
	DefaultBackOffMaxInterval             time.Duration = 2 * time.Hour
	DefaultBreakerConsecutiveFailureCount int64         = 3
)

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type DBConfig struct {
	PolicyDBUrl    string `yaml:"policy_db_url"`
	AppMetricDBUrl string `yaml:"app_metrics_db_url"`
}

type AggregatorConfig struct {
	MetricPollerCount         int           `yaml:"metric_poller_count"`
	AppMonitorChannelSize     int           `yaml:"app_monitor_channel_size"`
	AppMetricChannelSize      int           `yaml:"app_metric_channel_size"`
	AggregatorExecuteInterval time.Duration `yaml:"aggregator_execute_interval"`
	PolicyPollerInterval      time.Duration `yaml:"policy_poller_interval"`
	SaveInterval              time.Duration `yaml:"save_interval"`
}

type EvaluatorConfig struct {
	EvaluatorCount            int           `yaml:"evaluator_count"`
	TriggerArrayChannelSize   int           `yaml:"trigger_array_channel_size"`
	EvaluationManagerInterval time.Duration `yaml:"evaluation_manager_execute_interval"`
}

type ScalingEngineConfig struct {
	ScalingEngineUrl string          `yaml:"scaling_engine_url"`
	TLSClientCerts   models.TLSCerts `yaml:"tls"`
}

type MetricCollectorConfig struct {
	MetricCollectorUrl string          `yaml:"metric_collector_url"`
	TLSClientCerts     models.TLSCerts `yaml:"tls"`
}

type LockConfig struct {
	LockTTL             time.Duration `yaml:"lock_ttl"`
	LockRetryInterval   time.Duration `yaml:"lock_retry_interval"`
	ConsulClusterConfig string        `yaml:"consul_cluster_config"`
}

type DBLockConfig struct {
	LockTTL           time.Duration `yaml:"ttl"`
	LockDBURL         string        `yaml:"url"`
	LockRetryInterval time.Duration `yaml:"retry_interval"`
}

type CircuitBreakerConfig struct {
	BackOffInitialInterval  time.Duration `yaml:"back_off_initial_interval"`
	BackOffMaxInterval      time.Duration `yaml:"back_off_max_interval"`
	ConsecutiveFailureCount int64         `yaml:"consecutive_failure_count"`
}

var defaultDBLockConfig = DBLockConfig{
	LockTTL:           DefaultDBLockTTL,
	LockRetryInterval: DefaultDBLockRetryInterval,
}

type Config struct {
	Logging                   LoggingConfig         `yaml:"logging"`
	DB                        DBConfig              `yaml:"db"`
	Aggregator                AggregatorConfig      `yaml:"aggregator"`
	Evaluator                 EvaluatorConfig       `yaml:"evaluator"`
	ScalingEngine             ScalingEngineConfig   `yaml:"scalingEngine"`
	MetricCollector           MetricCollectorConfig `yaml:"metricCollector"`
	Lock                      LockConfig            `yaml:"lock"`
	DefaultStatWindowSecs     int                   `yaml:"defaultStatWindowSecs"`
	DefaultBreachDurationSecs int                   `yaml:"defaultBreachDurationSecs"`
	DBLock                    DBLockConfig          `yaml:"db_lock"`
	EnableDBLock              bool                  `yaml:"enable_db_lock"`
	CircuitBreaker            CircuitBreakerConfig  `yaml:"circuitBreaker"`
}

func LoadConfig(bytes []byte) (*Config, error) {
	conf := &Config{
		Logging: LoggingConfig{
			Level: DefaultLoggingLevel,
		},
		Aggregator: AggregatorConfig{
			AggregatorExecuteInterval: DefaultAggregatorExecuteInterval,
			PolicyPollerInterval:      DefaultPolicyPollerInterval,
			SaveInterval:              DefaultSaveInterval,
			MetricPollerCount:         DefaultMetricPollerCount,
			AppMonitorChannelSize:     DefaultAppMonitorChannelSize,
			AppMetricChannelSize:      DefaultAppMetricChannelSize,
		},
		Evaluator: EvaluatorConfig{
			EvaluationManagerInterval: DefaultEvaluationExecuteInterval,
			EvaluatorCount:            DefaultEvaluatorCount,
			TriggerArrayChannelSize:   DefaultTriggerArrayChannelSize,
		},
		Lock: LockConfig{
			LockRetryInterval: DefaultRetryInterval,
			LockTTL:           DefaultLockTTL,
		},
		DBLock:       defaultDBLockConfig,
		EnableDBLock: false,
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
	if c.DB.PolicyDBUrl == "" {
		return fmt.Errorf("Configuration error: Policy DB url is empty")
	}
	if c.DB.AppMetricDBUrl == "" {
		return fmt.Errorf("Configuration error: AppMetric DB url is empty")
	}
	if c.ScalingEngine.ScalingEngineUrl == "" {
		return fmt.Errorf("Configuration error: Scaling engine url is empty")
	}
	if c.MetricCollector.MetricCollectorUrl == "" {
		return fmt.Errorf("Configuration error: Metric collector url is empty")
	}
	if c.Aggregator.AggregatorExecuteInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: aggregator execute interval is less-equal than 0")
	}
	if c.Aggregator.PolicyPollerInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: policy poller interval is less-equal than 0")
	}
	if c.Aggregator.SaveInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: save interval is less-equal than 0")
	}
	if c.Aggregator.MetricPollerCount <= 0 {
		return fmt.Errorf("Configuration error: metric poller count is less-equal than 0")
	}
	if c.Aggregator.AppMonitorChannelSize <= 0 {
		return fmt.Errorf("Configuration error: appMonitor channel size is less-equal than 0")
	}
	if c.Aggregator.AppMetricChannelSize <= 0 {
		return fmt.Errorf("Configuration error: appMetric channel size is less-equal than 0")
	}
	if c.Evaluator.EvaluationManagerInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: evalution manager execeute interval is less-equal than 0")
	}
	if c.Evaluator.EvaluatorCount <= 0 {
		return fmt.Errorf("Configuration error: evaluator count is less-equal than 0")
	}
	if c.Evaluator.TriggerArrayChannelSize <= 0 {
		return fmt.Errorf("Configuration error: trigger-array channel size is less-equal than 0")
	}
	if c.Lock.LockRetryInterval <= 0 {
		return fmt.Errorf("Configuration error: lock retry interval is less than or equal to 0")
	}
	if c.Lock.LockTTL <= 0 {
		return fmt.Errorf("Configuration error: lock ttl is less than or equal to 0")
	}
	if c.DefaultBreachDurationSecs < 60 || c.DefaultBreachDurationSecs > 3600 {
		return fmt.Errorf("Configuration error: defaultBreachDurationSecs should be between 60 and 3600")
	}
	if c.DefaultStatWindowSecs < 60 || c.DefaultStatWindowSecs > 3600 {
		return fmt.Errorf("Configuration error: defaultStatWindowSecs should be between 60 and 3600")
	}
	if c.EnableDBLock && c.DBLock.LockDBURL == "" {
		return fmt.Errorf("Configuration error: Lock DB URL is empty")
	}
	return nil

}
