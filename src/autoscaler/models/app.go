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
	AppId        string
	Timestamp    int64
	ScalingType  ScalingType
	Status       ScalingStatus
	OldInstances int
	NewInstances int
	Reason       string
	Message      string
	Error        string
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
