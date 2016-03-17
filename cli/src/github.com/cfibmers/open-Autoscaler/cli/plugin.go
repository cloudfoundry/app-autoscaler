package main

import (
	"fmt"
	"github.com/cloudfoundry/cli/plugin"
)

type PluginAutoScaler struct {
}


func (p *PluginAutoScaler) Run(cliConnection plugin.CliConnection, args []string) {
	fmt.Printf("Run autoscaler command: %s \n", args)
}

func (po *PluginAutoScaler) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "AutoScaler",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 0,
			Build: 1,
		},
		Commands: []plugin.Command{
			{
				Name: "as-policy",
				Alias: "asp",
				HelpText: "Show the auto-scaling policy of the application",
				UsageDetails: plugin.Usage{
					Usage: "cf as-policy APP_NAME [--json]", 
					Options: map[string]string{
						"json": "show the policy in json format",
					},
				},	
			},
			{
				Name: "as-attach-policy",
				Alias: "asap",
				HelpText: "Attach an auto-scaling policy to the application",
				UsageDetails: plugin.Usage{
					Usage: "cf as-attach-policy APP_NAME -p POLICY_FILE", 
					Options: map[string]string{
						"p": "the name of the policy file in JSON, if 'policy.json' in the current directory is not used",
					},
				},	
			},
			{
				Name: "as-detach-policy",
				Alias: "asdp",
				HelpText: "detach and delete the auto-scaling policy from the application",
				UsageDetails: plugin.Usage{
					Usage: "cf as-detach-policy APP_NAME", 
				},	
			},
			{
				Name: "as-enable-policy",
				Alias: "aseap",
				HelpText: "resume the enforcement of the auto-scaling policy",
				UsageDetails: plugin.Usage{
					Usage: "cf as-enable-policy APP_NAME", 
				},	
			},
			{
				Name: "as-disable-policy",
				Alias: "asdap",
				HelpText: "suspend the auto-scaling policy temporarily",
				UsageDetails: plugin.Usage{
					Usage: "cf as-disable-policy APP_NAME", 
				},	
			},
			{
				Name: "as-history",
				Alias: "ash",
				HelpText: "Query the auto-caling history",
				UsageDetails: plugin.Usage{
					Usage: "cf as-history APP_NAME --start-time START --end-time END --json", 
					Options: map[string]string{
						"--start-time": "the start time of the query, in format 'yyyy-MM-ddTHH:mm:ss+/-hhmm' or 'yyyy-MM-ddTHH:mm:ssZ'",
						"--end-time": "the end time of the query, in format 'yyyy-MM-ddTHH:mm:ss+/-hhmm' or 'yyyy-MM-ddTHH:mm:ssZ'",
						"--json": "show the history in JSON format",
					},
				},	
			},

		},
	}
}