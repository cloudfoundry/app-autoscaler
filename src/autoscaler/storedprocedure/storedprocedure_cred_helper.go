package storedprocedure

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager"
)

type Credentials struct {
	storedProcedureDb db.StoredProcedureDB
	maxRetry          int
	logger            lager.Logger
}

var _ cred_helper.Credentials = &Credentials{}

func NewWithStoredProcedureDb(storedProcedureDb db.StoredProcedureDB, maxRetry int, logger lager.Logger) cred_helper.Credentials {
	return &Credentials{
		storedProcedureDb: storedProcedureDb,
		maxRetry:          maxRetry,
		logger:            logger,
	}
}

func (c *Credentials) Create(appId string, _ *models.Credential) (*models.Credential, error) {
	logger := c.logger.Session("create-stored-procedure-credential", lager.Data{"appId": appId})
	var err error
	var count int
	var cred *models.Credential
	options := models.CredentialsOptions{BindingId: appId, InstanceId: appId}
	for {
		if count == c.maxRetry {
			return nil, err
		}
		cred, err = c.storedProcedureDb.CreateCredentials(options)
		if err == nil {
			return cred, nil
		}
		logger.Error("stored-procedure-create-credentials-call-failed", err, lager.Data{"try": count})
		// try to clean up the credentials if they were already there
		if cleanUpErr := c.storedProcedureDb.DeleteCredentials(options); cleanUpErr != nil {
			logger.Error("stored-procedure-delete-credentials-cleanup-call-failed", cleanUpErr, lager.Data{"try": count})
		}
		count++
	}
}

func (c *Credentials) Delete(appId string) error {
	logger := c.logger.Session("delete-stored-procedure-credential", lager.Data{"appId": appId})
	var err error
	var count int
	options := models.CredentialsOptions{BindingId: appId, InstanceId: appId}
	for {
		if count == c.maxRetry {
			return err
		}
		err = c.storedProcedureDb.DeleteCredentials(options)
		if err == nil {
			return nil
		}
		logger.Error("stored-procedure-delete-credentials-call-failed", err, lager.Data{"try": count})
		count++
	}
}

func (c *Credentials) Get(appId string) (*models.Credential, error) {
	return nil, nil
}

func (c *Credentials) InitializeConfig(_ map[db.Name]db.DatabaseConfig, _ helpers.LoggingConfig) error {
	panic("Not implemented")
}
