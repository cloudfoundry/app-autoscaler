package models

import (
	"github.com/codegangsta/cli"
)

type CommandMetadata struct {
	Name        string
	Alias       string
	Description string
	Usage       string
	Flags       []cli.Flag
}

type Command interface {
	Metadata() CommandMetadata
	Run(*cli.Context)
}
