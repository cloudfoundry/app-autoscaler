package models

import (
	"encoding/json"
	"time"
)

type AppPolicy struct {
	AppId         string
	ScalingPolicy *ScalingPolicy
}

type PolicyJson struct {
	AppId     string
	PolicyStr string
}

func (p1 *PolicyJson) Equals(p2 *PolicyJson) bool {
	if p1 == p2 {
		return true
	} else if p1 != nil && p2 != nil {
		if p1.AppId == p2.AppId && p1.PolicyStr == p2.PolicyStr {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}
func (p *PolicyJson) GetAppPolicy() *AppPolicy {
	scalingPolicy := ScalingPolicy{}
	json.Unmarshal([]byte(p.PolicyStr), &scalingPolicy)
	return &AppPolicy{AppId: p.AppId, ScalingPolicy: &scalingPolicy}
}

type ScalingPolicy struct {
	InstanceMin  int               `json:"instance_min_count"`
	InstanceMax  int               `json:"instance_max_count"`
	ScalingRules []*ScalingRule    `json:"scaling_rules,omitempty"`
	Schedules    *ScalingSchedules `json:"schedules,omitempty"`
}

type ScalingRule struct {
	MetricType            string `json:"metric_type"`
	BreachDurationSeconds int    `json:"breach_duration_secs,omitempty"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	CoolDownSeconds       int    `json:"cool_down_secs,omitempty"`
	Adjustment            string `json:"adjustment"`
}

type ScalingSchedules struct {
	Timezone              string                  `json:"timezone"`
	RecurringSchedules    []*RecurringSchedule    `json:"recurring_schedule,omitempty"`
	SpecificDateSchedules []*SpecificDateSchedule `json:"specific_date,omitempty"`
}

type RecurringSchedule struct {
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

type SpecificDateSchedule struct {
	StartDateTime         string `json:"start_date_time"`
	EndDateTime           string `json:"end_date_time"`
	ScheduledInstanceMin  int    `json:"instance_min_count"`
	ScheduledInstanceMax  int    `json:"instance_max_count"`
	ScheduledInstanceInit int    `json:"initial_min_instance_count,omitempty"`
}

func (r *ScalingRule) BreachDuration(defaultBreachDurationSecs int) time.Duration {
	if r.BreachDurationSeconds <= 0 {
		return time.Duration(defaultBreachDurationSecs) * time.Second
	}
	return time.Duration(r.BreachDurationSeconds) * time.Second
}

func (r *ScalingRule) CoolDown(defaultCoolDownSecs int) time.Duration {
	if r.CoolDownSeconds <= 0 {
		return time.Duration(defaultCoolDownSecs) * time.Second
	}
	return time.Duration(r.CoolDownSeconds) * time.Second
}

type Trigger struct {
	AppId                 string `json:"app_id"`
	MetricType            string `json:"metric_type"`
	MetricUnit            string `json:"metric_unit"`
	BreachDurationSeconds int    `json:"breach_duration_secs"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	CoolDownSeconds       int    `json:"cool_down_secs"`
	Adjustment            string `json:"adjustment"`
}

func (t Trigger) BreachDuration() time.Duration {
	return time.Duration(t.BreachDurationSeconds) * time.Second
}

func (t Trigger) CoolDown(defaultCoolDownSecs int) time.Duration {
	if t.CoolDownSeconds <= 0 {
		return time.Duration(defaultCoolDownSecs) * time.Second
	}
	return time.Duration(t.CoolDownSeconds) * time.Second
}

type ActiveSchedule struct {
	ScheduleId         string
	InstanceMin        int `json:"instance_min_count"`
	InstanceMax        int `json:"instance_max_count"`
	InstanceMinInitial int `json:"initial_min_instance_count"`
}
