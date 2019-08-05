package custom_metrics_cred_helper

import (
	"autoscaler/db"
	"autoscaler/models"

	uuid "github.com/nu7hatch/gouuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	MaxRetry = 5
)

func _createCustomMetricsCredential(appId string, policyDB db.PolicyDB) (*models.CustomMetricCredentials, error) {
	credUsername, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	userNameHash, err := bcrypt.GenerateFromPassword([]byte(credUsername.String()), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	credPassword, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(credPassword.String()), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	cred := models.CustomMetricCredentials{
		Username: credUsername.String(),
		Password: credPassword.String(),
	}

	err = policyDB.SaveCustomMetricsCred(appId, models.CustomMetricCredentials{
		Username: string(userNameHash),
		Password: string(passwordHash),
	})
	if err != nil {
		return nil, err
	}
	return &cred, nil
}
func CreateCustomMetricsCredential(appId string, policyDB db.PolicyDB, maxRetry int) (*models.CustomMetricCredentials, error) {

	var err error
	var count int
	var cred *models.CustomMetricCredentials
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
