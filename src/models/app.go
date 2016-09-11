package models

type AppInfo struct {
	Entity AppEntity `json:"entity"`
}

type AppEntity struct {
	Instances int `json:"instances"`
}

const (
	ScalingTypeDynamic     = 0
	ScalingTypeSchedule    = 1
	ScalingStatusSucceeded = 0
	ScalingStatusFailed    = 1
	ScalingStatusIgnored   = 2
)

type AppScalingHistory struct {
	AppId        string
	Timestamp    int64
	ScalingType  int
	Status       int
	OldInstances int
	NewInstances int
	Reason       string
	Message      string
	Error        string
}
