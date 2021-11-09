package cred_helper

import (
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
)

type Credentials interface {
	Create(appId string, userProvidedCredential *models.Credential) (*models.Credential, error)
	Delete(appId string) error
	Get(appId string) (*models.Credential, error)
	InitializeConfig(dbConfigs map[db.Name]db.DatabaseConfig, loggingConfig helpers.LoggingConfig) error
}

type CreateArgs struct {
	AppId                  string
	UserProvidedCredential *models.Credential
}

type InitializeConfigArgs struct {
	DbConfigs     map[db.Name]db.DatabaseConfig
	LoggingConfig helpers.LoggingConfig
}
