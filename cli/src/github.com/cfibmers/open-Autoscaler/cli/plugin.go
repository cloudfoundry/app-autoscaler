package main

import (
	"github.com/cloudfoundry/cli/plugin"
	"github.com/codegangsta/cli"

	"github.com/cfibmers/open-Autoscaler/cli/context"
	"github.com/cfibmers/open-Autoscaler/cli/registry"
)

type PluginAutoScaler struct {
}

var _COMMAND_HELP = "NAME:" + `
   {{.Name}} - {{.Description}}{{with .ShortName}}

` + "ALIAS:" + `
   {{.}}{{end}}

` + "USAGE:" + `
   {{.Usage}}
{{with .Flags}}

` + "OPTIONS:" + `
{{range .}}   {{.}}
{{end}}{{end}}
`

func main() {
	plugin.Start(new(PluginAutoScaler))
}

func (p *PluginAutoScaler) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}
	context.Init(cliConnection)
	context.CheckPrerequisties()

	app := cli.NewApp()
	app.Name = "AutoScaler"
	app.Usage = "Manage auto-scaling service"
	app.Version = "0.0.1"
	app.Commands = registry.GetCliCommands()
	cli.CommandHelpTemplate = _COMMAND_HELP
	app.Run(append([]string{"auto-AutoScaler"}, args...))
}

func (p *PluginAutoScaler) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "AutoScaler",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 0,
			Build: 1,
		},
		Commands: registry.GetPluginCommandsMetadata(),
	}
}
