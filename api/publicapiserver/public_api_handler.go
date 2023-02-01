package publicapiserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/policyvalidator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/schedulerutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	"code.cloudfoundry.org/lager"
	uuid "github.com/nu7hatch/gouuid"
)

type PublicApiHandler struct {
	logger               lager.Logger
	conf                 *config.Config
	policydb             db.PolicyDB
	bindingdb            db.BindingDB
	scalingEngineClient  *http.Client
	eventGeneratorClient *http.Client
	policyValidator      *policyvalidator.PolicyValidator
	schedulerUtil        *schedulerutil.SchedulerUtil
	credentials          cred_helper.Credentials
}

func NewPublicApiHandler(logger lager.Logger, conf *config.Config, policydb db.PolicyDB, bindingdb db.BindingDB, credentials cred_helper.Credentials) *PublicApiHandler {
	seClient, err := helpers.CreateHTTPClient(&conf.ScalingEngine.TLSClientCerts)
	if err != nil {
		logger.Error("Failed to create http client for ScalingEngine", err, lager.Data{"scalingengine": conf.ScalingEngine.TLSClientCerts})
		os.Exit(1)
	}

	egClient, err := helpers.CreateHTTPClient(&conf.EventGenerator.TLSClientCerts)
	if err != nil {
		logger.Error("Failed to create http client for EventGenerator", err, lager.Data{"eventgenerator": conf.EventGenerator.TLSClientCerts})
		os.Exit(1)
	}

	return &PublicApiHandler{
		logger:               logger,
		conf:                 conf,
		policydb:             policydb,
		bindingdb:            bindingdb,
		scalingEngineClient:  seClient,
		eventGeneratorClient: egClient,
		policyValidator:      policyvalidator.NewPolicyValidator(conf.PolicySchemaPath, conf.ScalingRules.CPU.LowerThreshold, conf.ScalingRules.CPU.UpperThreshold),
		schedulerUtil:        schedulerutil.NewSchedulerUtil(conf, logger),
		credentials:          credentials,
	}
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	handlers.WriteJSONResponse(w, statusCode, models.ErrorResponse{
		Code:    http.StatusText(statusCode),
		Message: message})
}

func (h *PublicApiHandler) GetScalingPolicy(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		writeErrorResponse(w, http.StatusBadRequest, "AppId is required")
		return
	}
	logger := h.logger.Session("GetScalingPolicy", lager.Data{"appId": appId})
	logger.Info("Get Scaling Policy")

	scalingPolicy, err := h.policydb.GetAppPolicy(r.Context(), appId)
	if err != nil {
		logger.Error("Failed to retrieve scaling policy from database", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving scaling policy")
		return
	}

	if scalingPolicy == nil {
		logger.Info("policy doesn't exist")
		writeErrorResponse(w, http.StatusNotFound, "Policy Not Found")
		return
	}

	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err = jsonEncoder.Encode(scalingPolicy)
	if err != nil {
		logger.Error("Failed to json encode scaling policy", err, lager.Data{"policy": fmt.Sprintf("%+v", scalingPolicy)})
		writeErrorResponse(w, http.StatusInternalServerError, "Error encode scaling policy")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bf.Bytes())
	if err != nil {
		logger.Error("failed-to-write-body", err)
	}
}

func (h *PublicApiHandler) AttachScalingPolicy(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		writeErrorResponse(w, http.StatusBadRequest, "AppId is required")
		return
	}

	logger := h.logger.Session("AttachScalingPolicy", lager.Data{"appId": appId})
	logger.Info("Attach Scaling Policy")

	policyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to read request body")
		return
	}

	policy := string(policyBytes)

	errResults, valid, policy := h.policyValidator.ValidatePolicy(policy)
	if !valid {
		logger.Error("Failed to validate policy", nil, lager.Data{"errors": errResults})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, errResults)
		return
	}

	policyGuid, err := uuid.NewV4()
	if err != nil {
		logger.Error("Failed to generate policy guid", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error generating policy guid")
		return
	}

	err = h.policydb.SaveAppPolicy(r.Context(), appId, policy, policyGuid.String())
	if err != nil {
		logger.Error("Failed to save policy", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error saving policy")
		return
	}
	h.logger.Info("creating/updating schedules", lager.Data{"policy": policy})
	err = h.schedulerUtil.CreateOrUpdateSchedule(r.Context(), appId, policy, policyGuid.String())
	if err != nil {
		logger.Error("Failed to create/update schedule", err)
	}
	w.WriteHeader(http.StatusOK)
	_, err = io.WriteString(w, policy)
	if err != nil {
		logger.Error("Failed to write body", err)
	}
}

func (h *PublicApiHandler) DetachScalingPolicy(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		writeErrorResponse(w, http.StatusBadRequest, "AppId is required")
		return
	}
	logger := h.logger.Session("DetachScalingPolicy", lager.Data{"appId": appId})
	logger.Info("Deleting policy json", lager.Data{"appId": appId})
	err := h.policydb.DeletePolicy(r.Context(), appId)
	if err != nil {
		logger.Error("Failed to delete policy from database", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting policy")
		return
	}
	logger.Info("Deleting schedules")
	err = h.schedulerUtil.DeleteSchedule(r.Context(), appId)
	if err != nil {
		logger.Error("Failed to delete schedule", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting schedules")
		return
	}

	if h.bindingdb != nil && !reflect.ValueOf(h.bindingdb).IsNil() {
		//TODO this is a copy of part of the attach ... this should use a common function.
		// brokered offering: check if there's a default policy that could apply
		serviceInstance, err := h.bindingdb.GetServiceInstanceByAppId(appId)
		if err != nil {
			logger.Error("Failed to find service instance for app", err)
			writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving service instance")
			return
		}
		if serviceInstance.DefaultPolicy != "" {
			policyStr := serviceInstance.DefaultPolicy
			policyGuidStr := serviceInstance.DefaultPolicyGuid
			logger.Info("saving default policy json for app", lager.Data{"policy": policyStr})
			err = h.policydb.SaveAppPolicy(r.Context(), appId, policyStr, policyGuidStr)
			if err != nil {
				logger.Error("failed to save policy", err, lager.Data{"policy": policyStr})
				writeErrorResponse(w, http.StatusInternalServerError, "Error attaching the default policy")
				return
			}

			logger.Info("creating/updating schedules", lager.Data{"policy": policyStr})
			err = h.schedulerUtil.CreateOrUpdateSchedule(r.Context(), appId, policyStr, policyGuidStr)
			//while there is synchronization between policy and schedule, so creating schedule error does not break
			//the whole creating binding process
			if err != nil {
				logger.Error("failed to create/update schedules", err, lager.Data{"policy": policyStr})
			}
		}
	}
	// find via the app id the binding -> service instance
	// default policy? then apply that

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("{}"))
	if err != nil {
		logger.Error("failed-to-write-body", err)
	}
}

func (h *PublicApiHandler) GetScalingHistories(w http.ResponseWriter, req *http.Request, vars map[string]string) {
	appId := vars["appId"]
	logger := h.logger.Session("GetScalingHistories", lager.Data{"appId": appId})
	logger.Info("Get ScalingHistories")

	parameters, err := parseParameter(req, vars)
	if err != nil {
		logger.Error("Bad Request", err, lager.Data{"appId": appId})
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	pathFn := func() string {
		path, _ := routes.ScalingEngineRoutes().Get(routes.GetScalingHistoriesRouteName).URLPath("appid", appId)
		return h.conf.ScalingEngine.ScalingEngineUrl + path.RequestURI() + "?" + parameters.Encode()
	}
	proxyRequest(pathFn, h.scalingEngineClient.Get, w, req.URL, parameters, "scaling history from scaling engine", logger)
}

func proxyRequest(pathFn func() string, call func(url string) (*http.Response, error), w http.ResponseWriter, reqUrl *url.URL, parameters *url.Values, requestDescription string, logger lager.Logger) {
	aUrl := pathFn()
	resp, err := call(aUrl)
	if err != nil {
		logger.Error("Failed to retrieve "+requestDescription, err, lager.Data{"url": aUrl})
		writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving "+requestDescription)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error occurred during parsing "+requestDescription+" result", err, lager.Data{"url": aUrl})
		writeErrorResponse(w, http.StatusInternalServerError, "Error parsing "+requestDescription)
		return
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("Error occurred during getting "+requestDescription, nil, lager.Data{"statusCode": resp.StatusCode, "body": string(responseData), "url": aUrl})
		writeErrorResponse(w, resp.StatusCode, string(responseData))
		return
	}
	paginatedResponse, err := paginateResource(responseData, parameters, reqUrl)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	handlers.WriteJSONResponse(w, resp.StatusCode, paginatedResponse)
}

func (h *PublicApiHandler) GetAggregatedMetricsHistories(w http.ResponseWriter, req *http.Request, vars map[string]string) {
	appId := vars["appId"]
	metricType := vars["metricType"]
	logger := h.logger.Session("GetScalingHistories", lager.Data{"appId": appId, "metricType": metricType})
	logger.Info("Get AggregatedMetricHistories", lager.Data{"appId": appId, "metricType": metricType})

	parameters, err := parseParameter(req, vars)
	if err != nil {
		logger.Error("Bad Request", err)
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if metricType == "" {
		logger.Error("Bad Request", nil)
		writeErrorResponse(w, http.StatusBadRequest, "Metrictype is required")
		return
	}

	pathFn := func() string {
		path, _ := routes.EventGeneratorRoutes().Get(routes.GetAggregatedMetricHistoriesRouteName).URLPath("appid", appId, "metrictype", metricType)
		return h.conf.EventGenerator.EventGeneratorUrl + path.RequestURI() + "?" + parameters.Encode()
	}
	proxyRequest(pathFn, h.eventGeneratorClient.Get, w, req.URL, parameters, "metrics history from eventgenerator", logger)
}

func (h *PublicApiHandler) GetApiInfo(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
	info, err := os.ReadFile(h.conf.InfoFilePath)
	if err != nil {
		h.logger.Error("Failed to info file", err, lager.Data{"info-file-path": h.conf.InfoFilePath})
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to load info")
		return
	}

	_, err = w.Write(info)
	if err != nil {
		h.logger.Error("failed-to-write-body", err)
	}
}

func (h *PublicApiHandler) GetHealth(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
	_, err := w.Write([]byte(`{"alive":"true"}`))
	if err != nil {
		h.logger.Error("failed-to-write-body", err)
	}
}

func (h *PublicApiHandler) CreateCredential(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		writeErrorResponse(w, http.StatusBadRequest, "AppId is required")
		return
	}
	logger := h.logger.Session("CreateCredential", lager.Data{"appId": appId})
	var userProvidedCredential *models.Credential
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read user provided credential request body", err, lager.Data{"appId": appId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error creating credential")
		return
	}
	if len(bodyBytes) > 0 {
		userProvidedCredential = &models.Credential{}
		err = json.Unmarshal(bodyBytes, userProvidedCredential)
		if err != nil {
			logger.Error("Failed to unmarshal user provided credential", err, lager.Data{"body": bodyBytes})
			writeErrorResponse(w, http.StatusBadRequest, "Invalid credential format")
			return
		}
		if !(userProvidedCredential.Username != "" && userProvidedCredential.Password != "") {
			logger.Info("Username or password is missing", lager.Data{"userProvidedCredential": userProvidedCredential})
			writeErrorResponse(w, http.StatusBadRequest, "Username and password are both required")
			return
		}
	}

	logger.Info("Create credential")
	cred, err := h.credentials.Create(r.Context(), appId, userProvidedCredential)
	if err != nil {
		logger.Error("Failed to create credential", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error creating credential")
		return
	}
	handlers.WriteJSONResponse(w, http.StatusOK, struct {
		AppId string `json:"app_id"`
		*models.Credential
		Url string `json:"url"`
	}{
		AppId:      appId,
		Credential: cred,
		Url:        h.conf.MetricsForwarder.MetricsForwarderUrl,
	})
}

func (h *PublicApiHandler) DeleteCredential(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		writeErrorResponse(w, http.StatusBadRequest, "AppId is required")
		return
	}
	logger := h.logger.Session("DeleteCredential", lager.Data{"appId": appId})
	logger.Info("Delete credential")
	err := h.credentials.Delete(r.Context(), appId)
	if err != nil {
		logger.Error("Failed to delete credential", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting credential")
		return
	}
	handlers.WriteJSONResponse(w, http.StatusOK, nil)
}
