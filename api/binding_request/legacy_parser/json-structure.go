package legacy_parser

import "code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

type policyAndBindingCfg struct {
	BindingConfig bindingConfig     `json:"configuration"`
	InstanceMin   int               `json:"instance_min_count"`
	InstanceMax   int               `json:"instance_max_count"`
	ScalingRules  []*scalingRule    `json:"scaling_rules,omitempty"`
	Schedules     *scalingSchedules `json:"schedules,omitempty"`
}

// ================================================================================
// Binding-configuration
// ================================================================================

type bindingConfig struct {
	AppGUID       models.GUID         `json:"app_guid,omitempty"` // Empty value represents null-value (i.e. not set).
	CustomMetrics customMetricsConfig `json:"custom_metrics,omitempty"`
}

type customMetricsConfig struct {
	MetricSubmissionStrategy metricsSubmissionStrategy `json:"metric_submission_strategy"`
}

type metricsSubmissionStrategy struct {
	AllowFrom string `json:"allow_from"`
}

// ================================================================================
// Scaling-policy details
// ================================================================================

type scalingRule struct {
	MetricType            string `json:"metric_type"`
	BreachDurationSeconds int    `json:"breach_duration_secs,omitempty"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	CoolDownSeconds       int    `json:"cool_down_secs,omitempty"`
	Adjustment            string `json:"adjustment"`
}

type scalingSchedules struct {
	Timezone              string                  `json:"timezone"`
	RecurringSchedules    []*recurringSchedule    `json:"recurring_schedule,omitempty"`
	SpecificDateSchedules []*specificDateSchedule `json:"specific_date,omitempty"`
}

type recurringSchedule struct {
	StartTime             string `json:"start_time"`
	EndTime               string `json:"end_time"`
	DaysOfWeek            []int  `json:"days_of_week,omitempty"`
	DaysOfMonth           []int  `json:"days_of_month,omitempty"`
	StartDate             string `json:"start_date,omitempty"`
	EndDate               string `json:"end_date,omitempty"`
	ScheduledInstanceMin  int    `json:"instance_min_count"`
	ScheduledInstanceMax  int    `json:"instance_max_count"`
	ScheduledInstanceInit int    `json:"initial_min_instance_count,omitempty"`
}

type specificDateSchedule struct {
	StartDateTime         string `json:"start_date_time"`
	EndDateTime           string `json:"end_date_time"`
	ScheduledInstanceMin  int    `json:"instance_min_count"`
	ScheduledInstanceMax  int    `json:"instance_max_count"`
	ScheduledInstanceInit int    `json:"initial_min_instance_count,omitempty"`
}
