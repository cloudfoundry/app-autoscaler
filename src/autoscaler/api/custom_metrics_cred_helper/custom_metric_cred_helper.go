package custom_metrics_cred_helper

import (
	"database/sql"

	"autoscaler/db"
	"autoscaler/models"

	uuid "github.com/nu7hatch/gouuid"
)

const (
	MaxRetry = 5
)

func _createCustomMetricsCredential(appId string, policyDB db.PolicyDB) (*models.CustomMetricCredentials, error) {
	credUsername, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	credPassword, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	cred := models.CustomMetricCredentials{
		Username: credUsername.String(),
		Password: credPassword.String(),
	}
	err = policyDB.SaveCustomMetricsCred(appId, cred)
	if err != nil {
		return nil, err
	}
	return &cred, nil
}
func CreateCustomMetricsCredential(appId string, policyDB db.PolicyDB, maxRetry int) (*models.CustomMetricCredentials, error) {

	var err error
	var count int
	var cred *models.CustomMetricCredentials
	cred, err = GetCustomMetricsCredential(appId, policyDB, maxRetry)
	if err == sql.ErrNoRows {
		for {
			if count == maxRetry {
				return nil, err
			}
			cred, err = _createCustomMetricsCredential(appId, policyDB)
			if err == nil {
				return cred, nil
			}
			count++
		}
	}
	if err != nil {
		return nil, err
	}
	return cred, err

}

func _deleteCustomMetricsCredential(appId string, policyDB db.PolicyDB) error {
	err := policyDB.DeleteCustomMetricsCred(appId)
	if err != nil {
		return err
	}
	return nil
}
func DeleteCustomMetricsCredential(appId string, policyDB db.PolicyDB, maxRetry int) error {

	var err error
	var count int
	for {
		if count == maxRetry {
			return err
		}
		err = _deleteCustomMetricsCredential(appId, policyDB)
		if err == nil {
			return nil
		}
		count++
	}

}

func _getCustomMetricsCredential(appId string, policyDB db.PolicyDB) (*models.CustomMetricCredentials, error) {
	cred, err := policyDB.GetCustomMetricsCreds(appId)
	if err != nil {
		return nil, err
	}
	return cred, nil
}
func GetCustomMetricsCredential(appId string, policyDB db.PolicyDB, maxRetry int) (*models.CustomMetricCredentials, error) {
	var err error
	var count int
	var cred *models.CustomMetricCredentials
	for {
		if count == maxRetry {
			return nil, err
		}
		cred, err = _getCustomMetricsCredential(appId, policyDB)
		if err == nil {
			return cred, nil
		}
		if err == sql.ErrNoRows {
			return nil, err
		}
		count++
	}

}
