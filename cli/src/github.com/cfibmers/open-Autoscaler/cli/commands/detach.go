package commands

import (
	"fmt"
	"github.com/cfibmers/open-Autoscaler/cli/models"
	"github.com/codegangsta/cli"
)

type DetachPolicyCommand struct {
}

func NewDetachPolicyCommand() *DetachPolicyCommand {
	return &DetachPolicyCommand{}
}

func (cmd *DetachPolicyCommand) Metadata() models.CommandMetadata {
	return models.CommandMetadata{
		Name:        "as-detach-policy",
		Alias:       "asdp",
		Description: "Detach and delete the auto-scaling policy from the application",
		Usage:       "cf as-detach-policy APP_NAME",
	}
}

func (cmd *DetachPolicyCommand) Run(c *cli.Context) {
	fmt.Println("Run as-detach-policy command")

}
