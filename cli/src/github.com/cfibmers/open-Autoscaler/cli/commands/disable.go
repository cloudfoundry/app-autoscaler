package commands

import (
	"fmt"
	"github.com/cfibmers/open-Autoscaler/cli/models"
	"github.com/codegangsta/cli"
)

type DisablePolicyCommand struct {
}

func NewDisablePolicyCommand() *DisablePolicyCommand {
	return &DisablePolicyCommand{}
}

func (cmd *DisablePolicyCommand) Metadata() models.CommandMetadata {
	return models.CommandMetadata{
		Name:        "as-disable-policy",
		Alias:       "asdap",
		Description: "Suspend the auto-scaling policy temporarily",
		Usage:       "cf as-disable-policy APP_NAME",
	}
}

func (cmd *DisablePolicyCommand) Run(c *cli.Context) {
	fmt.Println("Run as-disable-policy command")

}
