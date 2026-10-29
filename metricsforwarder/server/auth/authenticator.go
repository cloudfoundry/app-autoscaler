package auth

import (
	"errors"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	xfcc "code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server/common"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	"code.cloudfoundry.org/lager/v3"
)

var ErrorAuthNotFound = errors.New("authentication method not found")

type Auth struct {
	logger      lager.Logger
	credentials cred_helper.Credentials
	bindingDB   db.BindingDB
	verifier    *xfcc.ChainVerifier
}

func New(logger lager.Logger, credentials cred_helper.Credentials, bindingDB db.BindingDB, config models.XFCCAuth) (*Auth, error) {
	verifier, err := xfcc.NewChainVerifier(logger, config)
	if err != nil {
		return nil, err
	}
	return &Auth{
		logger:      logger,
		credentials: credentials,
		bindingDB:   bindingDB,
		verifier:    verifier,
	}, nil
}

func (a *Auth) Authenticate(next http.Handler) http.Handler {
	return common.VarsFunc(a.AuthenticateHandler(next))
}

func (a *Auth) AuthenticateHandler(next http.Handler) func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		err := a.CheckAuth(r, vars["appid"])
		if err != nil {
			a.logger.Info("Authentication Failed", lager.Data{"error": err.Error()})
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
	errAuth = a.XFCCAuth(r, a.bindingDB, appID)
	if errAuth != nil {
		isBasicAuthAvailable := a.credentials != nil
		if errors.Is(errAuth, ErrXFCCHeaderNotFound) && isBasicAuthAvailable {
			a.logger.Info("Trying basic auth", lager.Data{"app_id": appID})
			errAuth = a.BasicAuth(r, appID)
		}
	}
	return errAuth
}
