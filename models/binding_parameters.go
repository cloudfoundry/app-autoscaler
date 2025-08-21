package models

import (
	"encoding/json"
	"fmt"
)

// 🚧 To-do: BindingParameters ↦ AppScalingConfig

// BindingParameters contains all the necessary data to establish the relationship between
// application binding configuration and its associated scaling behavior.
//
// ⛔ Do not create `BindingParameters` values directly via `BindingParameters{}` because it can lead to
// undefined behaviour due to bypassing all validations. Use the constructor-functions instead!
type BindingParameters struct {
	// configuration contains the binding configuration settings.
	configuration BindingConfig

	// scalingPolicy defines the scaling behavior and rules for the binding.
	scalingPolicy ScalingPolicy // 🚧 To-do: We should distinguish between raw data and correctly validated data.
}

func NewBindingParameters(
	configuration BindingConfig, scalingPolicy ScalingPolicy,
) (bps *BindingParameters) {
	return &BindingParameters{
		configuration: configuration,
		scalingPolicy: scalingPolicy,
	}
}

func (bp *BindingParameters) GetConfiguration() BindingConfig {
	return bp.configuration
}

// GetScalingPolicy returns the scaling policy for the binding and nil if no one has been set (which
// means, the default-policy is used).
func (bp *BindingParameters) GetScalingPolicy() (p *ScalingPolicy) {
	return &bp.scalingPolicy
}

// ================================================================================
// Deserialisation and serialisation methods for BindingParameters
// ================================================================================

type bindingParamsJsonRawRepr struct {
	CfgRaw    json.RawMessage `json:"binding_cfg,omitempty"`
	PolicyRaw json.RawMessage `json:"policy,omitempty"`
}

func (bp BindingParameters) ToRawJSON() (json.RawMessage, error) {
	cfgRaw, err := bp.configuration.ToRawJSON()
	if err != nil {
		return nil, fmt.Errorf(
			"could not serialise configuration to json: %s\n\t%w",
			bp.configuration, err)
	}

	policyRaw, err := bp.scalingPolicy.ToRawJSON()
	if err != nil {
		return nil, fmt.Errorf(
			"could not serialise scaling policy to json: \n\t%w", err)
	}

	bpRaw := bindingParamsJsonRawRepr{
		CfgRaw:    cfgRaw,
		PolicyRaw: policyRaw,
	}

	data, err := json.Marshal(bpRaw)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func BindingParametersFromRawJSON(data json.RawMessage) (*BindingParameters, error) {
	var bpRaw bindingParamsJsonRawRepr
	if err := json.Unmarshal(data, &bpRaw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BindingParameters: %w", err)
	}

	cfg, err := BindingConfigFromRawJSON(bpRaw.CfgRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal binding configuration: %w", err)
	}

	policy, err := ScalingPolicyFromRawJSON(bpRaw.PolicyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal scaling policy: %w", err)
	}

	return NewBindingParameters(*cfg, *policy), nil
}
