package publicapiserver

import (
	"autoscaler/cf"
	"autoscaler/models"
	"net/http"
	"strings"

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
			handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    "Unauthorized",
				Message: "Authorization is required"})
			return
		}
		if !oam.isValidUserToken(userToken) {
			handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    "Unauthorized",
				Message: "Authorization is invalid formated"})
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
		isUserSpaceDeveloper, err := oam.cfClient.IsUserSpaceDeveloper(userToken, appId)
		if err != nil {
			if err == cf.ErrUnauthrorized {
				handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
					Code:    "Unauthorized",
					Message: "You are not authorized to perform the requested action"})
				return
			} else {
				oam.logger.Error("failed to check space developer permissions", err, nil)
				handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
					Code:    "Interal-Server-Error",
					Message: "Failed to check space developer permission"})
				return
			}

		}
		if isUserSpaceDeveloper {
			next.ServeHTTP(w, r)
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

		handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
			Code:    "Unauthorized",
			Message: "You are not authorized to perform the requested action"})
		return
	})
}

func (oam *OAuthMiddleware) isValidUserToken(userToken string) bool {
	lowerCaseToken := strings.ToLower(userToken)
	if !strings.HasPrefix(lowerCaseToken, "bearer ") {
		oam.logger.Error("Token should start with bearer", cf.ErrInvalidTokenFormat)
		return false
	}
	tokenSplitted := strings.Split(lowerCaseToken, " ")
	if len(tokenSplitted) != 2 {
		oam.logger.Error("Token should contain two parts separated by space", cf.ErrInvalidTokenFormat)
		return false
	}

	return true
}
