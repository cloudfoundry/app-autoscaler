package helpers

import (
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
	"golang.org/x/crypto/bcrypt"
)

type BasicAuthenticationMiddleware struct {
	usernameHash []byte
	passwordHash []byte
	logger       lager.Logger
}

// middleware basic authentication middleware functionality for healthcheck
func (bam *BasicAuthenticationMiddleware) BasicAuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, authOK := r.BasicAuth()

		bam.logger.Info("basic-authentication-middleware", lager.Data{"usernameHash": bam.usernameHash, "passwordHash": bam.passwordHash})
		if bam.usernameHash == nil && bam.passwordHash == nil {
			next.ServeHTTP(w, r)
			return
		}

		if !authOK || bcrypt.CompareHashAndPassword(bam.usernameHash, []byte(username)) != nil || bcrypt.CompareHashAndPassword(bam.passwordHash, []byte(password)) != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CreateBasicAuthMiddleware(logger lager.Logger, ba models.BasicAuth) (*BasicAuthenticationMiddleware, error) {
	var basicAuthentication *BasicAuthenticationMiddleware
	usernameHash, username, passwordHash, password := ba.UsernameHash, ba.Username, ba.PasswordHash, ba.Password

	usernameHashByte, err := getUserHashBytes(logger, usernameHash, username)
	if err != nil {
		return nil, err
	}

	passwordHashByte, err := getPasswordHashBytes(logger, passwordHash, password)
	if err != nil {
		return nil, err
	}

	basicAuthentication = &BasicAuthenticationMiddleware{
		usernameHash: usernameHashByte,
		passwordHash: passwordHashByte,
		logger:       logger,
	}
	return basicAuthentication, nil
}

func getUserHashBytes(logger lager.Logger, usernameHash string, username string) ([]byte, error) {
	var usernameHashByte []byte
	var err error
	if usernameHash == "" {
		if len(username) > 72 {
			logger.Error("warning-configured-username-too-long-using-only-first-72-characters", bcrypt.ErrPasswordTooLong, lager.Data{"username-length": len(username)})
			username = username[:72]
		}
		// when username and password are set for health check
		usernameHashByte, err = bcrypt.GenerateFromPassword([]byte(username), bcrypt.MinCost) // use MinCost as the config already provided it as cleartext
		if err != nil {
			logger.Error("failed-new-server-username", err)
			return nil, err
		}
	} else {
		usernameHashByte = []byte(usernameHash)
	}
	return usernameHashByte, err
}

func getPasswordHashBytes(logger lager.Logger, passwordHash string, password string) ([]byte, error) {
	var passwordHashByte []byte
	var err error
	if passwordHash == "" {
		if len(password) > 72 {
			logger.Error("warning-configured-password-too-long-using-only-first-72-characters", bcrypt.ErrPasswordTooLong, lager.Data{"password-length": len(password)})
			password = password[:72]
		}
		passwordHashByte, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost) // use MinCost as the config already provided it as cleartext
		if err != nil {
			logger.Error("failed-new-server-password", err)
			return nil, err
		}
	} else {
		passwordHashByte = []byte(passwordHash)
	}
	return passwordHashByte, nil
}
