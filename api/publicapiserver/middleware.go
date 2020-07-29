package publicapiserver

import (
	"net/http"
	"strings"

	"autoscaler/api"
	"autoscaler/cf"
	"autoscaler/models"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
)

type Middleware struct {
	logger           lager.Logger
	cf               cf.CFConfig
	cfClient         cf.CFClient
	cfTokenEndpoint  string
	checkBindingFunc api.CheckBindingFunc
}

func NewMiddleware(logger lager.Logger, cfClient cf.CFClient, checkBindingFunc api.CheckBindingFunc) *Middleware {
	return &Middleware{
		logger:           logger,
		cfClient:         cfClient,
		checkBindingFunc: checkBindingFunc,
	}
}

func (mw *Middleware) Oauth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		userToken := r.Header.Get("Authorization")
		if userToken == "" {
			mw.logger.Error("userToken is not present", nil, lager.Data{"url": r.URL.String()})
			handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    "Unauthorized",
				Message: "User token is not present in Authorization header"})
			return
		}
		if !mw.isValidUserToken(userToken) {
			handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    "Unauthorized",
				Message: "Invalid bearer token"})
			return
		}
		appId := vars["appId"]
		if appId == "" {
			mw.logger.Error("appId is not present", nil, lager.Data{"url": r.URL.String()})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad Request",
				Message: "Malformed or missing appId",
			})
			return
		}
		isUserAdmin, err := mw.cfClient.IsUserAdmin(userToken)
		if err != nil {
			mw.logger.Error("failed to check if user is admin", err, nil)
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Interal-Server-Error",
				Message: "Failed to check if user is admin"})
			return
		}
		if isUserAdmin {
			next.ServeHTTP(w, r)
			return
		}
		isUserSpaceDeveloper, err := mw.cfClient.IsUserSpaceDeveloper(userToken, appId)
		if err != nil {
			if err == cf.ErrUnauthrorized {
				handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
					Code:    "Unauthorized",
					Message: "You are not authorized to perform the requested action"})
				return
			} else {
				mw.logger.Error("failed to check space developer permissions", err, nil)
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

		handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
			Code:    "Unauthorized",
			Message: "You are not authorized to perform the requested action"})
		return
	})
}

func (mw *Middleware) CheckServiceBinding(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		appId := vars["appId"]
		if appId == "" {
			mw.logger.Error("appId is not present", nil, lager.Data{"url": r.URL.String()})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad Request",
				Message: "AppId is required",
			})
			return
		}
		hasBinding := mw.checkBindingFunc(appId)
		if hasBinding {
			mw.logger.Debug("binding is present", nil, lager.Data{"appId": appId})
			next.ServeHTTP(w, r)
			return
		}
		mw.logger.Error("binding is not present", nil, lager.Data{"appId": appId})
		http.Error(w, "{ \"error\": \"The application is not bound to Auto-Scaling service\" }", http.StatusForbidden)
		return

	})
}

func (mw *Middleware) RejectCredentialOperationInServiceOffering(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.WriteJSONResponse(w, http.StatusForbidden, models.ErrorResponse{
			Code:    "Forbidden",
			Message: "This command is only valid for build-in auto-scaling capacity. Please operate service credential with \"cf bind/unbind-service\" command.",
		})
		return

	})
}

func (mw *Middleware) isValidUserToken(userToken string) bool {
	lowerCaseToken := strings.ToLower(userToken)
	if !strings.HasPrefix(lowerCaseToken, "bearer ") {
		mw.logger.Error("Token should start with bearer", cf.ErrInvalidTokenFormat)
		return false
	}
	tokenSplitted := strings.Split(lowerCaseToken, " ")
	if len(tokenSplitted) != 2 {
		mw.logger.Error("Token should contain two parts separated by space", cf.ErrInvalidTokenFormat)
		return false
	}

	return true
}
