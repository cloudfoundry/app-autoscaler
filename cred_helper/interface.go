package cred_helper

import (
	"context"
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
	Create(ctx context.Context, appId string, userProvidedCredential *models.Credential) (*models.Credential, error)
	Delete(ctx context.Context, appId string) error
	Validate(ctx context.Context, appId string, credential models.Credential) (bool, error)
}
