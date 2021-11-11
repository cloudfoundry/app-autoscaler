package cred_helper

import (
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"net/rpc"
)

type CredentialsRPCClient struct {
	client *rpc.Client
}

func (g *CredentialsRPCClient) Create(appId string, userProvidedCredentials *models.Credential) (*models.Credential, error) {
	var reply = models.Credential{}
	err := g.client.Call("Plugin.Create", CreateArgs{AppId: appId, UserProvidedCredential: userProvidedCredentials}, &reply)
	if err != nil {
		return nil, err
	}

	return &reply, nil
}

func (g *CredentialsRPCClient) Delete(appId string) error {
	var reply interface{}
	err := g.client.Call("Plugin.Delete", appId, &reply)
	if err != nil {
		return err
	}

	return nil
}

func (g *CredentialsRPCClient) Get(appId string) (*models.Credential, error) {
	var reply = models.Credential{}
	err := g.client.Call("Plugin.Get", appId, &reply)
	if err != nil {
		return nil, err
	}

	return &reply, nil
}

func (g *CredentialsRPCClient) InitializeConfig(dbConfigs map[db.Name]db.DatabaseConfig, loggingConfig helpers.LoggingConfig) error {
	var reply interface{}
	err := g.client.Call("Plugin.InitializeConfig", InitializeConfigArgs{DbConfigs: dbConfigs,
		LoggingConfig: loggingConfig}, &reply)
	if err != nil {
		return err
	}

	return nil
}

// Golang standard: check if the interface implements the methods
var _ Credentials = &CredentialsRPCClient{}

type CredentialsRPCServer struct {
	Impl Credentials
}

func (s *CredentialsRPCServer) Create(args CreateArgs, reply *models.Credential) error {
	r, err := s.Impl.Create(args.AppId, args.UserProvidedCredential)
	if err != nil {
		return err
	}
	if r != nil {
		reply.Username = r.Username
		reply.Password = r.Password
	}
	return nil
}

func (s *CredentialsRPCServer) Delete(appId string, _ *interface{}) error {
	return s.Impl.Delete(appId)
}

func (s *CredentialsRPCServer) Get(appId string, reply *models.Credential) error {
	r, err := s.Impl.Get(appId)
	if err != nil {
		return err
	}
	if r != nil {
		reply.Username = r.Username
		reply.Password = r.Password
	}
	return nil
}

func (s *CredentialsRPCServer) InitializeConfig(args InitializeConfigArgs, _ *interface{}) error {
	return s.Impl.InitializeConfig(args.DbConfigs, args.LoggingConfig)
}
