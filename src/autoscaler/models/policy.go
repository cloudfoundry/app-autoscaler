package models

import (
	"encoding/json"
	"time"
)

type GetPolicies func() map[string]*AppPolicy

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
	InstanceMin  int            `json:"instance_min_count"`
	InstanceMax  int            `json:"instance_max_count"`
	ScalingRules []*ScalingRule `json:"scaling_rules"`
}

type ScalingRule struct {
	MetricType            string `json:"metric_type"`
	StatWindowSeconds     int    `json:"stat_window_secs"`
	BreachDurationSeconds int    `json:"breach_duration_secs"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	CoolDownSeconds       int    `json:"cool_down_secs"`
	Adjustment            string `json:"adjustment"`
}

func (r *ScalingRule) StatWindow() time.Duration {
	return time.Duration(r.StatWindowSeconds) * time.Second
}

func (r *ScalingRule) BreachDuration() time.Duration {
	return time.Duration(r.BreachDurationSeconds) * time.Second
}

func (r *ScalingRule) CoolDown() time.Duration {
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

func (t Trigger) CoolDown() time.Duration {
	return time.Duration(t.CoolDownSeconds) * time.Second
}

type ActiveSchedule struct {
	ScheduleId         string
	InstanceMin        int `json:"instance_min_count"`
	InstanceMax        int `json:"instance_max_count"`
	InstanceMinInitial int `json:"initial_min_instance_count"`
}
