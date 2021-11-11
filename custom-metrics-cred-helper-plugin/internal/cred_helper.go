package internal

import (
	"autoscaler/cred_helper"
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/helpers"
	"autoscaler/models"

	uuid "github.com/nu7hatch/gouuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	MaxRetry = 5
)

type Credentials struct {
	policyDB db.PolicyDB
	maxRetry int
}

func NewWithPolicyDb(policyDb db.PolicyDB, maxRetry int) cred_helper.Credentials {
	return &Credentials{
		policyDB: policyDb,
		maxRetry: maxRetry,
	}
}

func (c *Credentials) Create(appId string, userProvidedCredential *models.Credential) (*models.Credential, error) {
	return createCredential(appId, userProvidedCredential, c.policyDB, c.maxRetry)
}

func (c *Credentials) Delete(appId string) error {
	return deleteCredential(appId, c.policyDB, c.maxRetry)
}

func (c *Credentials) Get(appId string) (*models.Credential, error) {
	return c.policyDB.GetCredential(appId)
}

func (c *Credentials) InitializeConfig(dbConfigs map[db.Name]db.DatabaseConfig, loggingConfig helpers.LoggingConfig) error {
	logger := helpers.InitLoggerFromConfig(&loggingConfig, "custom_metrics_cred_helper")
	var err error
	c.policyDB, err = sqldb.NewPolicySQLDB(dbConfigs[db.PolicyDb], logger.Session("policy-db"))
	if err != nil {
		return err
	}
	c.maxRetry = MaxRetry
	return nil
}

var _ cred_helper.Credentials = &Credentials{}

func _createCredential(appId string, userProvidedCredential *models.Credential, policyDB db.PolicyDB) (*models.Credential, error) {
	var credUsername, credPassword string
	if userProvidedCredential == nil {
		credUsernameUUID, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}
		credPasswordUUID, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}
		credUsername = credUsernameUUID.String()
		credPassword = credPasswordUUID.String()
	} else {
		credUsername = userProvidedCredential.Username
		credPassword = userProvidedCredential.Password
	}

	userNameHash, err := bcrypt.GenerateFromPassword([]byte(credUsername), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(credPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	cred := models.Credential{
		Username: credUsername,
		Password: credPassword,
	}

	err = policyDB.SaveCredential(appId, models.Credential{
		Username: string(userNameHash),
		Password: string(passwordHash),
	})
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

func createCredential(appId string, userProvidedCredential *models.Credential, policyDB db.PolicyDB, maxRetry int) (*models.Credential, error) {
	var err error
	var count int
	var cred *models.Credential
	for {
		if count == maxRetry {
			return nil, err
		}
		cred, err = _createCredential(appId, userProvidedCredential, policyDB)
		if err == nil {
			return cred, nil
		}
		count++
	}
}

func _deleteCredential(appId string, policyDB db.PolicyDB) error {
	err := policyDB.DeleteCredential(appId)
	if err != nil {
		return err
	}
	return nil
}

func deleteCredential(appId string, policyDB db.PolicyDB, maxRetry int) error {
	var err error
	var count int
	for {
		if count == maxRetry {
			return err
		}
		err = _deleteCredential(appId, policyDB)
		if err == nil {
			return nil
		}
		count++
	}
}
