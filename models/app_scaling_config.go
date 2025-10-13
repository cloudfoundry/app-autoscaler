package models

import (
	"encoding/json"
	"fmt"
)

// AppScalingConfig contains all the necessary data to establish the relationship between
// application binding configuration and its associated scaling behavior.
//
// ⛔ Do not create `AppScalingConfig` values directly via `AppScalingConfig{}` because it can lead to
// undefined behaviour due to bypassing all validations. Use the constructor-functions instead!
type AppScalingConfig struct {
	// configuration contains the binding configuration settings.
	configuration BindingConfig

	// scalingPolicy defines the scaling behavior and rules for the binding.
	scalingPolicy ScalingPolicy
}

func NewAppScalingConfig(
	configuration BindingConfig, scalingPolicy ScalingPolicy,
) (bps *AppScalingConfig) {
	return &AppScalingConfig{
		configuration: configuration,
		scalingPolicy: scalingPolicy,
	}
}

func (asc *AppScalingConfig) GetConfiguration() *BindingConfig {
	return &asc.configuration
}

// GetScalingPolicy returns the scaling policy for the binding and nil if no one has been set (which
// means, the default-policy is used).
func (asc *AppScalingConfig) GetScalingPolicy() (p *ScalingPolicy) {
	return &asc.scalingPolicy
}

// ================================================================================
// Deserialisation and serialisation methods for AppScalingConfig
// ================================================================================

type appScalingCfgRawRepr struct {
	CfgRaw    json.RawMessage `json:"binding-configuration,omitempty"`
	PolicyRaw json.RawMessage `json:"scaling-policy,omitempty"`
}

func (asc AppScalingConfig) ToRawJSON() (json.RawMessage, error) {
	cfgRaw, err := asc.configuration.ToRawJSON()
	if err != nil {
		return nil, fmt.Errorf(
			"could not serialise configuration to json: %s\n\t%w",
			asc.configuration, err)
	}

	policyRaw, err := asc.scalingPolicy.ToRawJSON()
	if err != nil {
		return nil, fmt.Errorf(
			"could not serialise scaling policy to json: \n\t%w", err)
	}

	ascRaw := appScalingCfgRawRepr{
		CfgRaw:    cfgRaw,
		PolicyRaw: policyRaw,
	}

	data, err := json.Marshal(ascRaw)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func AppScalingConfigFromRawJSON(data json.RawMessage) (*AppScalingConfig, error) {
	var ascRaw appScalingCfgRawRepr
	if err := json.Unmarshal(data, &ascRaw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AppScalingConfig: %w", err)
	}

	cfg, err := BindingConfigFromRawJSON(ascRaw.CfgRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal binding configuration: %w", err)
	}

	policy, err := ScalingPolicyFromRawJSON(ascRaw.PolicyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal scaling policy: %w", err)
	}

	return NewAppScalingConfig(*cfg, *policy), nil
}
