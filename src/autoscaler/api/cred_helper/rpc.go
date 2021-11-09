package cred_helper

import (
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"net/rpc"
)

type CredentialsRPC struct {
	client *rpc.Client
}

func (g *CredentialsRPC) Create(appId string, userProvidedCredential *models.Credential) (*models.Credential, error) {
	var resp string
	err := g.client.Call("Plugin.Create", new(interface{}), &resp)
	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return nil, nil
}

func (g *CredentialsRPC) Delete(appId string) error {
	var resp string
	err := g.client.Call("Plugin.Delete", new(interface{}), &resp)
	if err != nil {
		return err
	}

	return nil
}

func (g *CredentialsRPC) Get(appId string) (*models.Credential, error) {
	var resp string
	err := g.client.Call("Plugin.Get", new(interface{}), &resp)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (g *CredentialsRPC) InitializeConfig(dbConfig map[string]db.DatabaseConfig, loggingConfig helpers.LoggingConfig) error {
	var resp string
	err := g.client.Call("Plugin.InitializeConfig", new(interface{}), &resp)
	if err != nil {
		return err
	}

	return nil
}

var _ Credentials = &CredentialsRPC{}

type CredentialsRPCServer struct {
	Impl Credentials
}

func (s *CredentialsRPCServer) Create(appId string, userProvidedCredential *models.Credential) (*models.Credential, error) {
	return s.Impl.Create(appId, userProvidedCredential)
}

func (s *CredentialsRPCServer) Delete(appId string) error {
	return s.Impl.Delete(appId)
}

func (s *CredentialsRPCServer) Get(appId string) (*models.Credential, error) {
	return s.Impl.Get(appId)
}

func (s *CredentialsRPCServer) InitializeConfig(dbConfig map[string]db.DatabaseConfig, loggingConfig helpers.LoggingConfig) error {
	return s.Impl.InitializeConfig(dbConfig, loggingConfig)
}

var _ Credentials = &CredentialsRPCServer{}
