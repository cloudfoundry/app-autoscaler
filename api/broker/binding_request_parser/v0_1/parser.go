package v0_1

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	brp "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker/binding_request_parser/types"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type BindingRequestParser struct {
	schema                             *gojsonschema.Schema
	defaultCustomMetricsCredentialType models.CustomMetricsBindingAuthScheme
}

var _ brp.Parser = BindingRequestParser{}

func new(
	jsonLoader gojsonschema.JSONLoader, defaultCustomMetricsCredentialType models.CustomMetricsBindingAuthScheme,
) (BindingRequestParser, error) {
	schema, err := gojsonschema.NewSchema(jsonLoader)
	if err != nil {
		return BindingRequestParser{}, err
	} else {
		parser := BindingRequestParser{
			schema:                             schema,
			defaultCustomMetricsCredentialType: defaultCustomMetricsCredentialType,
		}
		return parser, nil
	}
}

func NewFromString(
	jsonSchema string, defaultCustomMetricsCredentialType models.CustomMetricsBindingAuthScheme,
) (BindingRequestParser, error) {
	schemaLoader := gojsonschema.NewStringLoader(jsonSchema)
	return new(schemaLoader, defaultCustomMetricsCredentialType)
}

func NewFromFile(
	pathToSchemaFile string, defaultCustomMetricsCredentialType models.CustomMetricsBindingAuthScheme,
) (BindingRequestParser, error) {
	// The type for parameter `pathToSchemaFile` is same type as used in golang's std-library
	schemaLoader := gojsonschema.NewReferenceLoader(pathToSchemaFile)
	return new(schemaLoader, defaultCustomMetricsCredentialType)
}

func (p BindingRequestParser) Parse(
	bindingReqParams string, ccAppGuid models.GUID,
) (models.AppScalingConfig, error) {
	validationErr := p.Validate(bindingReqParams)
	if validationErr != nil {
		return models.AppScalingConfig{}, validationErr
	}

	var parsedParameters parameters
	err := json.Unmarshal([]byte(bindingReqParams), &parsedParameters)
	if err != nil {
		return models.AppScalingConfig{}, err
	}

	return p.toBindingParameters(parsedParameters, ccAppGuid)
}

func (p BindingRequestParser) Validate(bindingReqParams string) error {
	documentLoader := gojsonschema.NewStringLoader(bindingReqParams)
	validationResult, err := p.schema.Validate(documentLoader)
	if err != nil {
		return err
	} else if !validationResult.Valid() {
		allErrors := brp.JsonSchemaError(validationResult.Errors())
		return &allErrors
	}

	return nil
}

func (p BindingRequestParser) toBindingParameters(
	bindingReqParams parameters, ccAppGuid models.GUID,
) (models.AppScalingConfig, error) {
	appGuid, err := extractAppGuid(bindingReqParams, ccAppGuid)
	if err != nil {
		return models.AppScalingConfig{}, err
	}

	// 🚧 To-do: Single layer of abstraction

	customMetricsBindAuthScheme := &p.defaultCustomMetricsCredentialType
	if bindingReqParams.CredentialType != "" {
		var err error
		customMetricsBindAuthScheme, err = models.ParseCustomMetricsBindingAuthScheme(
			bindingReqParams.CredentialType)

		if err != nil {
			return models.AppScalingConfig{}, &models.InvalidArgumentError{
				Param: "credential-type",
				Value: err,
				Msg:   "Failed to parse the credential-type for custom metrics",
			}
		}
	}

	bindingConfig := *models.NewBindingConfig(appGuid, customMetricsBindAuthScheme)

	customMetricsStrat := models.DefaultCustomMetricsStrategy
	customMetricsStratIsSet := bindingReqParams.Configuration != nil && bindingReqParams.Configuration.CustomMetrics != nil
	if customMetricsStratIsSet {
		strat, err := models.ParseCustomMetricsStrategy(
			bindingReqParams.Configuration.CustomMetrics.MetricSubmissionStrategy.AllowFrom)

		if err != nil {
			return models.AppScalingConfig{}, &models.InvalidArgumentError{
				Param: "custom_metrics.metric_submission_strategy.allow_from",
				Value: err,
				Msg:   "Failed to parse custom-metric-submission-strategy",
			}
		}
		customMetricsStrat = *strat
	}

	var scalingPolicy *models.ScalingPolicy
	policyDefinition := readPolicyDefinition(bindingReqParams)
	scalingPolicy = models.NewScalingPolicy(customMetricsStrat, policyDefinition)

	return *models.NewAppScalingConfig(bindingConfig, *scalingPolicy), nil
}

func extractAppGuid(bindingReqParams parameters, ccAppGuid models.GUID) (models.GUID, error) {
	var bindCfgAppGuid models.GUID
	if bindingReqParams.Configuration != nil {
		bindCfgAppGuid = models.GUID(bindingReqParams.Configuration.AppGuid)
	}
	appGuidIsFromCC := ccAppGuid != ""
	appGuidIsFromBindingConfig := bindCfgAppGuid != ""

	var appGuid models.GUID
	switch {
	case appGuidIsFromCC && appGuidIsFromBindingConfig:
		msg := "error: app GUID provided in both, binding resource and binding configuration"
		err := fmt.Errorf("%s:\n\tfrom binding-request: %s", msg, bindCfgAppGuid)
		return models.GUID(""), err
	case appGuidIsFromCC:
		appGuid = ccAppGuid
	case appGuidIsFromBindingConfig:
		appGuid = bindCfgAppGuid
	default:
		return models.GUID(""), &brp.BindReqNoAppGuid{}
	}

	return appGuid, nil
}

func readPolicyDefinition(bindingReqParams parameters) *models.PolicyDefinition {
	noPolicyIsSet := bindingReqParams.InstanceMin == 0 && bindingReqParams.InstanceMax == 0 &&
		len(bindingReqParams.ScalingRules) == 0 && bindingReqParams.Schedules == nil
	if noPolicyIsSet {
		return nil
	}

	policyDefinition := models.PolicyDefinition{
		InstanceMin: bindingReqParams.InstanceMin,
		InstanceMax: bindingReqParams.InstanceMax,
	}

	for _, rule := range bindingReqParams.ScalingRules {
		scalingRule := &models.ScalingRule{
			MetricType:            rule.MetricType,
			BreachDurationSeconds: rule.BreachDurationSecs,
			Threshold:             rule.Threshold,
			Operator:              rule.Operator,
			CoolDownSeconds:       rule.CoolDownSecs,
			Adjustment:            rule.Adjustment,
		}
		policyDefinition.ScalingRules = append(policyDefinition.ScalingRules, scalingRule)
	}

	if bindingReqParams.Schedules != nil {
		policyDefinition.Schedules = &models.ScalingSchedules{
			Timezone: bindingReqParams.Schedules.Timezone,
		}

		for _, recurring := range bindingReqParams.Schedules.RecurringSchedule {
			recurringSchedule := &models.RecurringSchedule{
				StartTime:             recurring.StartTime,
				EndTime:               recurring.EndTime,
				DaysOfWeek:            recurring.DaysOfWeek,
				DaysOfMonth:           recurring.DaysOfMonth,
				StartDate:             recurring.StartDate,
				EndDate:               recurring.EndDate,
				ScheduledInstanceMin:  recurring.InstanceMinCount,
				ScheduledInstanceMax:  recurring.InstanceMaxCount,
				ScheduledInstanceInit: recurring.InitialMinInstanceCount,
			}
			policyDefinition.Schedules.RecurringSchedules = append(
				policyDefinition.Schedules.RecurringSchedules, recurringSchedule)
		}

		for _, specific := range bindingReqParams.Schedules.SpecificDate {
			specificDateSchedule := &models.SpecificDateSchedule{
				StartDateTime:         specific.StartDateTime,
				EndDateTime:           specific.EndDateTime,
				ScheduledInstanceMin:  specific.InstanceMinCount,
				ScheduledInstanceMax:  specific.InstanceMaxCount,
				ScheduledInstanceInit: specific.InitialMinInstanceCount,
			}
			policyDefinition.Schedules.SpecificDateSchedules = append(
				policyDefinition.Schedules.SpecificDateSchedules, specificDateSchedule)
		}
	}

	return &policyDefinition
}
