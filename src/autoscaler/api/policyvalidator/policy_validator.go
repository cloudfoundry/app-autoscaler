package policyvalidator

import (
	"autoscaler/models"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

const (
	DateTimeLayout = "2006-01-02T15:04"
	DateLayout     = "2006-01-02"
	TimeLayout     = "15:04"
)

type PolicyValidator struct {
	policySchemaPath   string
	policySchemaLoader gojsonschema.JSONLoader
}

type PolicyValidationError struct {
	gojsonschema.ResultErrorFields
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

func NewPolicyValidator(policySchemaPath string) *PolicyValidator {
	policyValidator := &PolicyValidator{
		policySchemaPath: policySchemaPath,
	}
	policyValidator.policySchemaLoader = gojsonschema.NewReferenceLoader("file://" + policyValidator.policySchemaPath)
	return policyValidator
}

func (pv *PolicyValidator) ValidatePolicy(policyStr string) error {
	policyLoader := gojsonschema.NewStringLoader(policyStr)

	result, err := gojsonschema.Validate(pv.policySchemaLoader, policyLoader)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	if !result.Valid() {
		return getErrorWithJSONMessage(result.Errors())
	}

	policy := models.ScalingPolicy{}
	err = json.Unmarshal([]byte(policyStr), &policy)
	if err != nil {
		return err
	}

	pv.validateAttributes(&policy, result)

	if len(result.Errors()) > 0 {
		return getErrorWithJSONMessage(result.Errors())
	}
	return nil
}

func (pv *PolicyValidator) validateAttributes(policy *models.ScalingPolicy, result *gojsonschema.Result) {

	rootContext := gojsonschema.NewJsonContext("(root)", nil)

	//check InstanceMinCount and InstanceMaxCount
	if policy.InstanceMin >= policy.InstanceMax {
		instanceMinContext := gojsonschema.NewJsonContext("instance_min_count", rootContext)
		errDetails := gojsonschema.ErrorDetails{
			"instance_min_count": policy.InstanceMin,
			"instance_max_count": policy.InstanceMax,
		}
		formatString := "instance_min_count {{.instance_min_count}} is higher or equal to instance_max_count {{.instance_max_count}}"
		err := newPolicyValidationError(instanceMinContext, formatString, errDetails)
		result.AddError(err, errDetails)
	}

	// check InstanceMinCount, IntanceMaxCount and InitialInstanceMinCount for schedules
	if policy.Schedules == nil {
		return
	}
	schedulesContext := gojsonschema.NewJsonContext("schedules", rootContext)

	pv.validateRecurringSchedules(policy, schedulesContext, result)
	pv.validateSpecificDateSchedules(policy, schedulesContext, result)
}

func (pv *PolicyValidator) validateRecurringSchedules(policy *models.ScalingPolicy, schedulesContext *gojsonschema.JsonContext, result *gojsonschema.Result) {
	recurringScheduleContext := gojsonschema.NewJsonContext("recurring_schedule", schedulesContext)
	for i, recSched := range policy.Schedules.RecurringSchedules {
		if recSched.ScheduledInstanceMin >= recSched.ScheduledInstanceMax {
			instanceMinContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d.instance_min_count", i), recurringScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"instance_min_count": recSched.ScheduledInstanceMin,
				"instance_max_count": recSched.ScheduledInstanceMin,
			}
			formatString := "instance_min_count {{.instance_min_count}} is higher or equal to instance_max_count {{.instance_max_count}}"
			err := newPolicyValidationError(instanceMinContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}

		if recSched.ScheduledInstanceInit > recSched.ScheduledInstanceMax {
			initialInstanceMinContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d.initial_min_instance_count", i), recurringScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"initial_min_instance_count": recSched.ScheduledInstanceInit,
				"instance_max_count":         recSched.ScheduledInstanceMin,
			}
			formatString := "initial_min_instance_count {{.initial_min_instance_count}} is greater than instance_max_count {{.instance_max_count}}"
			err := newPolicyValidationError(initialInstanceMinContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}

		//start_time should be before end_time
		if compareTimesGTEQ(recSched.StartTime, recSched.EndTime) {
			currentRecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", i), recurringScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"i": i,
			}
			formatString := "start_time is after end_time"
			err := newPolicyValidationError(currentRecSchedContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}
		// start_date should be after current_date and before end_date
		var startDate, endDate time.Time
		location, _ := time.LoadLocation(policy.Schedules.Timezone)

		if recSched.StartDate != "" {
			startDate, _ = time.ParseInLocation(DateLayout, recSched.StartDate, location)

			if startDate.Sub(time.Now()) < 0 {
				currentRecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", i), recurringScheduleContext)
				errDetails := gojsonschema.ErrorDetails{
					"i": i,
				}
				formatString := "start_date is before current_date"
				err := newPolicyValidationError(currentRecSchedContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		}
		if recSched.StartDate != "" {
			endDate, _ = time.ParseInLocation(DateLayout, recSched.EndDate, location)

			if endDate.Sub(time.Now()) < 0 {
				currentRecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", i), recurringScheduleContext)
				errDetails := gojsonschema.ErrorDetails{
					"i": i,
				}
				formatString := "end_date is before current_date"
				err := newPolicyValidationError(currentRecSchedContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		}
		if recSched.StartDate != "" && recSched.EndDate != "" {
			startDate, _ = time.ParseInLocation(DateLayout, recSched.StartDate, location)
			endDate, _ = time.ParseInLocation(DateLayout, recSched.EndDate, location)
			if endDate.Sub(startDate) < 0 {

				currentRecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", i), recurringScheduleContext)
				errDetails := gojsonschema.ErrorDetails{
					"i": i,
				}
				formatString := "start_date is after end_date"
				err := newPolicyValidationError(currentRecSchedContext, formatString, errDetails)
				result.AddError(err, errDetails)
			}
		}
	}

	pv.validateOverlappingInRecurringSchedules(policy, recurringScheduleContext, result)
}

func (pv *PolicyValidator) validateSpecificDateSchedules(policy *models.ScalingPolicy, schedulesContext *gojsonschema.JsonContext, result *gojsonschema.Result) {
	specficDateScheduleContext := gojsonschema.NewJsonContext("specific_date", schedulesContext)
	for i, specSched := range policy.Schedules.SpecificDateSchedules {
		if specSched.ScheduledInstanceMin >= specSched.ScheduledInstanceMax {
			instanceMinContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d.instance_min_count", i), specficDateScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"instance_min_count": specSched.ScheduledInstanceMin,
				"instance_max_count": specSched.ScheduledInstanceMax,
			}
			formatString := "instance_min_count {{.instance_min_count}} is higher or equal to instance_max_count {{.instance_max_count}}"
			err := newPolicyValidationError(instanceMinContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}
		if specSched.ScheduledInstanceInit > specSched.ScheduledInstanceMax {
			initialInstanceMinContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d.initial_min_instance_count", i), specficDateScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"initial_min_instance_count": specSched.ScheduledInstanceInit,
				"instance_max_count":         specSched.ScheduledInstanceMin,
			}
			formatString := "initial_min_instance_count {{.initial_min_instance_count}} is greater than instance_max_count {{.instance_max_count}}"
			err := newPolicyValidationError(initialInstanceMinContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}

		// start_date_time should be after current_date_time and before end_date_time
		dateTime := newDateTimeRange(specSched.StartDateTime, specSched.EndDateTime, policy.Schedules.Timezone)
		if dateTime.startDateTime.Sub(time.Now()) <= 0 {
			currentSpecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", i), specficDateScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"i": i,
			}
			formatString := "start_date_time is before current_date_time"
			err := newPolicyValidationError(currentSpecSchedContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}
		if dateTime.endDateTime.Sub(dateTime.startDateTime) <= 0 {
			currentSpecSchedContext := gojsonschema.NewJsonContext(fmt.Sprintf("%d", i), specficDateScheduleContext)
			errDetails := gojsonschema.ErrorDetails{
				"i": i,
			}
			formatString := "start_date_time is after end_date_time"
			err := newPolicyValidationError(currentSpecSchedContext, formatString, errDetails)
			result.AddError(err, errDetails)
		}
	}

	pv.validateOverlappingInSpecificDateSchedules(policy, specficDateScheduleContext, result)
}

func (pv *PolicyValidator) validateOverlappingInRecurringSchedules(policy *models.ScalingPolicy, recurringScheduleContext *gojsonschema.JsonContext, result *gojsonschema.Result) {
	length := len(policy.Schedules.RecurringSchedules)
	recScheds := policy.Schedules.RecurringSchedules
	for j := 0; j < length-1; j++ {
		for i := j + 1; i < length; i++ {
			if (recScheds[i].DaysOfWeek != nil && len(recScheds[i].DaysOfWeek) > 0) && (recScheds[j].DaysOfWeek != nil && len(recScheds[j].DaysOfWeek) > 0) {
				if hasIntersection(recScheds[i].DaysOfWeek, recScheds[j].DaysOfWeek) {
					if compareTimesGTEQ(recScheds[j].EndTime, recScheds[i].StartTime) && compareTimesGTEQ(recScheds[i].EndTime, recScheds[j].StartTime) &&
						compareDatesGTEQ(recScheds[j].EndDate, recScheds[i].StartDate) && compareDatesGTEQ(recScheds[i].EndDate, recScheds[j].StartDate) {
						context := gojsonschema.NewJsonContext(fmt.Sprintf("%d", j), recurringScheduleContext)
						errDetails := gojsonschema.ErrorDetails{
							"i": i,
							"j": j,
						}

						formatString := "recurring_schedule[{{.j}}] and recurring_schedule[{{.i}}] are overlapping"
						err := newPolicyValidationError(context, formatString, errDetails)
						result.AddError(err, errDetails)
					}
				}
			}

			if (recScheds[i].DaysOfMonth != nil && len(recScheds[i].DaysOfMonth) > 0) && (recScheds[j].DaysOfMonth != nil && len(recScheds[j].DaysOfMonth) > 0) {
				if hasIntersection(recScheds[i].DaysOfMonth, recScheds[j].DaysOfMonth) {
					if compareTimesGTEQ(recScheds[j].EndTime, recScheds[i].StartTime) && compareTimesGTEQ(recScheds[i].EndTime, recScheds[j].StartTime) &&
						compareDatesGTEQ(recScheds[j].EndDate, recScheds[i].StartDate) && compareDatesGTEQ(recScheds[i].EndDate, recScheds[j].StartDate) {
						context := gojsonschema.NewJsonContext(fmt.Sprintf("%d", j), recurringScheduleContext)
						errDetails := gojsonschema.ErrorDetails{
							"i": i,
							"j": j,
						}

						formatString := "recurring_schedule[{{.j}}] and recurring_schedule[{{.i}}] are overlapping"
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

	for j := 0; j < length; j++ {
		for i := j + 1; i < length; i++ {
			if dateTimeRangeList[j].overlaps(dateTimeRangeList[i]) {
				context := gojsonschema.NewJsonContext(fmt.Sprintf("%d", j), specficDateScheduleContext)
				errDetails := gojsonschema.ErrorDetails{
					"i":                i,
					"j":                j,
					"start_date_time1": policy.Schedules.SpecificDateSchedules[j].StartDateTime,
					"end_date_time1":   policy.Schedules.SpecificDateSchedules[j].StartDateTime,
					"start_date_time2": policy.Schedules.SpecificDateSchedules[i].StartDateTime,
					"end_date_time2":   policy.Schedules.SpecificDateSchedules[i].StartDateTime,
				}

				formatString := "specific_date[{{.j}}]:{start_date_time: {{.start_date_time1}}, end_date_time: {{.end_date_time1}}} and specific_date[{{.i}}]:{start_date_time: {{.start_date_time2}}, end_date_time: {{.end_date_time2}}} are overlapping"
				err := newPolicyValidationError(context, formatString, errDetails)
				result.AddError(err, errDetails)

			}
		}
	}

}

func getErrorWithJSONMessage(resErr []gojsonschema.ResultError) error {
	errString := "["
	for index, desc := range resErr {
		if index == len(resErr)-1 {
			errString += fmt.Sprintf("{\"context\": \"%s\", \"description\": \"%s\"}", desc.Context().String(), desc.Description())
		} else {
			errString += fmt.Sprintf("{\"context\": \"%s\", \"description\": \"%s\"},", desc.Context().String(), desc.Description())
		}
	}
	errString += "]"
	return fmt.Errorf(errString)
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
	if ft.Sub(st) >= 0 {
		return true
	}
	return false
}

func compareDatesGTEQ(firstDate string, secondDate string) bool {
	if firstDate == "" {
		firstDate = "0001-01-01"
	}
	if secondDate == "" {
		secondDate = "0001-01-01"
	}
	fd, _ := time.Parse(DateLayout, firstDate)
	sd, _ := time.Parse(DateLayout, secondDate)
	if fd.Sub(sd) >= 0 {
		return true
	}
	return false
}
