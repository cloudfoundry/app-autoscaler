package v0_1

type parameters struct {
	SchemaVersion  string           `json:"schema-version"`
	CredentialType string           `json:"credential-type,omitempty"`
	Configuration  *bindingCfg      `json:"configuration,omitempty"`
	InstanceMin    int              `json:"instance_min_count,omitempty"`
	InstanceMax    int              `json:"instance_max_count,omitempty"`
	ScalingRules   []scalingRule    `json:"scaling_rules,omitempty"`
	Schedules      *scalingSchedule `json:"schedules,omitempty"`
}

type bindingCfg struct {
	CustomMetrics *customMetricsCfg `json:"custom_metrics,omitempty"`
	AppGuid       string            `json:"app_guid,omitempty"`
}

type customMetricsCfg struct {
	MetricSubmissionStrategy metricSubmStrat `json:"metric_submission_strategy"`
}

type metricSubmStrat struct {
	AllowFrom string `json:"allow_from"`
}

type scalingRule struct {
	MetricType         string `json:"metric_type"`
	BreachDurationSecs int    `json:"breach_duration_secs,omitempty"`
	StatsWindowSecs    int    `json:"stats_window_secs,omitempty"`
	Threshold          int64  `json:"threshold"`
	Operator           string `json:"operator"`
	CoolDownSecs       int    `json:"cool_down_secs,omitempty"`
	Adjustment         string `json:"adjustment"`
}

type scalingSchedule struct {
	Timezone          string              `json:"timezone"`
	RecurringSchedule []recurringSchedule `json:"recurring_schedule,omitempty"`
	SpecificDate      []specificDate      `json:"specific_date,omitempty"`
}

type recurringSchedule struct {
	StartTime               string `json:"start_time"`
	EndTime                 string `json:"end_time"`
	DaysOfWeek              []int  `json:"days_of_week,omitempty"`
	DaysOfMonth             []int  `json:"days_of_month,omitempty"`
	InstanceMinCount        int    `json:"instance_min_count"`
	InstanceMaxCount        int    `json:"instance_max_count"`
	InitialMinInstanceCount int    `json:"initial_min_instance_count,omitempty"`
	StartDate               string `json:"start_date,omitempty"`
	EndDate                 string `json:"end_date,omitempty"`
}

type specificDate struct {
	StartDateTime           string `json:"start_date_time"`
	EndDateTime             string `json:"end_date_time"`
	InstanceMinCount        int    `json:"instance_min_count"`
	InstanceMaxCount        int    `json:"instance_max_count"`
	InitialMinInstanceCount int    `json:"initial_min_instance_count,omitempty"`
}
