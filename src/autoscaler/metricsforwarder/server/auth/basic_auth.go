package auth

import (
	"autoscaler/models"
	"database/sql"
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"code.cloudfoundry.org/lager"
)

func (a *Auth) BasicAuth(w http.ResponseWriter, r *http.Request, appID string) error {
	w.Header().Set("Content-Type", "application/json")

	username, password, parseOK := r.BasicAuth()

	if !parseOK {
		return ErrorAuthNotFound
	}

	var isValid bool

	res, found := a.credentialCache.Get(appID)
	if found {
		// Credentials found in cache
		credentials := res.(*models.Credential)
		isValid = a.validateCredentials(username, credentials.Username, password, credentials.Password)
	}

	// Credentials not found in cache or
	// stale cache entry with invalid credential found in cache
	// search in the database and update the cache
	if !found || !isValid {
		credentials, err := a.policyDB.GetCredential(appID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				a.logger.Error("no-credential-found-in-db", err, lager.Data{"appID": appID})
				return errors.New("basic authorization credential does not match")
			}
			a.logger.Error("error-during-getting-credentials-from-policyDB", err, lager.Data{"appid": appID})
			return errors.New("error getting binding credentials from policyDB")
		}
		// update the cache
		a.credentialCache.Set(appID, credentials, a.cacheTTL)

		isValid = a.validateCredentials(username, credentials.Username, password, credentials.Password)
		// If Credentials in DB is not valid
		if !isValid {
			a.logger.Error("error-validating-authorization-header", err)
			return errors.New("db basic authorization credential does not match")
		}
	}

	return nil
}

func (a *Auth) validateCredentials(username string, usernameHash string, password string, passwordHash string) bool {
	usernameAuthErr := bcrypt.CompareHashAndPassword([]byte(usernameHash), []byte(username))
	passwordAuthErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if usernameAuthErr == nil && passwordAuthErr == nil { // password matching successful
		return true
	}
	return false
}
