package cred_helper

import (
	"io"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	MaxRetry = 5
)

type Credentials interface {
	healthendpoint.Pinger
	io.Closer
	Create(appId string, userProvidedCredential *models.Credential) (*models.Credential, error)
	Delete(appId string) error
	Validate(appId string, credential models.Credential) (bool, error)
}
