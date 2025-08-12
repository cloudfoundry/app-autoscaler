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
	AppGUID       GUID                 `json:"app_guid,omitempty"` // Empty value represents null-value (i.e. not set).
	CustomMetrics *CustomMetricsConfig `json:"custom_metrics,omitempty"`
}

type CustomMetricsConfig struct {
	MetricSubmissionStrategy MetricsSubmissionStrategy `json:"metric_submission_strategy"`
}

type MetricsSubmissionStrategy struct {
	AllowFrom string `json:"allow_from"`
}

func (b *BindingConfig) GetCustomMetricsStrategy() string {
	var result string
	if b.CustomMetrics == nil {
		result = ""
	} else {
		result = b.CustomMetrics.MetricSubmissionStrategy.AllowFrom
	}

	return result
}

// SetCustomMetricsStrategy sets the custom metrics strategy for this binding configuration.
// Validates that the provided strategy is one of the supported values.
//
// Parameters:
//   - allowFrom: The custom metrics strategy to set. Must be either CustomMetricsSameApp or CustomMetricsBoundApp.
//
// Returns:
//   - error: InvalidArgumentError if the provided strategy is not supported, nil otherwise.
func (b *BindingConfig) SetCustomMetricsStrategy(allowFrom string) error {
	if b.CustomMetrics == nil {
		b.CustomMetrics = &CustomMetricsConfig{}
	}

	// Validate strategy
	if allowFrom != CustomMetricsSameApp && allowFrom != CustomMetricsBoundApp {
		return &InvalidArgumentError{
			Param: "allowFrom",
			Value: allowFrom,
			Msg:   "custom metrics strategy must be either 'same_app' or 'bound_app'",
		}
	}

	b.CustomMetrics.MetricSubmissionStrategy.AllowFrom = allowFrom
	return nil
}


// CreateBindingConfigWithValidation creates a BindingConfig from an AppGUID and CustomMetricsStrategy.
func BindingConfigFromParameters(appGUID GUID, customMetricsStrategy string) (*BindingConfig, error) {
	config := &BindingConfig{
		AppGUID: appGUID,
	}
	err := config.SetCustomMetricsStrategy(customMetricsStrategy)
	if err != nil {
		e := fmt.Errorf(
			"error: provided strategy is unsupported:\n\t%s, %w",
			customMetricsStrategy, err)
		return nil, e
	}

	return config, nil
}



/**
 * BindingConfigFromServiceBinding creates a binding configuration from a service binding.
 * Only creates a configuration if the service binding contains relevant custom metrics strategy
 * (other than "same_app") or has an AppID set.
 *
 * @param serviceBinding the service binding to extract configuration from; must not be nil
 * @return *BindingConfig the extracted binding configuration, or nil if no relevant config found
 * @return error InvalidArgumentError if serviceBinding is nil, nil otherwise
 */
func BindingConfigFromServiceBinding(serviceBinding *ServiceBinding) (*BindingConfig, error) {
	var bindingConfig *BindingConfig

	if serviceBinding == nil {
		err := InvalidArgumentError{
			Param: "serviceBinding",
			Value: serviceBinding,
			Msg:   "serviceBinding must not be nil, see function-contract;",
		}
		return nil, &err
	}

	bindingConfig = &BindingConfig{
		AppGUID: GUID(serviceBinding.AppID),
	}
	err := bindingConfig.SetCustomMetricsStrategy(serviceBinding.CustomMetricsStrategy)

	if err != nil {
		e := fmt.Errorf(
			"error: serviceBinding contained unsupported strategy:\n\t%s, %w",
			serviceBinding, err)
		return nil, e
	}

	return bindingConfig, nil
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
