package auth

import (
	"errors"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server/common"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	"github.com/patrickmn/go-cache"
)

var ErrorAuthNotFound = errors.New("authentication method not found")
var ErrCAFileEmpty = errors.New("CA file empty")

type Auth struct {
	logger          lager.Logger
	credentialCache cache.Cache
	credentials     cred_helper.Credentials
	cacheTTL        time.Duration
}

func New(logger lager.Logger, credentials cred_helper.Credentials, credentialCache cache.Cache, cacheTTL time.Duration) (*Auth, error) {
	return &Auth{
		logger:          logger,
		credentialCache: credentialCache,
		credentials:     credentials,
		cacheTTL:        cacheTTL,
	}, nil
}

func (a *Auth) Authenticate(next http.Handler) http.Handler {
	return common.VarsFunc(a.AuthenticateHandler(next))
}

func (a *Auth) AuthenticateHandler(next http.Handler) func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		err := a.CheckAuth(r, vars["appid"])
		if err != nil {
			a.logger.Info("Authentication Failed:", lager.Data{"error": err.Error()})
			if errors.Is(err, ErrorAppIDWrong) {
				handlers.WriteJSONResponse(w, http.StatusForbidden, models.ErrorResponse{
					Code:    http.StatusText(http.StatusForbidden),
					Message: "Unauthorized"})
			} else {
				handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
					Code:    http.StatusText(http.StatusUnauthorized),
					Message: "Unauthorized"})
			}
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (a *Auth) CheckAuth(r *http.Request, appID string) error {
	var errAuth error
	errAuth = a.XFCCAuth(r, appID)
	if errAuth != nil {
		if errors.Is(errAuth, ErrXFCCHeaderNotFound) {
			a.logger.Info("Trying basic auth")
			errAuth = a.BasicAuth(r, appID)
		}
	}
	return errAuth
}
