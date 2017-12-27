package commands

import (
	"cli/api"
	"cli/ui"
	"errors"
	"os"
)

type DetachPolicyCommand struct {
	RequiredlArgs DetachPolicyPositionalArgs `positional-args:"yes"`
}

type DetachPolicyPositionalArgs struct {
	AppName string `positional-arg-name:"APP_NAME" required:"true" `
}

func (command DetachPolicyCommand) Execute([]string) error {
	return DetachPolicy(AutoScaler.CLIConnection, command.RequiredlArgs.AppName)
}

func DetachPolicy(cliConnection api.Connection, appName string) error {

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

	ui.SayMessage(ui.DetachPolicyHint, appName)
	err = apihelper.DeletePolicy()
	if err != nil {
		return err
	}

	ui.SayOK()
	return nil
}
