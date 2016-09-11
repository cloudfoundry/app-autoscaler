package models

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
