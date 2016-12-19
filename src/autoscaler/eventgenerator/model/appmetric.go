package model

import (
	"time"
)

type AppMonitor struct {
	AppId      string
	MetricType string
	StatWindow time.Duration
}
type AppMetric struct {
	AppId      string
	MetricType string
	Value      *int64
	Unit       string
	Timestamp  int64
}
type Trigger struct {
	AppId                 string `json:"appId"`
	MetricType            string `json:"metric_type"`
	BreachDurationSeconds int    `json:"breach_duration_secs"`
	CoolDownSeconds       int    `json:"cool_down_secs"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	Adjustment            string `json:"adjustment"`
}

func (t Trigger) BreachDuration() time.Duration {
	return time.Duration(t.BreachDurationSeconds) * time.Second
}

func (t Trigger) CoolDown() time.Duration {
	return time.Duration(t.CoolDownSeconds) * time.Second
}
