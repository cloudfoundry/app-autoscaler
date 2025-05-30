package publicapiserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/policyvalidator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/schedulerclient"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/lager/v3"
	"github.com/google/uuid"
)

type PublicApiHandler struct {
	logger               lager.Logger
	conf                 *config.Config
	policydb             db.PolicyDB
	bindingdb            db.BindingDB
	eventGeneratorClient *http.Client
	policyValidator      *policyvalidator.PolicyValidator
	schedulerUtil        *schedulerclient.Client
	credentials          cred_helper.Credentials
}

const (
	ActionWriteBody             = "write-body"
	ActionCheckAppId            = "check-for-id-appid"
	ErrorMessageAppidIsRequired = "AppId is required"
)

var ErrInvalidConfigurations = errors.New("invalid binding configurations provided")

func NewPublicApiHandler(logger lager.Logger, conf *config.Config, policydb db.PolicyDB, bindingdb db.BindingDB, credentials cred_helper.Credentials) *PublicApiHandler {
	egClient, err := helpers.CreateHTTPSClient(&conf.EventGenerator.TLSClientCerts, helpers.DefaultClientConfig(), logger.Session("event_client"))
	if err != nil {
		logger.Error("Failed to create http client for EventGenerator", err, lager.Data{"eventgenerator": conf.EventGenerator.TLSClientCerts})
		os.Exit(1)
	}

	return &PublicApiHandler{
		logger:               logger,
		conf:                 conf,
		policydb:             policydb,
		bindingdb:            bindingdb,
		eventGeneratorClient: egClient,
		policyValidator:      createPolicyValidator(conf),
		schedulerUtil:        schedulerclient.New(conf, logger),
		credentials:          credentials,
	}
}

func createPolicyValidator(conf *config.Config) *policyvalidator.PolicyValidator {
	return policyvalidator.NewPolicyValidator(
		conf.PolicySchemaPath,
		conf.ScalingRules.CPU.LowerThreshold,
		conf.ScalingRules.CPU.UpperThreshold,
		conf.ScalingRules.CPUUtil.LowerThreshold,
		conf.ScalingRules.CPUUtil.UpperThreshold,
		conf.ScalingRules.DiskUtil.LowerThreshold,
		conf.ScalingRules.DiskUtil.UpperThreshold,
		conf.ScalingRules.Disk.LowerThreshold,
		conf.ScalingRules.Disk.UpperThreshold,
	)
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	handlers.WriteJSONResponse(w, statusCode, models.ErrorResponse{
		Code:    http.StatusText(statusCode),
		Message: message})
}

func (h *PublicApiHandler) GetScalingPolicy(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error(ActionCheckAppId, errors.New(ErrorMessageAppidIsRequired), nil)
		writeErrorResponse(w, http.StatusBadRequest, ErrorMessageAppidIsRequired)
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
	customMetricStrategy, err := h.bindingdb.GetCustomMetricStrategyByAppId(r.Context(), appId)
	if err != nil {
		logger.Error("Failed to retrieve customMetricStrategy from database", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving binding policy")
		return
	}
	scalingPolicyWithCustomMetricStrategy, err := models.GetBindingConfigAndPolicy(scalingPolicy, customMetricStrategy)
	if err != nil {
		logger.Error("Failed to build policy and config response object", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving binding policy")
		return
	}
	handlers.WriteJSONResponse(w, http.StatusOK, scalingPolicyWithCustomMetricStrategy)
}

func (h *PublicApiHandler) AttachScalingPolicy(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error(ActionCheckAppId, errors.New(ErrorMessageAppidIsRequired), nil)
		writeErrorResponse(w, http.StatusBadRequest, ErrorMessageAppidIsRequired)
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

	policy, errResults := h.policyValidator.ParseAndValidatePolicy(policyBytes)
	if errResults != nil {
		logger.Info("Failed to validate policy", lager.Data{"errResults": errResults, "policy": string(policyBytes)})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, errResults)
		return
	}

	bindingConfiguration, err := h.getBindingConfigurationFromRequest(policyBytes, logger)
	if err != nil {
		errMessage := "Failed to read binding configuration request body"
		logger.Error(errMessage, err)
		writeErrorResponse(w, http.StatusInternalServerError, errMessage)
		return
	}

	policyGuid := uuid.NewString()
	if err := h.policydb.SaveAppPolicy(r.Context(), appId, policy, policyGuid); err != nil {
		logger.Error("Failed to save policy", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error saving policy")
		return
	}

	h.logger.Info("creating/updating schedules", lager.Data{"policy": policy})

	if err := h.schedulerUtil.CreateOrUpdateSchedule(r.Context(), appId, policy, policyGuid); err != nil {
		logger.Error("Failed to create/update schedule", err)
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	validatedBindingConfiguration, err := bindingConfiguration.ValidateOrGetDefaultCustomMetricsStrategy()
	if err != nil {
		logger.Error(ErrInvalidConfigurations.Error(), err)
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	customMetricStrategy := validatedBindingConfiguration.GetCustomMetricsStrategy()
	logger.Info("saving custom metric submission strategy", lager.Data{"customMetricStrategy": customMetricStrategy, "appId": appId})
	err = h.bindingdb.SetOrUpdateCustomMetricStrategy(r.Context(), appId, validatedBindingConfiguration.GetCustomMetricsStrategy(), "update")
	if err != nil {
		actionName := "failed to save custom metric submission strategy in the database"
		logger.Error(actionName, err)
		writeErrorResponse(w, http.StatusInternalServerError, actionName)
		return
	}
	responseJson, err := buildResponse(policy, customMetricStrategy, logger)
	if err != nil {
		logger.Error("Failed to to build response", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error building response")
		return
	}
	_, err = w.Write(responseJson)
	if err != nil {
		h.logger.Error("Failed to write body", err)
	}
}

func (h *PublicApiHandler) DetachScalingPolicy(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	if appId == "" {
		h.logger.Error(ActionCheckAppId, errors.New(ErrorMessageAppidIsRequired), nil)
		writeErrorResponse(w, http.StatusBadRequest, ErrorMessageAppidIsRequired)
		return
	}

	logger := h.logger.Session("DetachScalingPolicy", lager.Data{"appId": appId})
	logger.Info("Deleting policy json", lager.Data{"appId": appId})

	if err := h.policydb.DeletePolicy(r.Context(), appId); err != nil {
		logger.Error("Failed to delete policy from database", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting policy")
		return
	}

	logger.Info("Deleting schedules")
	if err := h.schedulerUtil.DeleteSchedule(r.Context(), appId); err != nil {
		logger.Error("Failed to delete schedule", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting schedules")
		return
	}

	if err := h.resetDefaultPolicy(w, r, logger, appId); err != nil {
		return
	}

	if err := h.bindingdb.SetOrUpdateCustomMetricStrategy(r.Context(), appId, "", "delete"); err != nil {
		actionName := "failed to delete custom metric submission strategy in the database"
		logger.Error(actionName, err)
		writeErrorResponse(w, http.StatusInternalServerError, actionName)
		return
	}

	// find via the app id the binding -> service instance
	// default policy? then apply that
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("{}"))
	if err != nil {
		logger.Error(ActionWriteBody, err)
	}
}

func (h *PublicApiHandler) resetDefaultPolicy(w http.ResponseWriter, r *http.Request, logger lager.Logger, appId string) error {
	serviceInstance, err := h.bindingdb.GetServiceInstanceByAppId(appId)
	if err != nil {
		logger.Error("Failed to find service instance for app", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error retrieving service instance")
		return errors.New("error retrieving service instance")
	}

	if serviceInstance != nil && serviceInstance.DefaultPolicy != "" {
		return h.saveDefaultPolicy(w, r, logger, appId, serviceInstance)
	}

	return nil
}

func (h *PublicApiHandler) saveDefaultPolicy(w http.ResponseWriter, r *http.Request, logger lager.Logger, appId string, serviceInstance *models.ServiceInstance) error {
	policyStr := serviceInstance.DefaultPolicy
	policyGuidStr := serviceInstance.DefaultPolicyGuid
	logger.Info("saving default policy json for app", lager.Data{"policy": policyStr})

	var policy *models.ScalingPolicy
	if err := json.Unmarshal([]byte(policyStr), &policy); err != nil {
		h.logger.Error("default policy invalid", err, lager.Data{"appId": appId, "policy": policyStr})
		writeErrorResponse(w, http.StatusInternalServerError, "Default policy not valid")
		return errors.New("default policy not valid")
	}

	if err := h.policydb.SaveAppPolicy(r.Context(), appId, policy, policyGuidStr); err != nil {
		logger.Error("failed to save policy", err, lager.Data{"policy": policyStr})
		writeErrorResponse(w, http.StatusInternalServerError, "Error attaching the default policy")
		return errors.New("error attaching the default policy")
	}

	logger.Info("creating/updating schedules", lager.Data{"policy": policyStr})
	if err := h.schedulerUtil.CreateOrUpdateSchedule(r.Context(), appId, policy, policyGuidStr); err != nil {
		logger.Error("failed to create/update schedules", err, lager.Data{"policy": policyStr})
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update schedule:%s", err))
		return errors.New("failed to update schedule")
	}

	return nil
}

func (h *PublicApiHandler) proxyRequest(logger lager.Logger, appId string, metricType string, w http.ResponseWriter, req *http.Request, parameters *url.Values, requestDescription string) {
	reqUrl := req.URL
	r := routes.NewRouter()
	router := r.CreateEventGeneratorSubrouter()

	if router == nil {
		panic("Failed to create event generator routes")
	}

	route := router.Get(routes.GetAggregatedMetricHistoriesRouteName)
	path, err := route.URLPath("appid", appId, "metrictype", metricType)
	if err != nil {
		logger.Error("Failed to create path", err)
		panic(err)
	}

	aUrl := h.conf.EventGenerator.EventGeneratorUrl + path.RequestURI() + "?" + parameters.Encode()
	req, _ = http.NewRequest("GET", aUrl, nil)

	resp, err := h.eventGeneratorClient.Do(req)
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

	h.proxyRequest(logger, appId, metricType, w, req, parameters, "metrics history from eventgenerator")
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
		h.logger.Error(ActionWriteBody, err)
	}
}

func (h *PublicApiHandler) GetHealth(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
	_, err := w.Write([]byte(`{"alive":"true"}`))
	if err != nil {
		h.logger.Error(ActionWriteBody, err)
	}
}

func (h *PublicApiHandler) getBindingConfigurationFromRequest(rawJson json.RawMessage, logger lager.Logger) (*models.BindingConfig, error) {
	bindingConfiguration := &models.BindingConfig{}
	var err error
	if rawJson != nil {
		err = json.Unmarshal(rawJson, &bindingConfiguration)
		if err != nil {
			logger.Error("unmarshal-binding-configuration", err)
			return bindingConfiguration, err
		}
	}
	return bindingConfiguration, err
}

func buildResponse(policy *models.ScalingPolicy, customMetricStrategy string, logger lager.Logger) ([]byte, error) {
	scalingPolicyWithCustomMetricStrategy, err := models.GetBindingConfigAndPolicy(policy, customMetricStrategy)
	if err != nil {
		logger.Error("Failed to build policy and config response object", err)
		return nil, errors.New("error: building binding and policy")
	}
	responseJson, err := json.Marshal(scalingPolicyWithCustomMetricStrategy)
	if err != nil {
		logger.Error("Failed to marshal policy", err)
		return nil, errors.New("error: marshalling policy")
	}
	return responseJson, nil
}
