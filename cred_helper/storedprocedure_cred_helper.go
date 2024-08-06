package cred_helper

import (
	"context"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3"
)

type storedProcedureCredentials struct {
	storedProcedureDb db.StoredProcedureDB
	bindingDB		  db.BindingDB
	maxRetry          int
	logger            lager.Logger
}

func (c *storedProcedureCredentials) Ping() error {
	return c.storedProcedureDb.Ping()
}

func (c *storedProcedureCredentials) Close() error {
	return c.storedProcedureDb.Close()
}

var _ Credentials = &storedProcedureCredentials{}

func NewStoredProcedureCredHelper(storedProcedureDb db.StoredProcedureDB, bindingDB db.BindingDB, maxRetry int, logger lager.Logger) Credentials {
	return &storedProcedureCredentials{
		storedProcedureDb: storedProcedureDb,
		bindingDB:         bindingDB,
		maxRetry:          maxRetry,
		logger:            logger,
	}
}

func (c *storedProcedureCredentials) Create(ctx context.Context, appId string, userProvidedCredential *models.Credential) (*models.Credential, error) {
	logger := c.logger.Session("create-stored-procedure-credential", lager.Data{"appId": appId})
	var err error
	var count int
	var cred *models.Credential
	options := models.CredentialsOptions{BindingId: appId, InstanceId: appId}
	for {
		if count == c.maxRetry {
			return nil, err
		}
		cred, err = c.storedProcedureDb.CreateCredentials(ctx, options)
		if err == nil {
			return cred, nil
		}
		logger.Error("stored-procedure-create-credentials-call-failed", err, lager.Data{"try": count})
		// try to clean up the credentials if they were already there
		if cleanUpErr := c.storedProcedureDb.DeleteCredentials(ctx, options); cleanUpErr != nil {
			logger.Error("stored-procedure-delete-credentials-cleanup-call-failed", cleanUpErr, lager.Data{"try": count})
		}
		count++
	}
}

func (c *storedProcedureCredentials) Delete(ctx context.Context, appId string) error {
	logger := c.logger.Session("delete-stored-procedure-credential", lager.Data{"appId": appId})
	var err error
	var count int
	options := models.CredentialsOptions{BindingId: appId, InstanceId: appId}
	for {
		if count == c.maxRetry {
			return err
		}
		err = c.storedProcedureDb.DeleteCredentials(ctx, options)
		if err == nil {
			return nil
		}
		logger.Error("stored-procedure-delete-credentials-call-failed", err, lager.Data{"try": count})
		count++
	}
}

func (c *storedProcedureCredentials) Validate(ctx context.Context, appId string, credential models.Credential) (bool, error) {
	bindingId, err := c.bindingDB.GetBindingIdByAppId(ctx, appId)
	if err != nil {
		return false, err
	}
	_, err = c.storedProcedureDb.ValidateCredentials(ctx, credential, bindingId)
	if err != nil {
		return false, err
	}
	return true, nil
}
