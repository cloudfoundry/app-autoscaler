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
      "auth": {
        "credential_type": "binding_secret"
      },
      "metric_submission_strategy": {
        "allow_from": "bound_app or same_app"
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
 * DetermineBindingConfigAndPolicy determines the binding configuration and policy based on the given parameters.
 * It establishes the relationship between the scaling policy and the custom metrics strategy.
 * @param scalingPolicy the scaling policy
 * @param customMetricStrategy the custom metric strategy
 * @return the binding configuration and policy if both are present, the scaling policy if only the policy is present,
* 			the binding configuration if only the configuration is present
 * @throws an error if no policy or custom metrics strategy is found
*/

func DetermineBindingConfigAndPolicy(scalingPolicy *ScalingPolicy, customMetricStrategy string) (interface{}, error) {
	if scalingPolicy == nil {
		return nil, fmt.Errorf("policy not found")
	}

	combinedConfig, bindingConfig := buildConfigurationIfPresent(customMetricStrategy)
	if combinedConfig != nil { //both are present
		combinedConfig.ScalingPolicy = *scalingPolicy
		combinedConfig.BindingConfig = *bindingConfig
		return combinedConfig, nil
	}
	return scalingPolicy, nil
}

func buildConfigurationIfPresent(customMetricsStrategy string) (*BindingConfigWithPolicy, *BindingConfig) {
	var combinedConfig *BindingConfigWithPolicy
	var bindingConfig *BindingConfig

	if customMetricsStrategy != "" && customMetricsStrategy != CustomMetricsSameApp { //if custom metric was given in the binding process
		combinedConfig = &BindingConfigWithPolicy{}
		bindingConfig = &BindingConfig{}
		bindingConfig.SetCustomMetricsStrategy(customMetricsStrategy)
		combinedConfig.BindingConfig = *bindingConfig
	}
	return combinedConfig, bindingConfig
}

func (b *BindingConfig) ValidateOrGetDefaultCustomMetricsStrategy(bindingConfiguration *BindingConfig) (*BindingConfig, error) {
	strategy := bindingConfiguration.GetCustomMetricsStrategy()
	if strategy == "" {
		bindingConfiguration.SetCustomMetricsStrategy(CustomMetricsSameApp)
	} else if strategy != CustomMetricsBoundApp {
		return nil, errors.New("error: custom metrics strategy not supported")
	}
	return bindingConfiguration, nil
}
