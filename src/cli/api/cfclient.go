package api

import (
	"cli/ui"
	"errors"
	"fmt"

	"code.cloudfoundry.org/cli/plugin/models"
)

type CFClient struct {
	connection    Connection
	CCAPIEndpoint string
	AuthToken     string
	AppId         string
	AppName       string
	IsSSLDisabled bool
}

type Connection interface {
	ApiEndpoint() (string, error)
	IsLoggedIn() (bool, error)
	AccessToken() (string, error)
	GetApp(string) (plugin_models.GetAppModel, error)
	IsSSLDisabled() (bool, error)
}

func NewCFClient(connection Connection) (*CFClient, error) {

	ccAPIEndpoint, err := connection.ApiEndpoint()
	if err != nil {
		return nil, err
	}

	isSSLDisabled, err := connection.IsSSLDisabled()
	if err != nil {
		return nil, err
	}

	client := &CFClient{
		connection:    connection,
		CCAPIEndpoint: ccAPIEndpoint,
		IsSSLDisabled: isSSLDisabled,
	}

	return client, nil

}

func (client *CFClient) Configure(appName string) error {

	if connected, err := client.connection.IsLoggedIn(); !connected {
		if err != nil {
			return err
		}
		return errors.New(fmt.Sprintf(ui.LoginRequired, client.CCAPIEndpoint))
	}

	authToken, err := client.connection.AccessToken()
	if err != nil {
		return err
	}

	app, err := client.connection.GetApp(appName)
	if err != nil {
		return err
	}
	client.AuthToken = authToken
	client.AppId = app.Guid
	client.AppName = appName
	return nil

}
