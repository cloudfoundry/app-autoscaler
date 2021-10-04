package auth

import (
	"autoscaler/db"
	"autoscaler/metricsforwarder/server/common"
	"autoscaler/models"
	"errors"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	"github.com/patrickmn/go-cache"
)

var ErrorAuthNotFound = errors.New("authentication method not found")

type Auth struct {
	logger                     lager.Logger
	credentialCache            cache.Cache
	policyDB                   db.PolicyDB
	cacheTTL                   time.Duration
	metricsForwarderMtlsCACert string
}

func New(logger lager.Logger, policyDB db.PolicyDB, credentialCache cache.Cache, cacheTTL time.Duration, metricsForwarderMtlsCACert string) *Auth {
	return &Auth{logger: logger, credentialCache: credentialCache, policyDB: policyDB, cacheTTL: cacheTTL, metricsForwarderMtlsCACert: metricsForwarderMtlsCACert}
}

func (a *Auth) Authenticate(next http.Handler) http.Handler {
	return common.VarsFunc(a.AuthenticateHandler(next))
}

func (a *Auth) AuthenticateHandler(next http.Handler) func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		err := a.CheckAuth(r, a.metricsForwarderMtlsCACert, vars["appid"])
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

func (a *Auth) CheckAuth(r *http.Request, metricsForwarderMtlsCACert string, appID string) error {
	var errAuth error
	isMtlsConfigured := isMtlsConfigured(metricsForwarderMtlsCACert)

	if isMtlsConfigured {
		errAuth = a.XFCCAuth(r, appID)
	}
	if errors.Is(errAuth, ErrorMTLSHeaderNotFound) || !isMtlsConfigured {
		a.logger.Info("Trying basic auth", lager.Data{"error": fmt.Sprintf("%+v", errAuth), "isMtlsConfigured": isMtlsConfigured})
		errAuth = a.BasicAuth(r, appID)
	}
	return errAuth
}

func isMtlsConfigured(metricsForwarderMtlsCACert string) bool {
	return metricsForwarderMtlsCACert != ""
}
