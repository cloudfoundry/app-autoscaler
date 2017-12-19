package commands

import (
	"cli/api"
)

type AutoScalerCmds struct {
	CLIConnection api.Connection

	API    ApiCommand    `command:"autoscaling-api" description:"Set or view AutoScaler service API endpoint"`
	Policy PolicyCommand `command:"autoscaling-policy" description:"Retrieve the scaling policy of an application"`

	UninstallPlugin UninstallHook `command:"CLI-MESSAGE-UNINSTALL"`
}

var AutoScaler AutoScalerCmds
