package commands

import (
	"fmt"
	"github.com/cfibmers/open-Autoscaler/cli/models"
	"github.com/codegangsta/cli"
)

type PolicyCommand struct {
}

func NewPolicyCommand() *PolicyCommand {
	return &PolicyCommand{}
}

func (cmd *PolicyCommand) Metadata() models.CommandMetadata {
	return models.CommandMetadata{
		Name:        "as-policy",
		Alias:       "asp",
		Description: "Show the auto-scaling policy of the application",
		Usage:       "cf as-policy APP_NAME [--json]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "json",
				Usage: "show the policy in json format",
			},
		},
	}
}

func (cmd *PolicyCommand) Run(c *cli.Context) {
	fmt.Println("Run as-policy command")

}
