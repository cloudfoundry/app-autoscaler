package policyvalidator

import (
	"encoding/json"
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"github.com/xeipuuv/gojsonschema"
)

const (
	DateTimeLayout = "2006-01-02T15:04"
	DateLayout     = "2006-01-02"
	TimeLayout     = "15:04"
)

type ScalingRulesConfig struct {
	CPU CPUConfig
}

type CPUConfig struct {
	LowerThreshold int
	UpperThreshold int
}

type PolicyValidator struct {
	scalingRules       ScalingRulesConfig
	policySchemaPath   string
	policySchemaLoader gojsonschema.JSONLoader
}

type PolicyValidationError struct {
	gojsonschema.ResultErrorFields
}

type PolicyValidationErrors struct {
	Context     string `json:"context"`
	Description string `json:"description"`
}

type DateTimeRange struct {
	startDateTime time.Time
	endDateTime   time.Time
}

func newDateTimeRange(startDateTime string, endDateTime string, timezone string) *DateTimeRange {
	location, _ := time.LoadLocation(timezone)
	dateTimeRange := DateTimeRange{}
	dateTimeRange.startDateTime, _ = time.ParseInLocation(DateTimeLayout, startDateTime, location)
	dateTimeRange.endDateTime, _ = time.ParseInLocation(DateTimeLayout, endDateTime, location)
	return &dateTimeRange
}

func (dtr *DateTimeRange) overlaps(otherDtr *DateTimeRange) bool {
	if otherDtr.endDateTime.Sub(dtr.startDateTime) > 0 && dtr.endDateTime.Sub(otherDtr.startDateTime) > 0 {
		return true
	}
	return false
}

func newPolicyValidationError(context *gojsonschema.JsonContext, formatString string, errDetails gojsonschema.ErrorDetails) *PolicyValidationError {
	err := PolicyValidationError{}
	err.SetType("custom_invalid_policy_error")
	err.SetContext(context)
	err.SetDescriptionFormat(formatString)
	err.SetDetails(errDetails)
	return &err
}

func NewPolicyValidator(policySchemaPath string, lowerThreshold int, upperThreshold int) *PolicyValidator {
	policyValidator := &PolicyValidator{
		policySchemaPath: policySchemaPath,
		scalingRules: ScalingRulesConfig{
			CPU: CPUConfig{
				LowerThreshold: lowerThreshold,
				UpperThreshold: upperThreshold,
			},
		},
	}
	policyValidator.policySchemaLoader = gojsonschema.NewReferenceLoader("file://" + policyValidator.policySchemaPath)
	return policyValidator
}

func (pv *PolicyValidator) ValidatePolicy(policyStr string) (*[]PolicyValidationErrors, bool, string) {
	policyLoader := gojsonschema.NewStringLoader(policyStr)

	result, err := gojsonschema.Validate(pv.policySchemaLoader, policyLoader)
	if err != nil {
		resultErrors := []PolicyValidationErrors{
			{Context: "(root)", Description: err.Error()},
		}
		return &resultErrors, false, ""
	}

	if !result.Valid() {
		return getErrorsObject(result.Errors()), false, ""
	}

	policy := models.ScalingPolicy{}
	err = json.Unmarshal([]byte(policyStr), &policy)
	if err != nil {
		resultErrors := []PolicyValidationErrors{
			{Context: "(root)", Description: err.Error()},
		}
		return &resultErrors, false, ""
	}

	pv.validateAttributes(&policy, result)

	if len(result.Errors()) > 0 {
		return getErrorsObject(result.Errors()), false, ""
	}

	validatedPolicyStr, err := json.Marshal(policy)
	if err != nil {
		resultErrors := []PolicyValidationErrors{
			{Context: "(root)", Description: err.Error()},
		}
		return &resultErrors, false, ""
	}

	return nil, true, string(validatedPolicyStr)
}

func (pv *PolicyValidator) validateAttributes(policy *models.ScalingPolicy, result *gojsonschema.Result) {
	rootContext := gojsonschema.NewJsonContext("(root)", nil)

	//check InstanceMinCount and InstanceMaxCount
	if policy.InstanceMin > policy.InstanceMax {
		instanceMinContext := gojsonschema.NewJsonContext("instance_min_count", rootContext)
		errDetails := gojsonschema.ErrorDetails{
			"instance_min_count": policy.InstanceMin,
			"instance_max_count": policy.InstanceMax,
		}
		formatString := "instance_min_count {{.instance_min_count}} is higher than instance_max_count {{.instance_max_count}}"
		err := newPolicyValidationError(instanceMinContext, formatString, errDetails)
		result.AddError(err, errDetails)
	}

	scalingRulesContext := gojsonschema.NewJsonContext("scaling_rules", rootContext)
	pv.validateScalingRuleThreshold(policy, scalingRulesContext, result)

	if policy.Schedules == nil {
		return
	}
	schedulesContext := gojsonschema.NewJsonContext("schedules", rootContext)

	pv.validateRecurringSchedules(policy, schedulesContext, result)
	pv.validateSpecificDateSchedules(policy, schedulesContext, result)
}

func (pv *PolicyValidator) validateScalingRuleThreshold(policy *models.ScalingPolicy, scalingRulesContext *gojsonschema.JsonContext, result *gojsonschema.Result) {
	for srIndex, scalingRule := range policy.ScalingRules {
		currentContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", srIndex), scalingRulesContext)
		errDetails := gojsonschema.ErrorDetails{
			"scalingRuleIndex": srIndex,
		}

		switch scalingRule.MetricType {
		case "memoryused":
			if scalingRule.Threshold <= 0 {
				formatString := "scaling_rules[{{.scalingRuleIndex}}].threshold for metric_type memoryused should be greater than 0"
				err := newPolicyValidationError(currentContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		case "memoryutil":
			if scalingRule.Threshold <= 0 || scalingRule.Threshold > 100 {
				formatString := "scaling_rules[{{.scalingRuleIndex}}].threshold for metric_type memoryutil should be greater than 0 and less than equal to 100"
				err := newPolicyValidationError(currentContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		case "responsetime":
			if scalingRule.Threshold <= 0 {
				formatString := "scaling_rules[{{.scalingRuleIndex}}].threshold for metric_type responsetime should be greater than 0"
				err := newPolicyValidationError(currentContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		case "throughput":
			if scalingRule.Threshold <= 0 {
				formatString := "scaling_rules[{{.scalingRuleIndex}}].threshold for metric_type throughput should be greater than 0"
				err := newPolicyValidationError(currentContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		case "cpu":
			if scalingRule.Threshold < int64(pv.scalingRules.CPU.LowerThreshold) || scalingRule.Threshold >= int64(pv.scalingRules.CPU.UpperThreshold) {
				formatString := fmt.Sprintf("scaling_rules[{{.scalingRuleIndex}}].threshold for metric_type cpu should be greater than %d and less than or equal to %d", pv.scalingRules.CPU.LowerThreshold, pv.scalingRules.CPU.UpperThreshold)
				err := newPolicyValidationError(currentContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		default:
		}
	}
}

func (pv *PolicyValidator) validateRecurringSchedules(policy *models.ScalingPolicy, schedulesContext *gojsonschema.JsonContext, result *gojsonschema.Result) {
	recurringScheduleContext := gojsonschema.NewJsonContext("recurring_schedule", schedulesContext)
	for scheduleIndex, recSched := range policy.Schedules.RecurringSchedules {
		if recSched.ScheduledInstanceMin > recSched.ScheduledInstanceMax {
			instanceMinContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d.instance_min_count", scheduleIndex), recurringScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"scheduleIndex":      scheduleIndex,
				"instance_min_count": recSched.ScheduledInstanceMin,
				"instance_max_count": recSched.ScheduledInstanceMax,
			}
			formatString := "recurring_schedule[{{.scheduleIndex}}].instance_min_count {{.instance_min_count}} is higher than recurring_schedule[{{.scheduleIndex}}].instance_max_count {{.instance_max_count}}"
			err := newPolicyValidationError(instanceMinContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}

		if (recSched.ScheduledInstanceInit != 0) && (recSched.ScheduledInstanceInit < recSched.ScheduledInstanceMin) {
			initialInstanceMinContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d.initial_min_instance_count", scheduleIndex), recurringScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"scheduleIndex":              scheduleIndex,
				"initial_min_instance_count": recSched.ScheduledInstanceInit,
				"instance_min_count":         recSched.ScheduledInstanceMin,
			}
			formatString := "recurring_schedule[{{.scheduleIndex}}].initial_min_instance_count {{.initial_min_instance_count}} is smaller than recurring_schedule[{{.scheduleIndex}}].instance_min_count {{.instance_min_count}}"
			err := newPolicyValidationError(initialInstanceMinContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}

		if (recSched.ScheduledInstanceInit != 0) && (recSched.ScheduledInstanceInit > recSched.ScheduledInstanceMax) {
			initialInstanceMinContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d.initial_min_instance_count", scheduleIndex), recurringScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"scheduleIndex":              scheduleIndex,
				"initial_min_instance_count": recSched.ScheduledInstanceInit,
				"instance_max_count":         recSched.ScheduledInstanceMax,
			}
			formatString := "recurring_schedule[{{.scheduleIndex}}].initial_min_instance_count {{.initial_min_instance_count}} is greater than recurring_schedule[{{.scheduleIndex}}].instance_max_count {{.instance_max_count}}"
			err := newPolicyValidationError(initialInstanceMinContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}

		//start_time should be before end_time
		if compareTimesGTEQ(recSched.StartTime, recSched.EndTime) {
			currentRecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", scheduleIndex), recurringScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"scheduleIndex": scheduleIndex,
			}
			formatString := "recurring_schedule[{{.scheduleIndex}}].start_time is same or after recurring_schedule[{{.scheduleIndex}}].end_time"
			err := newPolicyValidationError(currentRecSchedContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}
		// start_date should be after current_date and before end_date
		var startDate, endDate time.Time
		location, _ := time.LoadLocation(policy.Schedules.Timezone)

		currentDate, _ := time.ParseInLocation(DateLayout, time.Now().Format(DateLayout), location)
		if recSched.StartDate != "" {
			startDate, _ = time.ParseInLocation(DateLayout, recSched.StartDate, location)

			if startDate.Sub(currentDate) < 0 {
				currentRecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", scheduleIndex), recurringScheduleContext)
				errDetails := gojsonschema.ErrorDetails{
					"scheduleIndex": scheduleIndex,
				}
				formatString := "recurring_schedule[{{.scheduleIndex}}].start_date is before recurring_schedule[{{.scheduleIndex}}].current_date"
				err := newPolicyValidationError(currentRecSchedContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		}
		if recSched.EndDate != "" {
			endDate, _ = time.ParseInLocation(DateLayout, recSched.EndDate, location)

			if endDate.Sub(currentDate) < 0 {
				currentRecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", scheduleIndex), recurringScheduleContext)
				errDetails := gojsonschema.ErrorDetails{
					"scheduleIndex": scheduleIndex,
				}
				formatString := "recurring_schedule[{{.scheduleIndex}}].end_date is before recurring_schedule[{{.scheduleIndex}}].current_date"
				err := newPolicyValidationError(currentRecSchedContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		}

		if recSched.StartDate != "" && recSched.EndDate != "" {
			startDate, _ = time.ParseInLocation(DateLayout, recSched.StartDate, location)
			endDate, _ = time.ParseInLocation(DateLayout, recSched.EndDate, location)
			if endDate.Sub(startDate) < 0 {
				currentRecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", scheduleIndex), recurringScheduleContext)
				errDetails := gojsonschema.ErrorDetails{
					"scheduleIndex": scheduleIndex,
				}
				formatString := "recurring_schedule[{{.scheduleIndex}}].start_date is after recurring_schedule[{{.scheduleIndex}}].end_date"
				err := newPolicyValidationError(currentRecSchedContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		}
	}

	pv.validateOverlappingInRecurringSchedules(policy, recurringScheduleContext, result)
}

func (pv *PolicyValidator) validateSpecificDateSchedules(policy *models.ScalingPolicy, schedulesContext *gojsonschema.JsonContext, result *gojsonschema.Result) {
	specficDateScheduleContext := gojsonschema.NewJsonContext("specific_date", schedulesContext)
	for scheduleIndex, specSched := range policy.Schedules.SpecificDateSchedules {
		if specSched.ScheduledInstanceMin > specSched.ScheduledInstanceMax {
			instanceMinContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d.instance_min_count", scheduleIndex), specficDateScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"scheduleIndex":      scheduleIndex,
				"instance_min_count": specSched.ScheduledInstanceMin,
				"instance_max_count": specSched.ScheduledInstanceMax,
			}
			formatString := "specific_date[{{.scheduleIndex}}].instance_min_count {{.instance_min_count}} is higher than specific_date[{{.scheduleIndex}}].instance_max_count {{.instance_max_count}}"
			err := newPolicyValidationError(instanceMinContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}
		if (specSched.ScheduledInstanceInit != 0) && (specSched.ScheduledInstanceInit < specSched.ScheduledInstanceMin) {
			initialInstanceMinContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d.initial_min_instance_count", scheduleIndex), specficDateScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"scheduleIndex":              scheduleIndex,
				"initial_min_instance_count": specSched.ScheduledInstanceInit,
				"instance_min_count":         specSched.ScheduledInstanceMin,
			}
			formatString := "specific_date[{{.scheduleIndex}}].initial_min_instance_count {{.initial_min_instance_count}} is smaller than specific_date[{{.scheduleIndex}}].instance_min_count {{.instance_min_count}}"
			err := newPolicyValidationError(initialInstanceMinContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}

		if (specSched.ScheduledInstanceInit != 0) && (specSched.ScheduledInstanceInit > specSched.ScheduledInstanceMax) {
			initialInstanceMinContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d.initial_min_instance_count", scheduleIndex), specficDateScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"scheduleIndex":              scheduleIndex,
				"initial_min_instance_count": specSched.ScheduledInstanceInit,
				"instance_max_count":         specSched.ScheduledInstanceMax,
			}
			formatString := "specific_date[{{.scheduleIndex}}].initial_min_instance_count {{.initial_min_instance_count}} is greater than specific_date[{{.scheduleIndex}}].instance_max_count {{.instance_max_count}}"
			err := newPolicyValidationError(initialInstanceMinContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}

		// start_date_time should be after current_date_time and before end_date_time
		dateTime := newDateTimeRange(specSched.StartDateTime, specSched.EndDateTime, policy.Schedules.Timezone)
		if time.Until(dateTime.startDateTime) <= 0 {
			currentSpecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", scheduleIndex), specficDateScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"scheduleIndex": scheduleIndex,
			}
			formatString := "specific_date[{{.scheduleIndex}}].start_date_time is before current date time"
			err := newPolicyValidationError(currentSpecSchedContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}
		if dateTime.endDateTime.Sub(dateTime.startDateTime) <= 0 {
			currentSpecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", scheduleIndex), specficDateScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"scheduleIndex": scheduleIndex,
			}
			formatString := "specific_date[{{.scheduleIndex}}].start_date_time is after specific_date[{{.scheduleIndex}}].end_date_time"
			err := newPolicyValidationError(currentSpecSchedContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}
	}

	pv.validateOverlappingInSpecificDateSchedules(policy, specficDateScheduleContext, result)
}

func (pv *PolicyValidator) validateOverlappingInRecurringSchedules(policy *models.ScalingPolicy, recurringScheduleContext *gojsonschema.JsonContext, result *gojsonschema.Result) {
	length := len(policy.Schedules.RecurringSchedules)
	recScheds := policy.Schedules.RecurringSchedules
	for scheduleIndexB := 0; scheduleIndexB < length-1; scheduleIndexB++ {
		for scheduleIndexA := scheduleIndexB + 1; scheduleIndexA < length; scheduleIndexA++ {
			if (recScheds[scheduleIndexA].DaysOfWeek != nil && len(recScheds[scheduleIndexA].DaysOfWeek) > 0) && (recScheds[scheduleIndexB].DaysOfWeek != nil && len(recScheds[scheduleIndexB].DaysOfWeek) > 0) {
				if hasIntersection(recScheds[scheduleIndexA].DaysOfWeek, recScheds[scheduleIndexB].DaysOfWeek) {
					if compareTimesGTEQ(recScheds[scheduleIndexB].EndTime, recScheds[scheduleIndexA].StartTime) && compareTimesGTEQ(recScheds[scheduleIndexA].EndTime, recScheds[scheduleIndexB].StartTime) &&
						compareDatesGTEQ(recScheds[scheduleIndexB].EndDate, recScheds[scheduleIndexA].StartDate) && compareDatesGTEQ(recScheds[scheduleIndexA].EndDate, recScheds[scheduleIndexB].StartDate) {
						context := gojsonschema.NewJsonContext(fmt.Sprintf("%d", scheduleIndexB), recurringScheduleContext)
						errDetails := gojsonschema.ErrorDetails{
							"scheduleIndexA": scheduleIndexA,
							"scheduleIndexB": scheduleIndexB,
						}

						formatString := "recurring_schedule[{{.scheduleIndexB}}] and recurring_schedule[{{.scheduleIndexA}}] are overlapping"
						err := newPolicyValidationError(context, formatString, errDetails)
						result.AddError(err, errDetails)
					}
				}
			}

			if (recScheds[scheduleIndexA].DaysOfMonth != nil && len(recScheds[scheduleIndexA].DaysOfMonth) > 0) && (recScheds[scheduleIndexB].DaysOfMonth != nil && len(recScheds[scheduleIndexB].DaysOfMonth) > 0) {
				if hasIntersection(recScheds[scheduleIndexA].DaysOfMonth, recScheds[scheduleIndexB].DaysOfMonth) {
					if compareTimesGTEQ(recScheds[scheduleIndexB].EndTime, recScheds[scheduleIndexA].StartTime) && compareTimesGTEQ(recScheds[scheduleIndexA].EndTime, recScheds[scheduleIndexB].StartTime) &&
						compareDatesGTEQ(recScheds[scheduleIndexB].EndDate, recScheds[scheduleIndexA].StartDate) && compareDatesGTEQ(recScheds[scheduleIndexA].EndDate, recScheds[scheduleIndexB].StartDate) {
						context := gojsonschema.NewJsonContext(fmt.Sprintf("%d", scheduleIndexB), recurringScheduleContext)
						errDetails := gojsonschema.ErrorDetails{
							"scheduleIndexA": scheduleIndexA,
							"scheduleIndexB": scheduleIndexB,
						}

						formatString := "recurring_schedule[{{.scheduleIndexB}}] and recurring_schedule[{{.scheduleIndexA}}] are overlapping"
						err := newPolicyValidationError(context, formatString, errDetails)
						result.AddError(err, errDetails)
					}
				}
			}
		}
	}
}

func (pv *PolicyValidator) validateOverlappingInSpecificDateSchedules(policy *models.ScalingPolicy, specficDateScheduleContext *gojsonschema.JsonContext, result *gojsonschema.Result) {
	length := len(policy.Schedules.SpecificDateSchedules)
	var dateTimeRangeList []*DateTimeRange
	for _, specSched := range policy.Schedules.SpecificDateSchedules {
		dateTimeRangeList = append(dateTimeRangeList, newDateTimeRange(specSched.StartDateTime, specSched.EndDateTime, policy.Schedules.Timezone))
	}

	for scheduleIndexB := 0; scheduleIndexB < length; scheduleIndexB++ {
		for scheduleIndexA := scheduleIndexB + 1; scheduleIndexA < length; scheduleIndexA++ {
			if dateTimeRangeList[scheduleIndexB].overlaps(dateTimeRangeList[scheduleIndexA]) {
				context := gojsonschema.NewJsonContext(fmt.Sprintf("%d", scheduleIndexB), specficDateScheduleContext)
				errDetails := gojsonschema.ErrorDetails{
					"scheduleIndexA":   scheduleIndexA,
					"scheduleIndexB":   scheduleIndexB,
					"start_date_time1": policy.Schedules.SpecificDateSchedules[scheduleIndexB].StartDateTime,
					"end_date_time1":   policy.Schedules.SpecificDateSchedules[scheduleIndexB].StartDateTime,
					"start_date_time2": policy.Schedules.SpecificDateSchedules[scheduleIndexA].StartDateTime,
					"end_date_time2":   policy.Schedules.SpecificDateSchedules[scheduleIndexA].StartDateTime,
				}

				formatString := "specific_date[{{.scheduleIndexB}}]:{start_date_time: {{.start_date_time1}}, end_date_time: {{.end_date_time1}}} and specific_date[{{.scheduleIndexA}}]:{start_date_time: {{.start_date_time2}}, end_date_time: {{.end_date_time2}}} are overlapping"
				err := newPolicyValidationError(context, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		}
	}
}

func getErrorsObject(resErr []gojsonschema.ResultError) *[]PolicyValidationErrors {
	policyValidationErrorsResult := []PolicyValidationErrors{}
	for _, err := range resErr {
		policyValidationErrorsResult = append(policyValidationErrorsResult, PolicyValidationErrors{
			Context:     err.Context().String(),
			Description: err.Description(),
		})
	}
	return &policyValidationErrorsResult
}

func hasIntersection(a []int, b []int) bool {
	m := make(map[int]bool)
	for _, item := range a {
		m[item] = true
	}
	for _, item := range b {
		if _, ok := m[item]; ok {
			return true
		}
	}
	return false
}

func compareTimesGTEQ(firstTime string, secondTime string) bool {
	ft, _ := time.Parse(TimeLayout, firstTime)
	st, _ := time.Parse(TimeLayout, secondTime)
	return ft.Sub(st) >= 0
}

func compareDatesGTEQ(endDate string, startDate string) bool {
	if endDate == "" {
		endDate = "9999-01-01"
	}
	if startDate == "" {
		startDate = "0000-01-01"
	}
	fd, _ := time.Parse(DateLayout, endDate)
	sd, _ := time.Parse(DateLayout, startDate)
	return fd.Sub(sd) >= 0
}
