package cred_helper

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/patrickmn/go-cache"
	"golang.org/x/crypto/bcrypt"
)

var _ Credentials = &customMetricsCredentials{}

type customMetricsCredentials struct {
	policyDB        db.PolicyDB
	maxRetry        int
	credentialCache cache.Cache
	cacheTTL        time.Duration
	logger          lager.Logger
}

func (c *customMetricsCredentials) Ping() error {
	return c.policyDB.Ping()
}

func (c *customMetricsCredentials) Close() error {
	return c.policyDB.Close()
}

func NewCustomMetricsCredHelper(policyDb db.PolicyDB, maxRetry int, logger lager.Logger) Credentials {
	return &customMetricsCredentials{
		policyDB: policyDb,
		maxRetry: maxRetry,
		logger:   logger,
	}
}

func NewCustomMetricsCredHelperWithCache(policyDb db.PolicyDB, maxRetry int, credentialCache cache.Cache, cacheTTL time.Duration, logger lager.Logger) Credentials {
	return &customMetricsCredentials{
		policyDB:        policyDb,
		maxRetry:        maxRetry,
		credentialCache: credentialCache,
		cacheTTL:        cacheTTL,
		logger:          logger,
	}
}

func (c *customMetricsCredentials) Create(appId string, userProvidedCredential *models.Credential) (*models.Credential, error) {
	return createCredential(appId, userProvidedCredential, c.policyDB, c.maxRetry)
}

func (c *customMetricsCredentials) Delete(appId string) error {
	return deleteCredential(appId, c.policyDB, c.maxRetry)
}

func (c *customMetricsCredentials) Validate(appId string, credential models.Credential) (bool, error) {
	var isValid bool

	res, found := c.credentialCache.Get(appId)
	if found {
		// Credentials found in cache
		credentials := res.(*models.Credential)
		isValid = validateCredentials(credential.Username, credentials.Username, credential.Password, credentials.Password)
	}

	// Credentials not found in cache or
	// stale cache entry with invalid credential found in cache
	// search in the database and update the cache
	if !found || !isValid {
		credentials, err := c.policyDB.GetCredential(appId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.logger.Error("no-credential-found-in-db", err, lager.Data{"appId": appId})
				return false, errors.New("basic authorization credential does not match")
			}
			c.logger.Error("error-during-getting-credentials-from-policyDB", err, lager.Data{"appId": appId})
			return false, fmt.Errorf("error getting binding credentials from policyDB %w", err)
		}
		// update the cache
		c.credentialCache.Set(appId, credentials, c.cacheTTL)

		return validateCredentials(credential.Username, credentials.Username, credential.Password, credentials.Password), nil
	}

	return isValid, nil
}

var _ Credentials = &customMetricsCredentials{}

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

func validateCredentials(username string, usernameHash string, password string, passwordHash string) bool {
	usernameAuthErr := bcrypt.CompareHashAndPassword([]byte(usernameHash), []byte(username))
	passwordAuthErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if usernameAuthErr == nil && passwordAuthErr == nil { // password matching successful
		return true
	}
	return false
}
