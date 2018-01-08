package commands

import (
	"cli/api"
	"cli/ui"
	"errors"
	"fmt"
	"io"
	"os"
)

type PolicyCommand struct {
	RequiredlArgs PolicyPositionalArgs `positional-args:"yes"`
	Output        string               `long:"output" description:"dump the policy to a file in JSON format"`
}

type PolicyPositionalArgs struct {
	AppName string `positional-arg-name:"APP_NAME" required:"true"`
}

func (command PolicyCommand) Execute([]string) error {

	var (
		err    error
		writer *os.File
	)

	if command.Output != "" {
		writer, err = os.OpenFile(command.Output, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	return RetrievePolicy(AutoScaler.CLIConnection, command.RequiredlArgs.AppName, writer)
}

func RetrievePolicy(cliConnection api.Connection, appName string, writer io.Writer) error {

	cfclient, err := api.NewCFClient(cliConnection)
	if err != nil {
		return err
	}
	err = cfclient.Configure(appName)
	if err != nil {
		return err
	}

	endpoint, err := api.GetEndpoint()
	if err != nil {
		return err
	}
	if endpoint.URL == "" {
		return errors.New(ui.NoEndpoint)
	}

	apihelper := api.NewAPIHelper(endpoint, cfclient, os.Getenv("CF_TRACE"))

	ui.SayMessage(ui.ShowPolicyHint, appName)

	policy, err := apihelper.GetPolicy()
	if err != nil {
		return err
	}

	ui.SayOK()
	fmt.Fprintf(writer, string(policy))
	return nil
}
