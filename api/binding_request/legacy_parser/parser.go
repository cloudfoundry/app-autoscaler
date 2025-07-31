package legacy_parser

import (
	"encoding/json"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/policyvalidator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type LegacyBindingRequestParser struct {
	// This part here parses the legacy-policy.
	policyValidator policyvalidator.PolicyValidator // 🚧 To-do: Check if this is really needed!
}

var _ binding_request.Parser = LegacyBindingRequestParser{}

func New(pathToSchemaFile string) LegacyBindingRequestParser {
	p := policyvalidator.NewPolicyValidator(pathToSchemaFile, _, _, _, _, _, _, _, _)
	return LegacyBindingRequestParser{
		policyValidator: p,
	}
}

func (p LegacyBindingRequestParser) Parse(bindingReqParams string) (binding_request.Parameters, error) {
	rawJson := json.RawMessage(bindingReqParams)
	policy, err := p.policyValidator.ParseAndValidatePolicy(rawJson)
	if err != nil {
		err_info := fmt.Errorf("invalid policy provided: %w", err)
		return binding_request.Parameters{}, err_info
	}

	return binding_request.Parameters{
		Configuration: nil, // 🚧 To-do!
		ScalingPolicy: policy, // 🚧 To-do: Return nil, if policy is empty
	}, models.ErrUnimplemented
}


// func (p LegacyBindingRequestParser) getPolicyFromJsonRawMessage(policyJson json.RawMessage, instanceID string, planID string) (*models.ScalingPolicy, error) {
//	if policyJson != nil || len(policyJson) != 0 {
//		return p.validateAndCheckPolicy(policyJson, instanceID, planID)
//	}
//	return nil, nil
// }

// func (p LegacyBindingRequestParser) validateAndCheckPolicy(rawJson json.RawMessage, instanceID string, planID string) (*models.ScalingPolicy, error) {
//	policy, errResults := p.policyValidator.ParseAndValidatePolicy(rawJson)

//	if errResults != nil {
//		return policy, apiresponses.NewFailureResponse(fmt.Errorf("invalid policy provided: %s", string(resultsJson)), http.StatusBadRequest, "failed-to-validate-policy")
//	}
//	if err := b.planDefinitionExceeded(policy, planID, instanceID); err != nil {
//		return policy, err
//	}
//	return policy, nil
// }
