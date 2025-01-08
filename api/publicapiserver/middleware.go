package publicapiserver

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
)

type Middleware struct {
	logger           lager.Logger
	cfClient         cf.CFClient
	clientId         string
	checkBindingFunc api.CheckBindingFunc
}

func NewMiddleware(logger lager.Logger, cfClient cf.CFClient, checkBindingFunc api.CheckBindingFunc, clientId string) *Middleware {
	return &Middleware{
		logger:           logger,
		cfClient:         cfClient,
		clientId:         clientId,
		checkBindingFunc: checkBindingFunc,
	}
}

func (mw *Middleware) Oauth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		authHeaderValue := r.Header.Get("Authorization")
		if authHeaderValue == "" {
			mw.logger.Error("authorization-header-is-not-present", nil, lager.Data{"url": r.URL.String()})
			handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    "Unauthorized",
				Message: "Authorization header is not present"})
			return
		}
		userToken, err := mw.extractBearerToken(authHeaderValue)
		if err != nil {
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
				Code:    http.StatusText(http.StatusInternalServerError),
				Message: "Failed to check if user is admin"})
			return
		}
		if isUserAdmin {
			next.ServeHTTP(w, r)
			return
		}
		isUserSpaceDeveloper, err := mw.cfClient.IsUserSpaceDeveloper(userToken, cf.Guid(appId))
		if err != nil {
			switch {
			case cf.IsNotFound(err):
				handlers.WriteJSONResponse(w, http.StatusNotFound, models.ErrorResponse{
					Code:    "App not found",
					Message: "The app guid supplied does not exist"})
				return
			case errors.Is(err, cf.ErrUnauthorized):
				handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
					Code:    "Unauthorized",
					Message: "You are not authorized to perform the requested action"})
				return
			default:
				mw.logger.Error("failed to check space developer permissions", err, nil)
				handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
					Code:    "Internal-Server-Error",
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
	})
}

func (mw *Middleware) extractBearerToken(token string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
		mw.logger.Error("check-bearer-scheme", fmt.Errorf("authorization credentials should specify bearer scheme"))
		return "", cf.ErrInvalidTokenFormat
	}
	tokenSplitted := strings.Split(token, " ")
	if len(tokenSplitted) != 2 {
		mw.logger.Error("split-auth-credentials", fmt.Errorf("authorization credentials should contain scheme and token separated by space"))
		return "", cf.ErrInvalidTokenFormat
	}

	return tokenSplitted[1], nil
}

func (mw *Middleware) HasClientToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mw.clientId == "" {
			next.ServeHTTP(w, r)
			return
		}
		clientToken := r.Header.Get("X-Autoscaler-Token")
		if clientToken == "" {
			mw.logger.Error("client token is not present", nil, lager.Data{"url": r.URL.String()})
			writeErrorResponse(w, http.StatusUnauthorized, "client token is not present in X-Autoscaler-Token header. Are you using the correct API endpoint?")
			return
		}
		isTokenAuthorized, err := mw.cfClient.IsTokenAuthorized(clientToken, mw.clientId)
		if err != nil {
			if errors.Is(err, cf.ErrUnauthorized) {
				writeErrorResponse(w, http.StatusUnauthorized, "client is not authorized to perform the requested action")
				return
			} else {
				mw.logger.Error("failed to check if token is authorized", err)
				writeErrorResponse(w, http.StatusInternalServerError, "failed to check if token is authorized")
				return
			}
		}
		if isTokenAuthorized {
			next.ServeHTTP(w, r)
			return
		}

		writeErrorResponse(w, http.StatusUnauthorized, "client is not authorized to perform the requested action")
	})
}
