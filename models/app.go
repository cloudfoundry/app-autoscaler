package models

import "time"

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
	ScalingStatusPending
	ScalingStatusUnknown
	ScalingStatusSuperseded
	ScalingStatusAborted
)

const (
	AppStatusStopped = "STOPPED"
	AppStatusStarted = "STARTED"
)

type AppScalingHistory struct {
	ScalingDimension ScalingDimension `json:"scaling_dimension,omitempty"`
	CorrelationID    string           `json:"correlation_id,omitempty"`
	DeploymentGUID   string           `json:"deployment_guid,omitempty"`
	OldMemoryMB      int              `json:"old_memory_mb,omitempty"`
	NewMemoryMB      int              `json:"new_memory_mb,omitempty"`
	CoolDownSeconds  int              `json:"cool_down_secs,omitempty"`
	CompletedAt      int64            `json:"completed_at,omitempty"`
	AppId            string           `json:"app_id"`
	Timestamp        int64            `json:"timestamp"`
	ScalingType      ScalingType      `json:"scaling_type"`
	Status           ScalingStatus    `json:"status"`
	OldInstances     int              `json:"old_instances"`
	NewInstances     int              `json:"new_instances"`
	Reason           string           `json:"reason"`
	Message          string           `json:"message"`
	Error            string           `json:"error"`
}

type AppMonitor struct {
	AppId      string
	MetricType string
	StatWindow time.Duration
}

type AppScalingResult struct {
	Reason            string           `json:"reason,omitempty"`
	ScalingDimension  ScalingDimension `json:"scaling_dimension,omitempty"`
	DeploymentGUID    string           `json:"deployment_guid,omitempty"`
	AppId             string           `json:"app_id"`
	Status            ScalingStatus    `json:"status"`
	Adjustment        int              `json:"adjustment"`
	CooldownExpiredAt int64            `json:"cool_down_expired_at"`
}
