package publicapiserver

import (
	"autoscaler/cf"
	"autoscaler/models"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
)

type OAuthMiddleware struct {
	logger          lager.Logger
	cf              cf.CFConfig
	cfClient        cf.CFClient
	cfTokenEndpoint string
}

var ErrUnauthrorized = fmt.Errorf(http.StatusText(http.StatusUnauthorized))
var ErrInvalidTokenFormat = fmt.Errorf("Invalid token format")

func NewOauthMiddleware(logger lager.Logger, cfClient cf.CFClient) *OAuthMiddleware {
	return &OAuthMiddleware{
		logger:   logger,
		cfClient: cfClient,
	}
}

func (oam *OAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		userToken := r.Header.Get("Authorization")
		if userToken == "" {
			oam.logger.Error("userToken is not present", nil, lager.Data{"url": r.URL.String()})
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		appId := vars["appId"]
		if appId == "" {
			oam.logger.Error("appId is not present", nil, lager.Data{"url": r.URL.String()})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad Request",
				Message: "Malformed or missing appId",
			})
			return
		}

		isUserAdmin, err := oam.cfClient.IsUserAdmin(userToken)
		if err != nil {
			oam.logger.Error("failed to check if user is admin", err, nil)
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Interal-Server-Error",
				Message: "Failed to check if user is admin"})
			return
		}
		if isUserAdmin {
			next.ServeHTTP(w, r)
			return
		}

		isUserSpaceDeveloper, err := oam.cfClient.IsUserSpaceDeveloper(userToken, appId)
		if err != nil {
			oam.logger.Error("failed to check spacedeveloper permissions", err, nil)
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Interal-Server-Error",
				Message: "Failed to check space developer permission"})
			return
		}
		if isUserSpaceDeveloper {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	})
}
