package commands

import (
	"cli/api"
	"cli/ui"
	"errors"
	"io/ioutil"
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
	return RetrievePolicy(AutoScaler.CLIConnection, command.RequiredlArgs.AppName, command.Output)
}

func RetrievePolicy(cliConnection api.Connection, appName string, output string) error {

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

	if output != "" {
		err = ioutil.WriteFile(output, []byte(policy), 0666)
		if err != nil {
			return err
		}
		ui.SayOK()
	} else {
		ui.SayOK()
		ui.SayMessage(policy)
	}

	return nil
}
