/**
*
* The command line tool for CF open-autoscaler, built as a Cloudfoundry CLI plugin
*
**/
package main

import (
	"fmt"
	"github.com/cloudfoundry/cli/plugin"
)

type PluginAutoScaler struct {
}

func main() {

	plugin.Start(new(PluginAutoScaler))
}

func (p *PluginAutoScaler) Run(cliConnection plugin.CliConnection, args []string) {
	fmt.Println("Run autoscaler command")
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
			},
		},
	}
}