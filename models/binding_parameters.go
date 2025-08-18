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
	configuration BindingConfig

	// True if and only if there has not been set any policy for that app. If true, then the content
	// of the field `scalingPolicy` is meaningless.
	useDefaultPolicy bool

	// scalingPolicy defines the scaling behavior and rules for the binding.
	scalingPolicy ScalingPolicy // 🚧 To-do: We should distinguish between raw data and correctly validated data.
}


func NewBindingParameters(
	configuration BindingConfig, scalingPolicy *ScalingPolicy,
) (bps *BindingParameters) {
	if scalingPolicy == nil {
		bps = &BindingParameters{
			configuration:    configuration,
			useDefaultPolicy: true,
			scalingPolicy:    ScalingPolicy{}, // Default policy, which is empty.
		}
	} else {
		bps = &BindingParameters{
			configuration:    configuration,
			useDefaultPolicy: false,
			scalingPolicy:    *scalingPolicy,
		}
	}

	return bps
}

func (bp *BindingParameters) GetConfiguration() BindingConfig {
	return bp.configuration
}

// GetScalingPolicy returns the scaling policy for the binding and nil if no one has been set (which
// means, the default-policy is used).
func (bp *BindingParameters) GetScalingPolicy() (p *ScalingPolicy) {
	if bp.useDefaultPolicy {
		p = nil // No scaling policy has been set, so we return nil.
	} else {
		p = &bp.scalingPolicy
	}

	return p
}



// ================================================================================
// Deserialisation and serialisation methods for BindingParameters
// ================================================================================

type bindingParamsJsonRawRepr struct {
	Configuration    json.RawMessage `json:"configuration,omitempty"`
	*ScalingPolicy
}

func (bp BindingParameters) ToRawJSON() (json.RawMessage, error) {
	cfgRaw, err := bp.configuration.ToRawJSON()
	if err != nil {
		return nil, fmt.Errorf(
			"could not serialise configuration to json: %s\n\t%w",
			bp.configuration, err)
	}

	var policy *ScalingPolicy
	if bp.useDefaultPolicy {
		policy = nil // ScalingPolicy{} // Gets not serialized, which is equivalent to null in JSON.
	} else {
		policy = &bp.scalingPolicy
	}

	bpRaw := bindingParamsJsonRawRepr{
		Configuration:    cfgRaw,
		ScalingPolicy:    policy,
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

	var configuration *BindingConfig
	if len(bpRaw.Configuration) > 0 {
		var err error
		configuration, err = BindingConfigFromRawJSON(bpRaw.Configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
		}
	} else {
		configuration = DefaultBindingConfig()
	}

	scalingPolicy := bpRaw.ScalingPolicy

	return NewBindingParameters(*configuration, scalingPolicy), nil
}
