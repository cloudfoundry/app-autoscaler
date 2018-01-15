package commands

import (
	"cli/api"
)

type AutoScalerCmds struct {
	CLIConnection api.Connection

	API          ApiCommand          `command:"autoscaling-api" description:"Set or view AutoScaler service API endpoint"`
	Policy       PolicyCommand       `command:"autoscaling-policy" description:"Retrieve the scaling policy of an application"`
	AttachPolicy AttachPolicyCommand `command:"attach-autoscaling-policy" description:"Attach a scaling policy to an application"`
	DetachPolicy DetachPolicyCommand `command:"detach-autoscaling-policy" description:"Detach a scaling policy from an application"`
	Metrics      MetricsCommand      `command:"autoscaling-metrics" description:"Retrieve the metrics of an application"`

	UninstallPlugin UninstallHook `command:"CLI-MESSAGE-UNINSTALL"`
}

var AutoScaler AutoScalerCmds
