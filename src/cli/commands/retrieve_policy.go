package commands

import (
	"cli/api"
	"cli/ui"
	"errors"
	"os"
)

type PolicyCommand struct {
	RequiredlArgs PolicyPositionalArgs `positional-args:"yes"`
	Output        string               `long:"output" description:"dump the policy to a file in JSON format"`
}

type PolicyPositionalArgs struct {
	AppName string `positional-arg-name:"APP_NAME" required:"true" description:"The autoscaler API endpoint"`
}

func (command PolicyCommand) Execute([]string) error {
	return RetrievePolicy(AutoScaler.CLIConnection, command.RequiredlArgs.AppName)
}

func RetrievePolicy(cliConnection api.Connection, appName string) error {

	cfclient, err := api.NewCFClient(cliConnection)
	if err != nil {
		return err
	}
	err = cfclient.SetApp(appName)
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
	policy, err := apihelper.GetPolicy()
	if err != nil {
		return err
	}

	ui.SayMessage(policy)
	return nil
}
