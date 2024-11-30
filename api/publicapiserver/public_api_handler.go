package publicapiserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/policyvalidator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/schedulerclient"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"github.com/google/uuid"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	"code.cloudfoundry.org/lager/v3"
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
		policyValidator: policyvalidator.NewPolicyValidator(
			conf.PolicySchemaPath,
			conf.ScalingRules.CPU.LowerThreshold,
			conf.ScalingRules.CPU.UpperThreshold,
			conf.ScalingRules.CPUUtil.LowerThreshold,
			conf.ScalingRules.CPUUtil.UpperThreshold,
			conf.ScalingRules.DiskUtil.LowerThreshold,
			conf.ScalingRules.DiskUtil.UpperThreshold,
			conf.ScalingRules.Disk.LowerThreshold,
			conf.ScalingRules.Disk.UpperThreshold,
		),
		schedulerUtil: schedulerclient.New(conf, logger),
		credentials:   credentials,
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

	handlers.WriteJSONResponse(w, http.StatusOK, scalingPolicy)
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

	policy, errResults := h.policyValidator.ValidatePolicy(policyBytes)
	if errResults != nil {
		logger.Info("Failed to validate policy", lager.Data{"errResults": errResults})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, errResults)
		return
	}

	policyGuid := uuid.NewString()
	err = h.policydb.SaveAppPolicy(r.Context(), appId, policy, policyGuid)
	if err != nil {
		logger.Error("Failed to save policy", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error saving policy")
		return
	}
	h.logger.Info("creating/updating schedules", lager.Data{"policy": policy})
	err = h.schedulerUtil.CreateOrUpdateSchedule(r.Context(), appId, policy, policyGuid)
	if err != nil {
		logger.Error("Failed to create/update schedule", err)
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	response, err := json.Marshal(policy)
	if err != nil {
		logger.Error("Failed to marshal policy", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Error marshaling policy")
		return
	}
	_, err = w.Write(response)
	if err != nil {
		logger.Error("Failed to write body", err)
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
			var policy *models.ScalingPolicy
			err := json.Unmarshal([]byte(policyStr), &policy)
			if err != nil {
				h.logger.Error("default policy invalid", err, lager.Data{"appId": appId, "policy": policyStr})
				writeErrorResponse(w, http.StatusInternalServerError, "Default policy not valid")
				return
			}

			err = h.policydb.SaveAppPolicy(r.Context(), appId, policy, policyGuidStr)
			if err != nil {
				logger.Error("failed to save policy", err, lager.Data{"policy": policyStr})
				writeErrorResponse(w, http.StatusInternalServerError, "Error attaching the default policy")
				return
			}

			logger.Info("creating/updating schedules", lager.Data{"policy": policyStr})
			err = h.schedulerUtil.CreateOrUpdateSchedule(r.Context(), appId, policy, policyGuidStr)
			//while there is synchronization between policy and schedule, so creating schedule error does not break
			//the whole creating binding process
			if err != nil {
				logger.Error("failed to create/update schedules", err, lager.Data{"policy": policyStr})
				writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update schedule:%s", err))
			}
		}
	}
	// find via the app id the binding -> service instance
	// default policy? then apply that

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("{}"))
	if err != nil {
		logger.Error(ActionWriteBody, err)
	}
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
		r := routes.NewRouter()
		router := r.CreateEventGeneratorRoutes()
		if router == nil {
			panic("Failed to create event generator routes")
		}

		route := router.Get(routes.GetAggregatedMetricHistoriesRouteName)
		path, err := route.URLPath("appid", appId, "metrictype", metricType)
		if err != nil {
			logger.Error("Failed to create path", err)
			panic(err)
		}
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
		h.logger.Error(ActionWriteBody, err)
	}
}

func (h *PublicApiHandler) GetHealth(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
	_, err := w.Write([]byte(`{"alive":"true"}`))
	if err != nil {
		h.logger.Error(ActionWriteBody, err)
	}
}
