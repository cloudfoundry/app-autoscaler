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
	AppId            string        `json:"appId"`
	MetricType       string        `json:"metricType"`
	BreachDuration   time.Duration `json:"breachDuration"`
	CoolDownDuration time.Duration `json:"coolDownDuration"`
	Threshold        int64         `json:"threshold"`
	Operator         string        `json:"operator"`
	Adjustment       string        `json:"adjustment"`
}
