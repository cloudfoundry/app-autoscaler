package publicapiserver

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"autoscaler/api/config"
	"autoscaler/api/custom_metrics_cred_helper"
	"autoscaler/api/policyvalidator"
	"autoscaler/api/schedulerutil"
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"autoscaler/routes"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	uuid "github.com/nu7hatch/gouuid"
)

type PublicApiHandler struct {
	logger                 lager.Logger
	conf                   *config.Config
	policydb               db.PolicyDB
	scalingEngineClient    *http.Client
	metricsCollectorClient *http.Client
	eventGeneratorClient   *http.Client
	policyValidator        *policyvalidator.PolicyValidator
	schedulerUtil          *schedulerutil.SchedulerUtil
}

func NewPublicApiHandler(logger lager.Logger, conf *config.Config, policydb db.PolicyDB) *PublicApiHandler {
	seClient, err := helpers.CreateHTTPClient(&conf.ScalingEngine.TLSClientCerts)
	if err != nil {
		logger.Error("Failed to create http client for ScalingEngine", err, lager.Data{"scalingengine": conf.ScalingEngine.TLSClientCerts})
		os.Exit(1)
	}
	mcClient, err := helpers.CreateHTTPClient(&conf.MetricsCollector.TLSClientCerts)
	if err != nil {
		logger.Error("Failed to create http client for MetricsCollector", err, lager.Data{"metricscollector": conf.MetricsCollector.TLSClientCerts})
		os.Exit(1)
	}
	egClient, err := helpers.CreateHTTPClient(&conf.EventGenerator.TLSClientCerts)
	if err != nil {
		logger.Error("Failed to create http client for EventGenerator", err, lager.Data{"eventgenerator": conf.EventGenerator.TLSClientCerts})
		os.Exit(1)
	}
	return &PublicApiHandler{
		logger:                 logger,
		conf:                   conf,
		policydb:               policydb,
		scalingEngineClient:    seClient,
		metricsCollectorClient: mcClient,
		eventGeneratorClient:   egClient,
		policyValidator:        policyvalidator.NewPolicyValidator(conf.PolicySchemaPath),
		schedulerUtil:          schedulerutil.NewSchedulerUtil(conf, logger),
	}
}

func (h *PublicApiHandler) GetScalingPolicy(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "AppId is required",
		})
		return
	}

	h.logger.Info("Get Scaling Policy", lager.Data{"appId": appId})

	scalingPolicy, err := h.policydb.GetAppPolicy(appId)
	if err != nil {
		h.logger.Error("Failed to retrieve scaling policy from database", err, lager.Data{"appId": appId, "err": err})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error retrieving scaling policy"})
		return
	}

	if scalingPolicy == nil {
		h.logger.Info("policy doesn't exist", lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusNotFound, models.ErrorResponse{
			Code:    "Not Found",
			Message: "Policy Not Found"})
		return
	}

	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err = jsonEncoder.Encode(scalingPolicy)
	if err != nil {
		h.logger.Error("Failed to json encode scaling policy", err, lager.Data{"appId": appId, "policy": scalingPolicy})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error encode scaling policy"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(bf.Bytes())))
	w.WriteHeader(http.StatusOK)
	w.Write(bf.Bytes())
}

func (h *PublicApiHandler) AttachScalingPolicy(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "AppId is required",
		})
		return
	}

	h.logger.Info("Attach Scaling Policy", lager.Data{"appId": appId})

	policyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", err, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to read request body"})
		return
	}

	policyStr := string(policyBytes)

	errResults, valid := h.policyValidator.ValidatePolicy(policyStr)
	if !valid {
		h.logger.Error("Failed to validate policy", nil, lager.Data{"errResults": errResults})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, errResults)
		return
	}

	policyGuid, err := uuid.NewV4()
	if err != nil {
		h.logger.Error("Failed to generate policy guid", err, nil)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error generating policy guid"})
		return
	}

	h.logger.Info("saving policy json", lager.Data{"policy": policyStr})
	err = h.policydb.SaveAppPolicy(appId, policyStr, policyGuid.String())
	if err != nil {
		h.logger.Error("Failed to save policy", err, nil)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error saving policy"})
		return
	}

	h.logger.Info("creating/updating schedules", lager.Data{"policy": policyStr})
	err = h.schedulerUtil.CreateOrUpdateSchedule(appId, policyStr, policyGuid.String())
	if err != nil {
		h.logger.Error("Failed to create/update schedule", err, nil)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(policyBytes))
}

func (h *PublicApiHandler) DetachScalingPolicy(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "AppId is required",
		})
		return
	}

	h.logger.Info("Deleting policy json", lager.Data{"appId": appId})
	err := h.policydb.DeletePolicy(appId)
	if err != nil {
		h.logger.Error("Failed to delete policy from database", err, nil)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error deleting policy"})
		return
	}
	h.logger.Info("Deleting schedules", lager.Data{"appId": appId})
	err = h.schedulerUtil.DeleteSchedule(appId)
	if err != nil {
		h.logger.Error("Failed to delete schedule", err, nil)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error deleting schedules"})
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

func (h *PublicApiHandler) GetScalingHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]

	h.logger.Info("Get ScalingHistories", lager.Data{"appId": appId})

	parameters, err := parseParameter(r, vars)
	if err != nil {
		h.logger.Error("Bad Request", err, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: err.Error(),
		})
		return
	}

	path, _ := routes.ScalingEngineRoutes().Get(routes.GetScalingHistoriesRouteName).URLPath("appid", appId)

	url := h.conf.ScalingEngine.ScalingEngineUrl + path.RequestURI() + "?" + parameters.Encode()

	resp, err := h.scalingEngineClient.Get(url)
	if err != nil {
		h.logger.Error("Failed to retrieve scaling history from scaling engine", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error retrieving scaling history from scaling engine"})
		return
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Error occured during parsing scaling histories result", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error parsing scaling history from scaling engine"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Error occured during getting scaling histories", nil, lager.Data{"statusCode": resp.StatusCode, "body": string(responseData)})
		handlers.WriteJSONResponse(w, resp.StatusCode, models.ErrorResponse{
			Code:    http.StatusText(resp.StatusCode),
			Message: string(responseData)})
		return
	}
	paginatedResponse, err := paginateResource(responseData, parameters, r)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: err.Error()})
		return
	}

	handlers.WriteJSONResponse(w, resp.StatusCode, paginatedResponse)
}

func (h *PublicApiHandler) GetAggregatedMetricsHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	metricType := vars["metricType"]

	h.logger.Info("Get AggregatedMetricHistories", lager.Data{"appId": appId, "metricType": metricType})

	parameters, err := parseParameter(r, vars)
	if err != nil {
		h.logger.Error("Bad Request", err, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: err.Error(),
		})
		return
	}
	if metricType == "" {
		h.logger.Error("Bad Request", nil, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "Metrictype is required",
		})
		return
	}

	path, _ := routes.EventGeneratorRoutes().Get(routes.GetAggregatedMetricHistoriesRouteName).URLPath("appid", appId, "metrictype", metricType)

	url := h.conf.EventGenerator.EventGeneratorUrl + path.RequestURI() + "?" + parameters.Encode()

	resp, err := h.eventGeneratorClient.Get(url)
	if err != nil {
		h.logger.Error("Failed to retrieve metrics history from eventgenerator", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error retrieving metrics history from eventgenerator"})
		return
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Error occured during parsing metrics histories result", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error parsing metric history from eventgenerator"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Error occured during getting metric histories", nil, lager.Data{"statusCode": resp.StatusCode, "body": string(responseData)})
		handlers.WriteJSONResponse(w, resp.StatusCode, models.ErrorResponse{
			Code:    http.StatusText(resp.StatusCode),
			Message: string(responseData)})
		return
	}
	paginatedResponse, err := paginateResource(responseData, parameters, r)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: err.Error()})
		return
	}

	handlers.WriteJSONResponse(w, resp.StatusCode, paginatedResponse)
}

func (h *PublicApiHandler) GetInstanceMetricsHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]

	metricType := vars["metricType"]
	instanceIndex := r.URL.Query().Get("instance-index")

	h.logger.Info("GetInstanceMetricsHistories", lager.Data{"appId": appId, "metricType": metricType, "instanceIndex": instanceIndex})

	parameters, err := parseParameter(r, vars)
	if err != nil {
		h.logger.Error("Bad Request", err, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: err.Error(),
		})
		return
	}
	if metricType == "" {
		h.logger.Error("Bad Request", nil, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "Metrictype is required",
		})
		return
	}
	if instanceIndex != "" {
		parameters.Add("instanceindex", instanceIndex)
	}

	path, _ := routes.MetricsCollectorRoutes().Get(routes.GetMetricHistoriesRouteName).URLPath("appid", appId, "metrictype", metricType)

	url := h.conf.MetricsCollector.MetricsCollectorUrl + path.RequestURI() + "?" + parameters.Encode()

	resp, err := h.metricsCollectorClient.Get(url)
	if err != nil {
		h.logger.Error("Failed to retrieve metrics history from metricscollector", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error retrieving metrics history from metricscollector"})
		return
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Error occured during parsing metrics histories result", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error parsing metric history from metricscollector"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Error occured during getting metric histories", nil, lager.Data{"statusCode": resp.StatusCode, "body": string(responseData)})
		handlers.WriteJSONResponse(w, resp.StatusCode, models.ErrorResponse{
			Code:    http.StatusText(resp.StatusCode),
			Message: string(responseData)})
		return
	}
	paginatedResponse, err := paginateResource(responseData, parameters, r)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: err.Error()})
		return
	}

	handlers.WriteJSONResponse(w, resp.StatusCode, paginatedResponse)
}

func (h *PublicApiHandler) GetApiInfo(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	info, err := ioutil.ReadFile(h.conf.InfoFilePath)
	if err != nil {
		h.logger.Error("Failed to info file", err, lager.Data{"info-file-path": h.conf.InfoFilePath})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to load info"})
		return
	}
	w.Write([]byte(info))
}

func (h *PublicApiHandler) GetHealth(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	w.Write([]byte(`{"alive":"true"}`))
}

func (h *PublicApiHandler) CreateCredential(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "AppId is required",
		})
		return
	}
	var userProvidedCredential *models.Credential
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read user provided credential request body", err, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error creating credential"})
		return
	}
	if len(bodyBytes) > 0 {
		userProvidedCredential = &models.Credential{}
		err = json.Unmarshal(bodyBytes, userProvidedCredential)
		if err != nil {
			h.logger.Error("Failed to unmarshal user provided credential", err, lager.Data{"appId": appId, "body": bodyBytes})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad Request",
				Message: "Invalid credential format"})
			return
		}
		if !(userProvidedCredential.Username != "" && userProvidedCredential.Password != "") {
			h.logger.Info("Username or password is missing", lager.Data{"appId": appId, "userProvidedCredential": userProvidedCredential})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad Request",
				Message: "Username and password are both required",
			})
			return
		}
	}

	h.logger.Info("Create credential", lager.Data{"appId": appId})
	cred, err := custom_metrics_cred_helper.CreateCredential(appId, userProvidedCredential, h.policydb, custom_metrics_cred_helper.MaxRetry)
	if err != nil {
		h.logger.Error("Failed to create credential", err, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error creating credential"})
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
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "AppId is required",
		})
		return
	}

	h.logger.Info("Delete credential", lager.Data{"appId": appId})
	err := custom_metrics_cred_helper.DeleteCredential(appId, h.policydb, custom_metrics_cred_helper.MaxRetry)
	if err != nil {
		h.logger.Error("Failed to delete credential", err, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error deleting credential"})
		return
	}
	handlers.WriteJSONResponse(w, http.StatusOK, nil)

}