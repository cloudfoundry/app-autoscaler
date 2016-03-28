package commands

import (
	"fmt"
	"github.com/cfibmers/open-Autoscaler/cli/models"
	"github.com/codegangsta/cli"
)

type HistoryCommand struct {
}

func NewHistoryCommand() *HistoryCommand {
	return &HistoryCommand{}
}

func (cmd *HistoryCommand) Metadata() models.CommandMetadata {
	return models.CommandMetadata{
		Name:        "as-history",
		Alias:       "ash",
		Description: "Query the auto-caling history",
		Usage:       "cf as-history APP_NAME --start-time START --end-time END",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "start-time",
				Usage: "the start time of the query, in format 'yyyy-MM-ddTHH:mm:ss+/-hhmm' or 'yyyy-MM-ddTHH:mm:ssZ'",
			},
			cli.StringFlag{
				Name:  "end-time",
				Usage: "the end time of the query, in format 'yyyy-MM-ddTHH:mm:ss+/-hhmm' or 'yyyy-MM-ddTHH:mm:ssZ'",
			},
		},
	}
}

func (cmd *HistoryCommand) Run(c *cli.Context) {
	fmt.Println("Run as-history command")

}
