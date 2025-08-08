package models

import (
	"errors"
	"fmt"
)

// BindingConfig
/* The configuration object received as part of the binding parameters. Example config:
{
  "configuration": {
	"app-guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91",
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
	AppGUID       GUID                `json:"app_guid,omitempty"` // Empty value represents null-value (i.e. not set).
	CustomMetrics *CustomMetricsConfig `json:"custom_metrics,omitempty"`
}

type CustomMetricsConfig struct {
	MetricSubmissionStrategy MetricsSubmissionStrategy `json:"metric_submission_strategy"`
}

type MetricsSubmissionStrategy struct {
	AllowFrom string `json:"allow_from"`
}

func (b *BindingConfig) GetCustomMetricsStrategy() string {
	return b.CustomMetrics.MetricSubmissionStrategy.AllowFrom
}

func (b *BindingConfig) SetCustomMetricsStrategy(allowFrom string) {
	if b.CustomMetrics == nil {
		b.CustomMetrics = &CustomMetricsConfig{}
	}
	b.CustomMetrics.MetricSubmissionStrategy.AllowFrom = allowFrom
}

/**
 * GetBindingConfigAndPolicy combines the binding configuration and policy based on the given parameters.
 * It establishes the relationship between the scaling policy and the custom metrics strategy.
 * @param scalingPolicy the scaling policy
 * @param customMetricStrategy the custom metric strategy
 * @return the binding configuration and policy if both are present, the scaling policy if only the policy is present,
*			the binding configuration if only the configuration is present
 * @throws an error if no policy or custom metrics strategy is found
*/

func GetBindingConfigAndPolicy(scalingPolicy *ScalingPolicy, serviceBinding *ServiceBinding) (*ScalingPolicyWithBindingConfig, error) {
	if scalingPolicy == nil {
		return nil, fmt.Errorf("policy not found")
	}
	if serviceBinding == nil {
		return nil, fmt.Errorf("service-binding not found")
	}
	isBindingConfigRelevant := serviceBinding.CustomMetricsStrategy != "" &&
		serviceBinding.CustomMetricsStrategy != CustomMetricsSameApp ||
		serviceBinding.AppID != ""

	if isBindingConfigRelevant {
		return buildPolicyAndConfig(scalingPolicy, serviceBinding), nil
	}
	return &ScalingPolicyWithBindingConfig{
		ScalingPolicy: *scalingPolicy,
	}, nil
}

func buildPolicyAndConfig(scalingPolicy *ScalingPolicy, serviceBinding *ServiceBinding) *ScalingPolicyWithBindingConfig {
	bindingConfig := &BindingConfig{
		// As well correct, if AppID is `""`, see definition of type `BindingConfig`
		AppGUID: GUID(serviceBinding.AppID),
	}
	isCustomMetricsRelevant := serviceBinding.CustomMetricsStrategy != ""
	if isCustomMetricsRelevant {
		bindingConfig.SetCustomMetricsStrategy(serviceBinding.CustomMetricsStrategy)
	}

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
