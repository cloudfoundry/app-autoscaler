package models

import (
	"encoding/json"
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

// BindingConfig represents the configuration for a service binding.
type BindingConfig struct {
	appGUID       GUID                 // Empty value represents null-value (i.e. not set).
	customMetrics customMetricsConfig
}


func NewBindingConfig(appGUID GUID, customMetricStrategy CustomMetricsStrategy) *BindingConfig {
	return &BindingConfig{
		appGUID:       appGUID,
		customMetrics: customMetricsConfig{
			MetricSubmissionStrategy: metricsSubmissionStrategy{
				AllowFrom: customMetricStrategy,
			},
		},
	}
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
		appGUID: GUID(serviceBinding.AppID),
	}

	var customMetricStrategy CustomMetricsStrategy // Validierung nötig!
	switch serviceBinding.CustomMetricsStrategy {
	case "bound_app": customMetricStrategy = CustomMetricsBoundApp
	case "same_app", "": customMetricStrategy = CustomMetricsSameApp
	default: {
		err := fmt.Errorf(
			"error: serviceBinding contained unsupported strategy:\n\t%s",
			serviceBinding)
		return nil, err
	}}

	bindingConfig.customMetrics = customMetricsConfig{
		MetricSubmissionStrategy: metricsSubmissionStrategy{
			AllowFrom: customMetricStrategy,
		},
	}
	return bindingConfig, nil
}

// GetAppGUID returns the GUID of the application associated with this binding.
func (bc *BindingConfig) GetAppGUID() GUID {
	return bc.appGUID
}

// GetCustomMetricStrategy returns the custom metrics configuration for this binding.
func (bc *BindingConfig) GetCustomMetricStrategy() CustomMetricsStrategy {
	return bc.customMetrics.MetricSubmissionStrategy.AllowFrom
}



// ================================================================================
// Types expressing a binding-configuration
// ================================================================================

// CustomMetricsStrategy defines the strategy for submitting custom metrics. It can be either
// "bound_app" or "same_app".
type CustomMetricsStrategy struct {
	value string // Not exported to prohibit construction of CustomMetricsStrategy values outside
				 // this package.
}

var (
	CustomMetricsBoundApp = CustomMetricsStrategy{"bound_app"}

	// CustomMetricsSameApp default value if not specified
	CustomMetricsSameApp = CustomMetricsStrategy{"same_app"}
	DefaultCustomMetricsStrategy = CustomMetricsSameApp
)

func (s CustomMetricsStrategy) String() string {
	return s.value
}

func (s CustomMetricsStrategy) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.value)
}

func (s *CustomMetricsStrategy) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	switch value {
	case "bound_app":
		*s = CustomMetricsBoundApp
	case "same_app":
		*s = CustomMetricsSameApp
	default:
		return fmt.Errorf("unsupported CustomMetricsStrategy: %s", value)
	}

	return nil
}



// ================================================================================
// Deserialization and serialization methods for BindingConfig
// ================================================================================

type bindingConfigJsonRawRepr struct {
	AppGUID       GUID                 `json:"app_guid,omitempty"`       // Empty value represents null-value (i.e. not set).
	CustomMetrics *customMetricsConfig `json:"custom_metrics,omitempty"` // nil value represents null-value (i.e. not set).
}

type customMetricsConfig struct {
	MetricSubmissionStrategy metricsSubmissionStrategy `json:"metric_submission_strategy"`
}

type metricsSubmissionStrategy struct {
	AllowFrom CustomMetricsStrategy `json:"allow_from"`
}

func (bc BindingConfig) ToRawJSON() (json.RawMessage, error) {
	var customMetrics *customMetricsConfig
	if bc.GetCustomMetricStrategy() == DefaultCustomMetricsStrategy{
		customMetrics = nil // Default strategy does not need to be serialized
	} else {
		customMetrics = &bc.customMetrics
	}
	bindingConfigRaw := bindingConfigJsonRawRepr{
		AppGUID:       bc.appGUID,
		CustomMetrics: customMetrics,
	}

	data, err := json.Marshal(bindingConfigRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal BindingConfig: %w", err)
	}
	return json.RawMessage(data), nil
}

func BindingConfigFromRawJSON(data json.RawMessage) (*BindingConfig, error) {
	var bindingConfigRaw bindingConfigJsonRawRepr
	if err := json.Unmarshal(data, &bindingConfigRaw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BindingConfig: %w", err)
	}
	bindingConfig := &BindingConfig{
		appGUID: bindingConfigRaw.AppGUID,
	}
	if bindingConfigRaw.CustomMetrics != nil {
		bindingConfig.customMetrics = *bindingConfigRaw.CustomMetrics
	} else {
		bindingConfig.customMetrics = customMetricsConfig{
			MetricSubmissionStrategy: metricsSubmissionStrategy{
				AllowFrom: DefaultCustomMetricsStrategy,
			},
		}
	}
	return bindingConfig, nil
}
