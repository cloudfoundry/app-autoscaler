package auth

import (
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

func (a *Auth) BasicAuth(r *http.Request, appID string) error {
	username, password, parseOK := r.BasicAuth()

	if !parseOK {
		return ErrorAuthNotFound
	}

	return a.credentials.Validate(appID, models.Credential{Username: username, Password: password})
}
