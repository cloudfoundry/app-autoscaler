package legacy_parser

import (
	"encoding/json"

	"github.com/xeipuuv/gojsonschema"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type LegacyBindingRequestParser struct {
	schema *gojsonschema.Schema
}

var _ binding_request.Parser = LegacyBindingRequestParser{}


func New() (LegacyBindingRequestParser, error) {
	const schemaFilePath string = "file://./legacy-binding-request.json"
	schemaLoader := gojsonschema.NewReferenceLoader(schemaFilePath)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return LegacyBindingRequestParser{}, err
	}
	return LegacyBindingRequestParser{schema: schema}, nil
}

func (p LegacyBindingRequestParser) Parse(bindingReqParams string) (binding_request.Parameters, error) {
	documentLoader := gojsonschema.NewStringLoader(bindingReqParams)
	validationResult, err := p.schema.Validate(documentLoader)
	if err != nil {
		// Defined by the implementation of `Validate`, this only happens, if the provided document
		// (in this context `documentLoader`) can not be loaded.
		return binding_request.Parameters{}, err
	} else if !validationResult.Valid() {
		// The error contains a description of all detected violations against the schema.
		allErrors := binding_request.JsonSchemaError(validationResult.Errors())
		return binding_request.Parameters{}, allErrors
	}

	var parsedParameters policyAndBindingCfg
	err = json.Unmarshal([]byte(bindingReqParams), &parsedParameters)
	if err != nil {
		return binding_request.Parameters{}, err
	}

	return toBindingParameters(parsedParameters), nil
}

func toBindingParameters(params policyAndBindingCfg) binding_request.Parameters {
	result := binding_request.Parameters{}
	result.Configuration = &models.BindingConfig{
		AppGUID: models.GUID(params.BindingConfig.AppGUID),
		CustomMetrics: models.CustomMetricsConfig{
			MetricSubmissionStrategy: models.MetricsSubmissionStrategy{
				AllowFrom: params.BindingConfig.CustomMetrics.MetricSubmissionStrategy.AllowFrom,
			},
		},
	}

	result.ScalingPolicy = &models.ScalingPolicy{
		InstanceMin: params.InstanceMin,
		InstanceMax: params.InstanceMax,
	}

	for _, rule := range params.ScalingRules {
		scalingRule := &models.ScalingRule{
			MetricType:            rule.MetricType,
			BreachDurationSeconds: rule.BreachDurationSeconds,
			Threshold:             rule.Threshold,
			Operator:              rule.Operator,
			CoolDownSeconds:       rule.CoolDownSeconds,
			Adjustment:            rule.Adjustment,
		}
		result.ScalingPolicy.ScalingRules = append(result.ScalingPolicy.ScalingRules, scalingRule)
	}

	if params.Schedules != nil {
		result.ScalingPolicy.Schedules = &models.ScalingSchedules{
			Timezone: params.Schedules.Timezone,
		}

		for _, recurring := range params.Schedules.RecurringSchedules {
			recurringSchedule := &models.RecurringSchedule{
				StartTime:             recurring.StartTime,
				EndTime:               recurring.EndTime,
				DaysOfWeek:            recurring.DaysOfWeek,
				DaysOfMonth:           recurring.DaysOfMonth,
				StartDate:             recurring.StartDate,
				EndDate:               recurring.EndDate,
				ScheduledInstanceMin:  recurring.ScheduledInstanceMin,
				ScheduledInstanceMax:  recurring.ScheduledInstanceMax,
				ScheduledInstanceInit: recurring.ScheduledInstanceInit,
			}
			result.ScalingPolicy.Schedules.RecurringSchedules = append(result.ScalingPolicy.Schedules.RecurringSchedules, recurringSchedule)
		}

		for _, specific := range params.Schedules.SpecificDateSchedules {
			specificDateSchedule := &models.SpecificDateSchedule{
				StartDateTime:         specific.StartDateTime,
				EndDateTime:           specific.EndDateTime,
				ScheduledInstanceMin:  specific.ScheduledInstanceMin,
				ScheduledInstanceMax:  specific.ScheduledInstanceMax,
				ScheduledInstanceInit: specific.ScheduledInstanceInit,
			}
			result.ScalingPolicy.Schedules.SpecificDateSchedules = append(result.ScalingPolicy.Schedules.SpecificDateSchedules, specificDateSchedule)
		}
	}

	return result
}




// ================================================================================
// Alternative Legacy Binding Request Parser
// ================================================================================


// import (
//	"encoding/json"
//	"fmt"

//	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request"
//	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/policyvalidator"
//	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
// )

// type LegacyBindingRequestParser struct {
//	// This part here parses the legacy-policy.
//	policyValidator policyvalidator.PolicyValidator // 🚧 To-do: Check if this is really needed!
// }

// var _ binding_request.Parser = LegacyBindingRequestParser{}

// func New(pathToSchemaFile string) LegacyBindingRequestParser {
//	p := policyvalidator.NewPolicyValidator(pathToSchemaFile, _, _, _, _, _, _, _, _)
//	return LegacyBindingRequestParser{
//		policyValidator: p,
//	}
// }

// func (p LegacyBindingRequestParser) Parse(bindingReqParams string) (binding_request.Parameters, error) {
//	rawJson := json.RawMessage(bindingReqParams)
//	policy, err := p.policyValidator.ParseAndValidatePolicy(rawJson)
//	if err != nil {
//		err_info := fmt.Errorf("invalid policy provided: %w", err)
//		return binding_request.Parameters{}, err_info
//	}

//	return binding_request.Parameters{
//		Configuration: nil, // 🚧 To-do!
//		ScalingPolicy: policy, // 🚧 To-do: Return nil, if policy is empty
//	}, models.ErrUnimplemented
// }


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
