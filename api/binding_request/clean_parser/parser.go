package clean_parser

import (
	"encoding/json"

	"github.com/xeipuuv/gojsonschema"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type CleanBindingRequestParser struct {
	schema *gojsonschema.Schema
}

var _ binding_request.Parser = CleanBindingRequestParser{}

func new(jsonLoader gojsonschema.JSONLoader) (CleanBindingRequestParser, error) {
	schema, err := gojsonschema.NewSchema(jsonLoader)
	if err != nil {
		return CleanBindingRequestParser{}, err
	} else {
		return CleanBindingRequestParser{schema: schema}, nil
	}
}

func NewFromString(jsonSchema string) (CleanBindingRequestParser, error) {
	schemaLoader := gojsonschema.NewStringLoader(jsonSchema)
	return new(schemaLoader)
}

func NewFromFile(pathToSchemaFile string) (CleanBindingRequestParser, error) {
	// The type for parameter `pathToSchemaFile` is same type as used in golang's std-library
	schemaLoader := gojsonschema.NewReferenceLoader(pathToSchemaFile)
	return new(schemaLoader)
}

func (p CleanBindingRequestParser) Parse(bindingReqParams string) (binding_request.Parameters, error) {
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

	var parsedParameters parameters
	err = json.Unmarshal([]byte(bindingReqParams), &parsedParameters)
	if err != nil {
		return binding_request.Parameters{}, err
	}

	return toBindingParameters(parsedParameters), nil
}

func toBindingParameters(params parameters) binding_request.Parameters {
	result := binding_request.Parameters{}
	if params.Configuration != nil {
		result.Configuration = &models.BindingConfig{}

		result.Configuration.AppGUID = models.GUID(params.Configuration.AppGuid)
		result.Configuration.SetCustomMetricsStrategy(params.Configuration.CustomMetricsCfg.MetricSubmStrat.AllowFrom)
	}

	if params.ScalingPolicy != nil {
		result.ScalingPolicy = &models.ScalingPolicy{}

		result.ScalingPolicy.InstanceMax = params.ScalingPolicy.InstanceMaxCount
		result.ScalingPolicy.InstanceMin = params.ScalingPolicy.InstanceMinCount

		result.ScalingPolicy.ScalingRules = []*models.ScalingRule{}
		for _, rule := range params.ScalingPolicy.ScalingRules {
			r := models.ScalingRule{
				MetricType:            rule.MetricType,
				BreachDurationSeconds: rule.BreachDurationSecs,
				Threshold:             rule.Threshold,
				Operator:              rule.Operator,
				CoolDownSeconds:       rule.CoolDownSecs,
				Adjustment:            rule.Adjustment,
			}
			result.ScalingPolicy.ScalingRules = append(result.ScalingPolicy.ScalingRules, &r)
		}

		if params.ScalingPolicy.Schedules != nil {
			result.ScalingPolicy.Schedules = &models.ScalingSchedules{
				Timezone: params.ScalingPolicy.Schedules.Timezone,
			}

			if params.ScalingPolicy.Schedules.RecurringSchedule != nil {
				result.ScalingPolicy.Schedules.RecurringSchedules = []*models.RecurringSchedule{}
				for _, schedule := range params.ScalingPolicy.Schedules.RecurringSchedule {
					rs := models.RecurringSchedule{
						StartTime:             schedule.StartTime,
						EndTime:               schedule.EndTime,
						DaysOfWeek:            schedule.DaysOfWeek,
						DaysOfMonth:           schedule.DaysOfMonth,
						ScheduledInstanceMin:  schedule.InstanceMinCount,
						ScheduledInstanceMax:  schedule.InstanceMaxCount,
						ScheduledInstanceInit: schedule.InitialMinInstanceCount,
						StartDate:             schedule.StartDate,
						EndDate:               schedule.EndDate,
					}
					result.ScalingPolicy.Schedules.RecurringSchedules = append(result.ScalingPolicy.Schedules.RecurringSchedules, &rs)
				}
			}

			if params.ScalingPolicy.Schedules.SpecificDate != nil {
				result.ScalingPolicy.Schedules.SpecificDateSchedules = []*models.SpecificDateSchedule{}
				for _, specificDate := range params.ScalingPolicy.Schedules.SpecificDate {
					sd := models.SpecificDateSchedule{
						StartDateTime:         specificDate.StartDateTime,
						EndDateTime:           specificDate.EndDateTime,
						ScheduledInstanceMin:  specificDate.InstanceMinCount,
						ScheduledInstanceMax:  specificDate.InstanceMaxCount,
						ScheduledInstanceInit: specificDate.InitialMinInstanceCount,
					}
					result.ScalingPolicy.Schedules.SpecificDateSchedules = append(result.ScalingPolicy.Schedules.SpecificDateSchedules, &sd)
				}
			}
		}
	}

	return result
}
