package main

import (
	"os"

	"cli/commands"
	"cli/ui"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/jessevdk/go-flags"
)

type AutoScaler struct{}

func (as *AutoScaler) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "AutoScaler",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 1,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "autoscaling-api",
				Alias:    "asa",
				HelpText: "Set or view AutoScaler service API endpoint",
				UsageDetails: plugin.Usage{
					Usage: `cf autoscaling-api [URL] [--unset] [--skip-ssl-validation]

OPTIONS:
	--unset                 Unset the api endpoint,
	--skip-ssl-validation   Skip verification of the api endpoint. Not recommended! Inherit "cf" --skip-ssl-validation setting by default`,
				},
			},
			{
				Name:     "autoscaling-policy",
				Alias:    "asp",
				HelpText: "Retrieve the scaling policy of an application",
				UsageDetails: plugin.Usage{
					Usage: `cf autoscaling-policy APP_NAME [--output PATH_TO_FILE]

OPTIONS:
	--output	Dump the policy to a file in JSON format`,
				},
			},
			{
				Name:     "attach-autoscaling-policy",
				Alias:    "aasp",
				HelpText: "Attach a scaling policy to an application",
				UsageDetails: plugin.Usage{
					Usage: `cf attach-autoscaling-policy APP_NAME PATH_TO_FILE`,
				},
			},
			{
				Name:     "detach-autoscaling-policy",
				Alias:    "dasp",
				HelpText: "Detach the scaling policy from an application",
				UsageDetails: plugin.Usage{
					Usage: `cf detach-as-policy APP_NAME`,
				},
			},
			{
				Name:     "autoscaling-metrics",
				Alias:    "asm",
				HelpText: "Retrieve the metrics of an application",
				UsageDetails: plugin.Usage{
					Usage: `cf autoscaling-metrics APP_NAME METRIC_NAME [--start START_TIME] [--end END_TIME] [--desc] [--output PATH_TO_FILE]

METRIC_NAME: 
	memoryused, memoryutil, responsetime, throughput.	

OPTIONS:
	--start		Start time of metrics collected with format "yyyy-MM-ddTHH:mm:ss+/-HH:mm" or "yyyy-MM-ddTHH:mm:ssZ", default to very beginning if not specified.
	--end		End time of the metrics collected with format "yyyy-MM-ddTHH:mm:ss+/-HH:mm" or "yyyy-MM-ddTHH:mm:ssZ", default to current time if not speficied.
	--desc		Display in descending order, default to ascending order if not specified.
	--output	Dump the metrics to a file in table format.
					`,
				},
			},
			{
				Name:     "autoscaling-history",
				Alias:    "ash",
				HelpText: "Retrieve the scaling history of an application",
				UsageDetails: plugin.Usage{
					Usage: `cf autoscaling-history APP_NAME [--start START_TIME] [--end END_TIME] [--desc] [--output PATH_TO_FILE]

OPTIONS:
	--start		Start time of the scaling history with format "yyyy-MM-ddTHH:mm:ss+/-HH:mm" or "yyyy-MM-ddTHH:mm:ssZ", default to very beginning if not specified.
	--end		End time of the scaling history with format "yyyy-MM-ddTHH:mm:ss+/-HH:mm" or "yyyy-MM-ddTHH:mm:ssZ", default to current time if not speficied.
	--desc		Display in descending order, default to ascending order if not specified.
	--output	Dump the scaling history to a file in table format.
					`,
				},
			},
		},
	}
}

func main() {
	plugin.Start(new(AutoScaler))
}

func (as *AutoScaler) Run(cliConnection plugin.CliConnection, args []string) {

	commands.AutoScaler.CLIConnection = cliConnection
	parser := flags.NewParser(&commands.AutoScaler, flags.HelpFlag|flags.PassDoubleDash)
	parser.NamespaceDelimiter = "-"

	_, err := parser.ParseArgs(args)
	if err != nil {
		ui.SayFailed()
		ui.SayMessage("Error: %s", err.Error())
		os.Exit(1)
	}

}
