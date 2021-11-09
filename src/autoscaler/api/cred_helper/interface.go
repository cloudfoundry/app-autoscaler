package cred_helper

import "autoscaler/models"

type Credentials interface {
	Create(appId string, userProvidedCredential *models.Credential) (*models.Credential, error)
	Delete(appId string) error
}
