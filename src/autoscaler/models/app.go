package models

import "time"

type AppInfo struct {
	Entity AppEntity `json:"entity"`
}

type AppEntity struct {
	Instances int `json:"instances"`
}

type ScalingType int
type ScalingStatus int

const (
	ScalingTypeDynamic ScalingType = iota
	ScalingTypeSchedule
)

const (
	ScalingStatusSucceeded ScalingStatus = iota
	ScalingStatusFailed
	ScalingStatusIgnored
)

type AppScalingHistory struct {
	AppId        string        `json:"app_guid"`
	Timestamp    int64         `json:"timestamp"`
	ScalingType  ScalingType   `json:"scaling_type"`
	Status       ScalingStatus `json:"status"`
	OldInstances int           `json:"old_instances"`
	NewInstances int           `json:"new_instances"`
	Reason       string        `json:"reason"`
	Message      string        `json:"message"`
	Error        string        `json:"error"`
}

type AppMonitor struct {
	AppId      string
	MetricType string
	StatWindow time.Duration
}

type AppMetric struct {
	AppId      string
	MetricType string
	Value      string
	Unit       string
	Timestamp  int64
}
