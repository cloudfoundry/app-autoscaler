package model

import (
	"encoding/json"
	"time"
)

type GetPolicies func() map[string]*Policy

type PolicyJson struct {
	AppId     string `json:"appId"`
	PolicyStr string `json:"PolicyStr"`
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
func (p *PolicyJson) GetPolicy() *Policy {
	var triggerRecord TriggerRecord
	json.Unmarshal([]byte(p.PolicyStr), &triggerRecord)
	return &Policy{AppId: p.AppId, TriggerRecord: &triggerRecord}
}

type TriggerRecord struct {
	InstanceMaxCount int            `json:"instance_max_count"`
	InstanceMinCount int            `json:"instance_min_count"`
	ScalingRules     []*ScalingRule `json:"scaling_rules"`
}
type ScalingRule struct {
	MetricType            string `json:"metric_type"`
	StatWindowSeconds     int    `json:"stat_window_secs"`
	BreachDurationSeconds int    `json:"breach_duration_secs"`
	CoolDownSeconds       int    `json:"cool_down_secs"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
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

type Policy struct {
	AppId         string         `json:"appId"`
	TriggerRecord *TriggerRecord `json:"triggerRecord"`
}
