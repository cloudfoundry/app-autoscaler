package policy

import (
	"encoding/json"
)

type Policy interface {
	Equals(p *PolicyJson) bool
	GetTrigger(p *PolicyJson) *Trigger
}
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
func (p *PolicyJson) GetTrigger() *Trigger {
	var triggerRecord TriggerRecord
	json.Unmarshal([]byte(p.PolicyStr), &triggerRecord)
	return &Trigger{AppId: p.AppId, TriggerRecord: &triggerRecord}
}

type TriggerRecord struct {
	InstanceMaxCount int            `json:"instance_max_count"`
	InstanceMinCount int            `json:"instance_min_count"`
	ScalingRules     []*ScalingRule `json:"scaling_rules"`
}
type ScalingRule struct {
	MetricType         string `json:"metric_type"`
	StatWindowSecs     int    `json:"stat_window_secs"`
	BreachDurationSecs int    `json:"breach_duration_secs"`
	CoolDownSecs       int    `json:"cool_down_secs"`
	Threshold          int    `json:"threshold"`
	Operator           string `json:"operator"`
	Adjustment         string `json:"adjustment"`
}
type Trigger struct {
	AppId         string         `json:"appId"`
	TriggerRecord *TriggerRecord `json:"triggerRecord"`
}
