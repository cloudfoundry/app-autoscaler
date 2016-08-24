package models

type ScalingPolicy struct {
	InstanceMin int           `json:"instance_min_count"`
	InstanceMax int           `json:"instance_max_count"`
	Rules       []ScalingRule `json:"scaling_rules"`
}

type ScalingRule struct {
	MetricType            string `json:"metric_type"`
	BreachDurationSeconds int    `json:"breach_duration_secs"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	CoolDownSeconds       int    `json:"cool_down_secs"`
	Adjustment            string `json:"adjustment"`
}
