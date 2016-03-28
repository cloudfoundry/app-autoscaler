package registry

import (
	as_commands "github.com/cfibmers/open-Autoscaler/cli/commands"
	"github.com/cfibmers/open-Autoscaler/cli/models"
	"github.com/cloudfoundry/cli/plugin"
	"github.com/codegangsta/cli"
)

var commands = []models.Command{}
var actions = []func(context *cli.Context){}

func RegisterCommand(cmd models.Command) {
	commands = append(commands, cmd)
	actions = append(actions, func(context *cli.Context) {
		cmd.Run(context)
	})
}

func GetPluginCommandsMetadata() []plugin.Command {
	pluginCommands := []plugin.Command{}
	for _, c := range commands {
		m := c.Metadata()

		options := make(map[string]string)
		for _, flag := range m.Flags {
			switch flag.(type) {
			case cli.StringFlag:
				options[flag.(cli.StringFlag).Name] = flag.(cli.StringFlag).Usage
			case cli.BoolFlag:
				options[flag.(cli.BoolFlag).Name] = flag.(cli.BoolFlag).Usage
			case cli.IntFlag:
				options[flag.(cli.IntFlag).Name] = flag.(cli.IntFlag).Usage
			}
		}

		cmd := plugin.Command{
			Name:     m.Name,
			Alias:    m.Alias,
			HelpText: m.Description,
			UsageDetails: plugin.Usage{
				Usage:   m.Usage,
				Options: options,
			},
		}
		pluginCommands = append(pluginCommands, cmd)
	}
	return pluginCommands
}

func GetCliCommands() []cli.Command {
	clis := []cli.Command{}
	for index, cmd := range commands {
		m := cmd.Metadata()
		c := cli.Command{
			Name:        m.Name,
			Description: m.Description,
			Usage:       m.Usage,
			Flags:       m.Flags,
			Action:      actions[index],
		}
		clis = append(clis, c)
	}
	return clis
}

func loadCommands() interface{} {
	RegisterCommand(as_commands.NewPolicyCommand())
	RegisterCommand(as_commands.NewAttachPolicyCommand())
	RegisterCommand(as_commands.NewDetachPolicyCommand())
	RegisterCommand(as_commands.NewEnablePolicyCommand())
	RegisterCommand(as_commands.NewDisablePolicyCommand())
	RegisterCommand(as_commands.NewHistoryCommand())
	return nil
}

var _ = loadCommands()
