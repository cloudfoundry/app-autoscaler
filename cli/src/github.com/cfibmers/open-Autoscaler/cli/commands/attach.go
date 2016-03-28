package commands

import (
	"fmt"
	"github.com/cfibmers/open-Autoscaler/cli/models"
	"github.com/codegangsta/cli"
)

type AttachPolicyCommand struct {
}

func NewAttachPolicyCommand() *AttachPolicyCommand {
	return &AttachPolicyCommand{}
}

func (cmd *AttachPolicyCommand) Metadata() models.CommandMetadata {
	return models.CommandMetadata{
		Name:        "as-attach-policy",
		Alias:       "asap",
		Description: "Attach an auto-scaling policy to the application",
		Usage:       "cf as-attach-policy APP_NAME -p POLICY_FILE",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "p",
				Usage: "the name of the policy file in JSON, if 'policy.json' in the current directory is not used",
			},
		},
	}
}

func (cmd *AttachPolicyCommand) Run(c *cli.Context) {
	fmt.Println("Run as-attach-policy command")

}
