package sbss_cred_helper

import (
	"autoscaler/db"
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
)

const (
	MaxRetry = 5
)

func CreateCredential(appId string, sbssDB db.SbssDB, maxRetry int, logger lager.Logger) (*models.Credential, error) {
	logger = logger.Session("create-sbss-credential", lager.Data{"appId": appId})
	var err error
	var count int
	var cred *models.Credential
	options := models.CredentialsOptions{BindingId: appId, InstanceId: appId}
	for {
		if count == maxRetry {
			return nil, err
		}
		cred, err = sbssDB.CreateCredentials(options)
		if err == nil {
			return cred, nil
		}
		logger.Error("sbss-create-credentials-call-failed", err, lager.Data{"try": count})
		// try to clean up the credentials if they were already there
		if cleanUpErr := sbssDB.DeleteCredentials(options); cleanUpErr != nil {
			logger.Error("sbss-delete-credentials-cleanup-call-failed", cleanUpErr, lager.Data{"try": count})
		}
		count++
	}

}

func DeleteCredential(appId string, sbssDB db.SbssDB, maxRetry int, logger lager.Logger) error {
	logger = logger.Session("delete-sbss-credential", lager.Data{"appId": appId})
	var err error
	var count int
	options := models.CredentialsOptions{BindingId: appId, InstanceId: appId}
	for {
		if count == maxRetry {
			return err
		}
		err = sbssDB.DeleteCredentials(options)
		if err == nil {
			return nil
		}
		logger.Error("sbss-delete-credentials-call-failed", err, lager.Data{"try": count})
		count++
	}

}
