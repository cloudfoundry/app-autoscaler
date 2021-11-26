package cred_helper

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	MaxRetry = 5
)

type Credentials interface {
	Create(appId string, userProvidedCredential *models.Credential) (*models.Credential, error)
	Delete(appId string) error
	Validate(appId string, credential models.Credential) (bool, error)
}
