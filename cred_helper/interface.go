package cred_helper

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type Credentials interface {
	Create(appId string, userProvidedCredential *models.Credential) (*models.Credential, error)
	Delete(appId string) error
	Validate(appId string, credential models.Credential) error
}
