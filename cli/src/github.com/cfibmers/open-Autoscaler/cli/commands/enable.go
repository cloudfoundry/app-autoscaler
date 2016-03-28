package commands

import (
	"fmt"
	"github.com/cfibmers/open-Autoscaler/cli/models"
	"github.com/codegangsta/cli"
)

type EnablePolicyCommand struct {
}

func NewEnablePolicyCommand() *EnablePolicyCommand {
	return &EnablePolicyCommand{}
}

func (cmd *EnablePolicyCommand) Metadata() models.CommandMetadata {
	return models.CommandMetadata{
		Name:        "as-enable-policy",
		Alias:       "aseap",
		Description: "Enable the enforcement of the auto-scaling policy",
		Usage:       "cf as-enable-policy APP_NAME",
	}
}

func (cmd *EnablePolicyCommand) Run(c *cli.Context) {
	fmt.Println("Run as-enable-policy command")

}
