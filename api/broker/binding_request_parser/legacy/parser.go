package legacy

import (
	"encoding/json"

	"github.com/xeipuuv/gojsonschema"

	brp "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker/binding_request_parser/types"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type BindingRequestParser struct {
	schema                             *gojsonschema.Schema
	defaultCustomMetricsCredentialType models.CustomMetricsBindingAuthScheme
}

var _ brp.Parser = BindingRequestParser{}

// New creates a new LegacyBindingRequestParser with the JSON schema loaded from the specified file
// path.
//
// The schemaFilePath parameter should be a valid and absolute file URI (e.g.,
// "file:///path/to/schema.json").
//
// Returns an error if the schema file cannot be loaded or parsed.
func New(
	schemaFilePath string, defaultCustomMetricsCredentialType models.CustomMetricsBindingAuthScheme,
) (BindingRequestParser, error) {
	schemaLoader := gojsonschema.NewReferenceLoader(schemaFilePath)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return BindingRequestParser{}, err
	}

	parser := BindingRequestParser{
		schema:                             schema,
		defaultCustomMetricsCredentialType: defaultCustomMetricsCredentialType,
	}

	return parser, nil
}

func (p BindingRequestParser) Parse(
	bindingReqParams string, ccAppGuid models.GUID,
) (models.AppScalingConfig, error) {
	validationErr := p.Validate(bindingReqParams)
	if validationErr != nil {
		return models.AppScalingConfig{}, validationErr
	}

	var parsedParameters policyAndBindingCfg
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
		// Defined by the implementation of `Validate`, this only happens, if the provided document
		// (in this context `documentLoader`) can not be loaded.
		return err
	} else if !validationResult.Valid() {
		// The error contains a description of all detected violations against the schema.
		allErrors := brp.JsonSchemaError(validationResult.Errors())
		return &allErrors
	}

	return nil
}

func (p BindingRequestParser) toBindingParameters(
	bindingReqParams policyAndBindingCfg, ccAppGuid models.GUID,
) (models.AppScalingConfig, error) {
	appGuid := ccAppGuid
	if ccAppGuid == "" {
		return models.AppScalingConfig{}, &models.InvalidArgumentError{
			Param: "ccAppGuid",
			Value: ccAppGuid,
			Msg: `⛔ Did not get any app-guid from Cloud Controller.
This must not happen for the legacy-parser because …
 + the legacy-parser does not support service-keys and because of …
 + prior schema-validation.
This is a programming-error.`,
		}
	}

	customMetricsBindAuthScheme := &p.defaultCustomMetricsCredentialType
	if schemeIsSet := bindingReqParams.CredentialType != ""; schemeIsSet {
		var err error
		customMetricsBindAuthScheme, err = models.ParseCustomMetricsBindingAuthScheme(
			bindingReqParams.CredentialType)

		if err != nil {
			return models.AppScalingConfig{}, &models.InvalidArgumentError{
				Param: "err",
				Value: err,
				Msg: `⛔ Failed to parse the credential-type for custom metrics.
This must not happen because of prior schema-validation. This is a programming-error.`,
			}
		}
	}

	customMetricsStrat := models.DefaultCustomMetricsStrategy
	customMetricsStratIsSet := bindingReqParams.BindingConfig != nil && bindingReqParams.BindingConfig.CustomMetrics != nil
	if customMetricsStratIsSet {
		strategy, err := models.ParseCustomMetricsStrategy(
			bindingReqParams.BindingConfig.CustomMetrics.MetricSubmissionStrategy.AllowFrom)

		if err != nil {
			return models.AppScalingConfig{}, &models.InvalidArgumentError{
				Param: "err",
				Value: err,
				Msg: `
⛔ Failed to parse custom-metric-submission-strategy; This must not happen because of prior schema validation.
This is an programming-error.`,
			}
		}
		customMetricsStrat = *strategy
	}

	bindingConfig := *models.NewBindingConfig(appGuid, customMetricsBindAuthScheme)
	policyDefinition := readPolicyDefinition(bindingReqParams)
	scalingPolicy := models.NewScalingPolicy(customMetricsStrat, policyDefinition)

	return *models.NewAppScalingConfig(bindingConfig, *scalingPolicy), nil
}

func readPolicyDefinition(bindingReqParams policyAndBindingCfg) *models.PolicyDefinition {
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
			BreachDurationSeconds: rule.BreachDurationSeconds,
			Threshold:             rule.Threshold,
			Operator:              rule.Operator,
			CoolDownSeconds:       rule.CoolDownSeconds,
			Adjustment:            rule.Adjustment,
		}
		policyDefinition.ScalingRules = append(policyDefinition.ScalingRules, scalingRule)
	}

	if bindingReqParams.Schedules != nil {
		policyDefinition.Schedules = &models.ScalingSchedules{
			Timezone: bindingReqParams.Schedules.Timezone,
		}

		for _, recurring := range bindingReqParams.Schedules.RecurringSchedules {
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
			policyDefinition.Schedules.RecurringSchedules = append(
				policyDefinition.Schedules.RecurringSchedules, recurringSchedule)
		}

		for _, specific := range bindingReqParams.Schedules.SpecificDateSchedules {
			specificDateSchedule := &models.SpecificDateSchedule{
				StartDateTime:         specific.StartDateTime,
				EndDateTime:           specific.EndDateTime,
				ScheduledInstanceMin:  specific.ScheduledInstanceMin,
				ScheduledInstanceMax:  specific.ScheduledInstanceMax,
				ScheduledInstanceInit: specific.ScheduledInstanceInit,
			}
			policyDefinition.Schedules.SpecificDateSchedules = append(
				policyDefinition.Schedules.SpecificDateSchedules, specificDateSchedule)
		}
	}

	return &policyDefinition
}
