package publicapiserver

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/policyvalidator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/schedulerutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	uuid "github.com/nu7hatch/gouuid"
)

type PublicApiHandler struct {
	logger                 lager.Logger
	conf                   *config.Config
	policydb               db.PolicyDB
	bindingdb              db.BindingDB
	scalingEngineClient    *http.Client
	metricsCollectorClient *http.Client
	eventGeneratorClient   *http.Client
	policyValidator        *policyvalidator.PolicyValidator
	schedulerUtil          *schedulerutil.SchedulerUtil
	credentials            cred_helper.Credentials
}

func NewPublicApiHandler(logger lager.Logger, conf *config.Config, policydb db.PolicyDB, bindingdb db.BindingDB, credentials cred_helper.Credentials) *PublicApiHandler {
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
		bindingdb:              bindingdb,
		scalingEngineClient:    seClient,
		metricsCollectorClient: mcClient,
		eventGeneratorClient:   egClient,
		policyValidator:        policyvalidator.NewPolicyValidator(conf.PolicySchemaPath, conf.ScalingRules.CPU.LowerThreshold, conf.ScalingRules.CPU.UpperThreshold),
		schedulerUtil:          schedulerutil.NewSchedulerUtil(conf, logger),
		credentials:            credentials,
	}
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	handlers.WriteJSONResponse(w, statusCode, models.ErrorResponse{
		Code:    http.StatusText(statusCode),
		Message: message})
}

func (h *PublicApiHandler) GetScalingPolicy(w http.ResponseWriter, _ *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		writeErrorResponse(w, http.StatusBadRequest, "AppId is required")
		return
	}

	h.logger.Info("Get Scaling Policy", lager.Data{"appId": appId})

	scalingPolicy, err := h.policydb.GetAppPolicy(appId)
	if err != nil {
		h.logger.Error("Failed to retrieve scaling policy from database", err, lager.Data{"appId": appId, "err": err})
		writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving scaling policy")
		return
	}

	if scalingPolicy == nil {
		h.logger.Info("policy doesn't exist", lager.Data{"appId": appId})
		writeErrorResponse(w, http.StatusNotFound, "Policy Not Found")
		return
	}

	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err = jsonEncoder.Encode(scalingPolicy)
	if err != nil {
		h.logger.Error("Failed to json encode scaling policy", err, lager.Data{"appId": appId, "policy": scalingPolicy})
		writeErrorResponse(w, http.StatusInternalServerError, "Error encode scaling policy")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(bf.Bytes())))
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(bf.Bytes())
	if err != nil {
		h.logger.Error("failed-to-write-body", err)
	}
}

func (h *PublicApiHandler) AttachScalingPolicy(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		writeErrorResponse(w, http.StatusBadRequest, "AppId is required")
		return
	}

	h.logger.Info("Attach Scaling Policy", lager.Data{"appId": appId})

	policyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", err, lager.Data{"appId": appId})
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to read request body")
		return
	}

	policy := string(policyBytes)

	errResults, valid, policy := h.policyValidator.ValidatePolicy(policy)
	if !valid {
		h.logger.Error("Failed to validate policy", nil, lager.Data{"errResults": errResults})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, errResults)
		return
	}

	policyGuid, err := uuid.NewV4()
	if err != nil {
		h.logger.Error("Failed to generate policy guid", err, nil)
		writeErrorResponse(w, http.StatusInternalServerError, "Error generating policy guid")
		return
	}

	h.logger.Info("saving policy json", lager.Data{"policy": policy})
	err = h.policydb.SaveAppPolicy(appId, policy, policyGuid.String())
	if err != nil {
		h.logger.Error("Failed to save policy", err, nil)
		writeErrorResponse(w, http.StatusInternalServerError, "Error saving policy")
		return
	}

	h.logger.Info("creating/updating schedules", lager.Data{"policy": policy})
	err = h.schedulerUtil.CreateOrUpdateSchedule(appId, policy, policyGuid.String())
	if err != nil {
		h.logger.Error("Failed to create/update schedule", err, nil)
	}
	w.WriteHeader(http.StatusOK)
	_, err = io.WriteString(w, policy)
	if err != nil {
		h.logger.Error("failed-to-write-body", err)
	}
}

func (h *PublicApiHandler) DetachScalingPolicy(w http.ResponseWriter, _ *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		writeErrorResponse(w, http.StatusBadRequest, "AppId is required")
		return
	}

	h.logger.Info("Deleting policy json", lager.Data{"appId": appId})
	err := h.policydb.DeletePolicy(appId)
	if err != nil {
		h.logger.Error("Failed to delete policy from database", err, nil)
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting policy")
		return
	}
	h.logger.Info("Deleting schedules", lager.Data{"appId": appId})
	err = h.schedulerUtil.DeleteSchedule(appId)
	if err != nil {
		h.logger.Error("Failed to delete schedule", err, nil)
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting schedules")
		return
	}

	if h.bindingdb != nil && !reflect.ValueOf(h.bindingdb).IsNil() {
		// brokered offering: check if there's a default policy that could apply
		serviceinstance, err := h.bindingdb.GetServiceInstanceByAppId(appId)
		if err != nil {
			h.logger.Error("Failed to find service instance for app", err, lager.Data{"appId": appId})
			writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving service instance")
			return
		}
		if serviceinstance.DefaultPolicy != "" {
			policyStr := serviceinstance.DefaultPolicy
			policyGuidStr := serviceinstance.DefaultPolicyGuid
			h.logger.Info("saving default policy json for app", lager.Data{"policy": policyStr})
			err = h.policydb.SaveAppPolicy(appId, policyStr, policyGuidStr)
			if err != nil {
				h.logger.Error("failed to save policy", err, lager.Data{"appId": appId, "policy": policyStr})
				writeErrorResponse(w, http.StatusInternalServerError, "Error attaching the default policy")
				return
			}

			h.logger.Info("creating/updating schedules", lager.Data{"policy": policyStr})
			err = h.schedulerUtil.CreateOrUpdateSchedule(appId, policyStr, policyGuidStr)
			//while there is synchronization between policy and schedule, so creating schedule error does not break
			//the whole creating binding process
			if err != nil {
				h.logger.Error("failed to create/update schedules", err, lager.Data{"policy": policyStr})
			}
		}
	}
	// find via the app id the binding -> service instance
	// default policy? then apply that

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("{}"))
	if err != nil {
		h.logger.Error("failed-to-write-body", err)
	}
}

func (h *PublicApiHandler) GetScalingHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]

	h.logger.Info("Get ScalingHistories", lager.Data{"appId": appId})

	parameters, err := parseParameter(r, vars)
	if err != nil {
		h.logger.Error("Bad Request", err, lager.Data{"appId": appId})
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	path, _ := routes.ScalingEngineRoutes().Get(routes.GetScalingHistoriesRouteName).URLPath("appid", appId)

	url := h.conf.ScalingEngine.ScalingEngineUrl + path.RequestURI() + "?" + parameters.Encode()

	resp, err := h.scalingEngineClient.Get(url)
	if err != nil {
		h.logger.Error("Failed to retrieve scaling history from scaling engine", err, lager.Data{"url": url})
		writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving scaling history from scaling engine")
		return
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Error occurred during parsing scaling histories result", err, lager.Data{"url": url})
		writeErrorResponse(w, http.StatusInternalServerError, "Error parsing scaling history from scaling engine")
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Error occurred during getting scaling histories", nil, lager.Data{"statusCode": resp.StatusCode, "body": string(responseData)})
		writeErrorResponse(w, resp.StatusCode, string(responseData))
		return
	}
	paginatedResponse, err := paginateResource(responseData, parameters, r)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, err.Error())
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
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if metricType == "" {
		h.logger.Error("Bad Request", nil, lager.Data{"appId": appId})
		writeErrorResponse(w, http.StatusBadRequest, "Metrictype is required")
		return
	}

	path, _ := routes.EventGeneratorRoutes().Get(routes.GetAggregatedMetricHistoriesRouteName).URLPath("appid", appId, "metrictype", metricType)

	url := h.conf.EventGenerator.EventGeneratorUrl + path.RequestURI() + "?" + parameters.Encode()

	resp, err := h.eventGeneratorClient.Get(url)
	if err != nil {
		h.logger.Error("Failed to retrieve metrics history from eventgenerator", err, lager.Data{"url": url})
		writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving metrics history from eventgenerator")
		return
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Error occurred during parsing metrics histories result", err, lager.Data{"url": url})
		writeErrorResponse(w, http.StatusInternalServerError, "Error parsing metric history from eventgenerator")
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Error occurred during getting metric histories", nil, lager.Data{"statusCode": resp.StatusCode, "body": string(responseData)})
		writeErrorResponse(w, resp.StatusCode, string(responseData))
		return
	}
	paginatedResponse, err := paginateResource(responseData, parameters, r)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
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
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if metricType == "" {
		h.logger.Error("Bad Request", nil, lager.Data{"appId": appId})
		writeErrorResponse(w, http.StatusBadRequest, "Metrictype is required")
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
		writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving metrics history from metricscollector")
		return
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Error occurred during parsing metrics histories result", err, lager.Data{"url": url})
		writeErrorResponse(w, http.StatusInternalServerError, "Error parsing metric history from metricscollector")
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Error occurred during getting metric histories", nil, lager.Data{"statusCode": resp.StatusCode, "body": string(responseData)})
		writeErrorResponse(w, resp.StatusCode, string(responseData))
		return
	}
	paginatedResponse, err := paginateResource(responseData, parameters, r)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	handlers.WriteJSONResponse(w, resp.StatusCode, paginatedResponse)
}

func (h *PublicApiHandler) GetApiInfo(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
	info, err := ioutil.ReadFile(h.conf.InfoFilePath)
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
	var userProvidedCredential *models.Credential
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read user provided credential request body", err, lager.Data{"appId": appId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error creating credential")
		return
	}
	if len(bodyBytes) > 0 {
		userProvidedCredential = &models.Credential{}
		err = json.Unmarshal(bodyBytes, userProvidedCredential)
		if err != nil {
			h.logger.Error("Failed to unmarshal user provided credential", err, lager.Data{"appId": appId, "body": bodyBytes})
			writeErrorResponse(w, http.StatusBadRequest, "Invalid credential format")
			return
		}
		if !(userProvidedCredential.Username != "" && userProvidedCredential.Password != "") {
			h.logger.Info("Username or password is missing", lager.Data{"appId": appId, "userProvidedCredential": userProvidedCredential})
			writeErrorResponse(w, http.StatusBadRequest, "Username and password are both required")
			return
		}
	}

	h.logger.Info("Create credential", lager.Data{"appId": appId})
	cred, err := h.credentials.Create(appId, userProvidedCredential)
	if err != nil {
		h.logger.Error("Failed to create credential", err, lager.Data{"appId": appId})
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

func (h *PublicApiHandler) DeleteCredential(w http.ResponseWriter, _ *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error("AppId is missing", nil, nil)
		writeErrorResponse(w, http.StatusBadRequest, "AppId is required")
		return
	}

	h.logger.Info("Delete credential", lager.Data{"appId": appId})
	err := h.credentials.Delete(appId)
	if err != nil {
		h.logger.Error("Failed to delete credential", err, lager.Data{"appId": appId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting credential")
		return
	}
	handlers.WriteJSONResponse(w, http.StatusOK, nil)
}
