package models

import (
	"errors"
	"fmt"
)

// BindingConfig
/* The configuration object received as part of the binding parameters. Example config:
{
  "configuration": {
    "custom_metrics": {
      "metric_submission_strategy": {
        "allow_from": "bound_app"
      }
    }
  }
*/

const (
	CustomMetricsBoundApp = "bound_app"
	// CustomMetricsSameApp default value if not specified
	CustomMetricsSameApp = "same_app"
)

type BindingConfig struct {
	Configuration Configuration `json:"configuration"`
}
type Configuration struct {
	CustomMetrics CustomMetricsConfig `json:"custom_metrics"`
}

type CustomMetricsConfig struct {
	MetricSubmissionStrategy MetricsSubmissionStrategy `json:"metric_submission_strategy"`
}

type MetricsSubmissionStrategy struct {
	AllowFrom string `json:"allow_from"`
}

func (b *BindingConfig) GetCustomMetricsStrategy() string {
	return b.Configuration.CustomMetrics.MetricSubmissionStrategy.AllowFrom
}

func (b *BindingConfig) SetCustomMetricsStrategy(allowFrom string) {
	b.Configuration.CustomMetrics.MetricSubmissionStrategy.AllowFrom = allowFrom
}

/**
 * GetBindingConfigAndPolicy combines the binding configuration and policy based on the given parameters.
 * It establishes the relationship between the scaling policy and the custom metrics strategy.
 * @param scalingPolicy the scaling policy
 * @param customMetricStrategy the custom metric strategy
 * @return the binding configuration and policy if both are present, the scaling policy if only the policy is present,
* 			the binding configuration if only the configuration is present
 * @throws an error if no policy or custom metrics strategy is found
*/

func GetBindingConfigAndPolicy(scalingPolicy *ScalingPolicy, customMetricStrategy string) (*ScalingPolicyWithBindingConfig, error) {
	if scalingPolicy == nil {
		return nil, fmt.Errorf("policy not found")
	}
	if customMetricStrategy != "" && customMetricStrategy != CustomMetricsSameApp { //if customMetricStrategy found
		return buildPolicyAndConfig(scalingPolicy, customMetricStrategy), nil
	}
	return &ScalingPolicyWithBindingConfig{
		ScalingPolicy: *scalingPolicy,
	}, nil
}

func buildPolicyAndConfig(scalingPolicy *ScalingPolicy, customMetricStrategy string) *ScalingPolicyWithBindingConfig {
	bindingConfig := &BindingConfig{}
	bindingConfig.SetCustomMetricsStrategy(customMetricStrategy)

	return &ScalingPolicyWithBindingConfig{
		BindingConfig: bindingConfig,
		ScalingPolicy: *scalingPolicy,
	}
}

func (b *BindingConfig) ValidateOrGetDefaultCustomMetricsStrategy() (*BindingConfig, error) {
	strategy := b.GetCustomMetricsStrategy()
	if strategy == "" {
		b.SetCustomMetricsStrategy(CustomMetricsSameApp)
	} else if strategy != CustomMetricsBoundApp {
		return nil, errors.New("error: custom metrics strategy not supported")
	}
	return b, nil
}
