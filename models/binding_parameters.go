package models

import (
	"encoding/json"
	"fmt"
)

// BindingParameters contains all data the necessary data to establish the relationship between
// application binding configuration and its associated scaling behavior.
//
// This type is immutable and should be created using the constructor function
// `NewBindingParameters`.
type BindingParameters struct {
	// configuration contains the binding configuration settings.
	configuration BindingConfig // 🚧 To-do: We should distinguish between raw data and correctly validated data.

	// True if and only if there has not been set any policy for that app. If true, then the content
	// of the field `scalingPolicy` is meaningless.
	useDefaultPolicy bool

	// scalingPolicy defines the scaling behavior and rules for the binding.
	scalingPolicy ScalingPolicy // 🚧 To-do: We should distinguish between raw data and correctly validated data.
}


func NewBindingParameters(
	configuration BindingConfig, useDefaultPolicy bool, scalingPolicy ScalingPolicy,
) *BindingParameters {
	return &BindingParameters{
		configuration:    configuration,
		useDefaultPolicy: useDefaultPolicy,
		scalingPolicy:    scalingPolicy,
	}
}

func (bp *BindingParameters) GetConfiguration() BindingConfig {
	return bp.configuration
}

func (bp *BindingParameters) GetUseDefaultPolicy() bool {
	return bp.useDefaultPolicy
}

func (bp *BindingParameters) GetScalingPolicy() ScalingPolicy {
	return bp.scalingPolicy
}



// ================================================================================
// Deserialisation and serialisation methods for BindingParameters
// ================================================================================

type bindingParamsJsonRawRepr struct {
	Configuration    json.RawMessage `json:"configuration"`
	UseDefaultPolicy json.RawMessage `json:"use_default_policy"`
	ScalingPolicy    json.RawMessage `json:"scaling_policy"`
}

func (bp BindingParameters) ToRawJSON() (json.RawMessage, error) {
	var useDefaultPolicy json.RawMessage
	if bp.useDefaultPolicy {
		useDefaultPolicy = json.RawMessage("true")
	} else {
		useDefaultPolicy = json.RawMessage("false")
	}

	cfgRaw, err := bp.configuration.ToRawJSON()
	if err != nil {
		return nil, fmt.Errorf(
			"could not serialise configuration to json: %s\n\t%w",
			bp.configuration, err)
	}

	policyRaw, err := bp.scalingPolicy.ToRawJSON()
	if err != nil {
		return nil, fmt.Errorf(
			"could not serialise scaling policy to json: %s\n\t%w",
			bp.scalingPolicy, err)
	}

	bpRaw := bindingParamsJsonRawRepr{
		Configuration:    cfgRaw,
		UseDefaultPolicy: useDefaultPolicy,
		ScalingPolicy:    policyRaw,
	}

	data, err := json.Marshal(bpRaw)
	if err != nil {
		return nil, err
	}
	return data, nil
}
