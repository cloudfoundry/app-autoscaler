package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"strings"
	"time"
)

const (
	DefaultServerPort                int           = 8080
	DefaultLoggingLevel                            = "info"
	DefaultPolicyPollerInterval      time.Duration = 40 * time.Second
	DefaultAggregatorExecuteInterval time.Duration = 40 * time.Second
	DefaultMetricPollerCount         int           = 20
	DefaultAppMonitorChannelSize     int           = 200
	DefaultEvaluationExecuteInterval time.Duration = 40 * time.Second
	DefaultEvaluatorCount            int           = 20
	DefaultTriggerArrayChannelSize   int           = 200
)

type ServerConfig struct {
	Port int `yaml:"port"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type DBConfig struct {
	PolicyDBUrl    string `yaml:"policy_db_url"`
	AppMetricDBUrl string `yaml:"app_metrics_db_url"`
}

type AggregatorConfig struct {
	AggregatorExecuateIntervalInSeconds time.Duration `yaml:"aggregator_execute_interval_in_seconds"`
	PolicyPollerIntervalInSeconds       time.Duration `yaml:"policy_poller_interval_in_seconds"`
	MetricPollerCount                   int           `yaml:"metric_poller_count"`
	AppMonitorChannelSize               int           `yaml:"app_monitor_channel_size"`
	AggregatorExecuateInterval          time.Duration
	PolicyPollerInterval                time.Duration
}

type EvaluatorConfig struct {
	EvaluationManagerIntervalInSeconds time.Duration `yaml:"evaluation_manager_execute_interval_in_seconds"`
	EvaluatorCount                     int           `yaml:"evaluator_count"`
	TriggerArrayChannelSize            int           `yaml:"trigger_array_channel_size"`
	EvaluationManagerInterval          time.Duration
}

type ScalingEngineConfig struct {
	ScalingEngineUrl string `yaml:"scaling_engine_url"`
}
type MetricCollectorConfig struct {
	MetricCollectorUrl string `yaml:"metric_collector_url"`
}

type Config struct {
	Server          ServerConfig          `yaml:"server"`
	Logging         LoggingConfig         `yaml:"logging"`
	DB              DBConfig              `yaml:"db"`
	Aggregator      AggregatorConfig      `yaml:"aggregator"`
	Evaluator       EvaluatorConfig       `yaml:"evaluator"`
	ScalingEngine   ScalingEngineConfig   `yaml:"scalingEngine"`
	MetricCollector MetricCollectorConfig `yaml:"metricCollector"`
}

func LoadConfig(bytes []byte) (*Config, error) {
	conf := &Config{
		Server: ServerConfig{
			Port: DefaultServerPort,
		},
		Logging: LoggingConfig{
			Level: DefaultLoggingLevel,
		},
		Aggregator: AggregatorConfig{
			AggregatorExecuateInterval: DefaultAggregatorExecuteInterval,
			PolicyPollerInterval:       DefaultPolicyPollerInterval,
			MetricPollerCount:          DefaultMetricPollerCount,
			AppMonitorChannelSize:      DefaultAppMonitorChannelSize,
		},
		Evaluator: EvaluatorConfig{
			EvaluationManagerInterval: DefaultEvaluationExecuteInterval,
			EvaluatorCount:            DefaultEvaluatorCount,
			TriggerArrayChannelSize:   DefaultTriggerArrayChannelSize,
		},
	}
	err := yaml.Unmarshal(bytes, &conf)
	if err != nil {
		return nil, err
	}
	conf.Logging.Level = strings.ToLower(conf.Logging.Level)
	if conf.Aggregator.AggregatorExecuateIntervalInSeconds != 0 {
		conf.Aggregator.AggregatorExecuateInterval = time.Duration(conf.Aggregator.AggregatorExecuateIntervalInSeconds) * time.Second
	}
	if conf.Aggregator.PolicyPollerIntervalInSeconds != 0 {
		conf.Aggregator.PolicyPollerInterval = time.Duration(conf.Aggregator.PolicyPollerIntervalInSeconds) * time.Second
	}
	if conf.Evaluator.EvaluationManagerIntervalInSeconds != 0 {
		conf.Evaluator.EvaluationManagerInterval = time.Duration(conf.Evaluator.EvaluationManagerIntervalInSeconds) * time.Second
	}
	return conf, nil
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("Configuration error: server port is less-equal than 0 or more than 65535")
	}
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
	if c.Aggregator.AggregatorExecuateInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: aggregator execute interval is less-equal than 0")
	}
	if c.Aggregator.PolicyPollerInterval <= time.Duration(0) {
		return fmt.Errorf("Configuration error: policy poller interval is less-equal than 0")
	}
	if c.Aggregator.MetricPollerCount <= 0 {
		return fmt.Errorf("Configuration error: metric poller count is less-equal than 0")
	}
	if c.Aggregator.AppMonitorChannelSize <= 0 {
		return fmt.Errorf("Configuration error: appMonitor channel size is less-equal than 0")
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
	return nil

}
