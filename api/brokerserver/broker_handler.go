package brokerserver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"autoscaler/api/config"
	"autoscaler/api/custom_metrics_cred_helper"
	"autoscaler/api/policyvalidator"
	"autoscaler/api/schedulerutil"
	"autoscaler/db"
	"autoscaler/models"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	uuid "github.com/nu7hatch/gouuid"
)

type BrokerHandler struct {
	logger          lager.Logger
	conf            *config.Config
	bindingdb       db.BindingDB
	policydb        db.PolicyDB
	policyValidator *policyvalidator.PolicyValidator
	schedulerUtil   *schedulerutil.SchedulerUtil
}

func NewBrokerHandler(logger lager.Logger, conf *config.Config, bindingdb db.BindingDB, policydb db.PolicyDB) *BrokerHandler {

	return &BrokerHandler{
		logger:          logger,
		conf:            conf,
		bindingdb:       bindingdb,
		policydb:        policydb,
		policyValidator: policyvalidator.NewPolicyValidator(conf.PolicySchemaPath),
		schedulerUtil:   schedulerutil.NewSchedulerUtil(conf, logger),
	}

}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	handlers.WriteJSONResponse(w, statusCode, models.ErrorResponse{
		Code:    http.StatusText(statusCode),
		Message: message})
}

func (h *BrokerHandler) GetBrokerCatalog(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	catalog, err := ioutil.ReadFile(h.conf.CatalogPath)
	if err != nil {
		h.logger.Error("failed to read catalog file", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to load catalog")
		return
	}
	w.Write([]byte(catalog))
}

func (h *BrokerHandler) CreateServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]

	body := &models.InstanceCreationRequestBody{}
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read service provision request body", err, lager.Data{"instanceId": instanceId})
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to read request body")
		return
	}
	err = json.Unmarshal(bodyBytes, body)
	if err != nil {
		h.logger.Error("failed to unmarshal service provision body", err, lager.Data{"instanceId": instanceId, "body": string(bodyBytes)})
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body format")
		return
	}

	if instanceId == "" || body.OrgGUID == "" || body.SpaceGUID == "" || body.ServiceID == "" || body.PlanID == "" {
		h.logger.Error("failed to create service instance when trying to get mandatory data", nil, lager.Data{"instanceId": instanceId, "orgGuid": body.OrgGUID, "spaceGuid": body.SpaceGUID, "serviceId": body.ServiceID, "planId": body.PlanID})
		writeErrorResponse(w, http.StatusBadRequest, "Malformed or missing mandatory data")
		return
	}

	successResponse := func() {
		if h.conf.DashboardRedirectURI == "" {
			w.Write([]byte("{}"))
		} else {
			w.Write([]byte(fmt.Sprintf("{\"dashboard_url\":\"%s\"}", GetDashboardURL(h.conf, instanceId))))
		}
	}

	switch err := h.bindingdb.CreateServiceInstance(instanceId, body.OrgGUID, body.SpaceGUID); err {
	case nil:
		w.WriteHeader(http.StatusCreated)
		successResponse()
	case db.ErrAlreadyExists:
		h.logger.Error("failed to create service instance: service instance already exists", err, lager.Data{"instanceId": instanceId, "orgGuid": body.OrgGUID, "spaceGuid": body.SpaceGUID})
		successResponse()
	case db.ErrConflict:
		h.logger.Error("failed to create service instance: conflicting service instance exists", err, lager.Data{"instanceId": instanceId, "orgGuid": body.OrgGUID, "spaceGuid": body.SpaceGUID})
		writeErrorResponse(w, http.StatusConflict, fmt.Sprintf("Service instance with instance_id \"%s\" already exists with different parameters", instanceId))
	default:
		h.logger.Error("failed to create service instance", err, lager.Data{"instanceId": instanceId, "orgGuid": body.OrgGUID, "spaceGuid": body.SpaceGUID})
		writeErrorResponse(w, http.StatusInternalServerError, "Error creating service instance")
	}
}

func (h *BrokerHandler) DeleteServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]
	if instanceId == "" {
		h.logger.Error("failed to delete service instance when trying to get mandatory data", nil,
			lager.Data{"instanceId": instanceId})
		writeErrorResponse(w, http.StatusBadRequest, "Malformed or missing mandatory data")
		return
	}

	err := h.bindingdb.DeleteServiceInstance(instanceId)
	if err != nil {
		if err == db.ErrDoesNotExist {
			h.logger.Error("failed to delete service instance: service instance does not exist", err,
				lager.Data{"instanaceId": instanceId})
			writeErrorResponse(w, http.StatusGone, "Service Instance Doesn't Exist")
			return
		}
		h.logger.Error("failed to delete service instance", err, lager.Data{"instanaceId": instanceId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting service instance")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

func (h *BrokerHandler) BindServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]
	bindingId := vars["bindingId"]
	var policyGuid *uuid.UUID
	body := &models.BindingRequestBody{}
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read bind request body", err, lager.Data{"instanceId": instanceId, "bindingId": bindingId})
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to read request body")
		return
	}
	err = json.Unmarshal(bodyBytes, body)
	if err != nil {
		h.logger.Error("failed to unmarshal bind body", err, lager.Data{"instanceId": instanceId, "bindingId": bindingId, "body": string(bodyBytes)})
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body format")
		return
	}

	if body.AppID == "" || instanceId == "" || bindingId == "" || body.ServiceID == "" || body.PlanID == "" {
		h.logger.Error("failed to create binding when trying to get mandatory data", nil, lager.Data{"appId": body.AppID, "instanceId": instanceId, "bindingId": bindingId, "serviceId": body.ServiceID, "planId": body.PlanID})
		writeErrorResponse(w, http.StatusBadRequest, "Malformed or missing mandatory data")
		return
	}
	policyStr := string(body.Policy)
	if policyStr != "" {
		errResults, valid := h.policyValidator.ValidatePolicy(policyStr)
		if !valid {
			h.logger.Error("failed to validate policy", err, lager.Data{"appId": body.AppID, "policy": policyStr})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, errResults)
			return
		}
		policyGuid, err = uuid.NewV4()
		if err != nil {
			h.logger.Error("failed to create policy guid", err, lager.Data{"appId": body.AppID})
			writeErrorResponse(w, http.StatusInternalServerError, "Error generating policy guid")
			return
		}
	}
	err = h.bindingdb.CreateServiceBinding(bindingId, instanceId, body.AppID)
	if err != nil {
		if err == db.ErrAlreadyExists {
			h.logger.Error("failed to create binding: binding already exists", err, lager.Data{"appId": body.AppID})
			writeErrorResponse(w, http.StatusConflict, "An autoscaler service instance is already bound to the application. Multiple bindings are not supported.")
			return
		}
		h.logger.Error("failed to save binding", err, lager.Data{"appId": body.AppID, "bindingId": bindingId, "instanceId": instanceId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error creating service binding")
		return
	}
	cred, err := custom_metrics_cred_helper.CreateCredential(body.AppID, nil, h.policydb, custom_metrics_cred_helper.MaxRetry)
	if err != nil {
		//revert binding creating
		h.logger.Error("failed to create custom metrics credential", err, lager.Data{"appId": body.AppID})
		err = h.bindingdb.DeleteServiceBindingByAppId(body.AppID)
		if err != nil {
			h.logger.Error("failed to revert binding due to failed to create custom metrics credential", err, lager.Data{"appId": body.AppID})
		}
		writeErrorResponse(w, http.StatusInternalServerError, "Error creating service binding")
		return
	}
	if policyStr == "" {
		h.logger.Info("no policy json provided", lager.Data{})
	} else {
		h.logger.Info("saving policy json", lager.Data{"policy": policyStr})
		err = h.policydb.SaveAppPolicy(body.AppID, policyStr, policyGuid.String())
		if err != nil {
			h.logger.Error("failed to save policy", err, lager.Data{"appId": body.AppID, "policy": policyStr})
			//failed to save policy, so revert creating binding and custom metrics credential
			err = custom_metrics_cred_helper.DeleteCredential(body.AppID, h.policydb, custom_metrics_cred_helper.MaxRetry)
			if err != nil {
				h.logger.Error("failed to revert custom metrics credential due to failed to save policy", err, lager.Data{"appId": body.AppID})
			}
			err = h.bindingdb.DeleteServiceBindingByAppId(body.AppID)
			if err != nil {
				h.logger.Error("failed to revert binding due to failed to save policy", err, lager.Data{"appId": body.AppID})
			}
			writeErrorResponse(w, http.StatusInternalServerError, "Error saving policy")
			return
		}

		h.logger.Info("creating/updating schedules", lager.Data{"policy": policyStr})
		err = h.schedulerUtil.CreateOrUpdateSchedule(body.AppID, policyStr, policyGuid.String())
		//while there is synchronization between policy and schedule, so creating schedule error does not break
		//the whole creating binding process
		if err != nil {
			h.logger.Error("failed to create/update schedules", err, lager.Data{"policy": policyStr})
		}
	}
	handlers.WriteJSONResponse(w, http.StatusCreated, models.CredentialResponse{
		Credentials: models.Credentials{
			CustomMetrics: models.CustomMetricsCredentials{
				Credential: cred,
				URL:        h.conf.MetricsForwarder.MetricsForwarderUrl,
			},
		},
	})
}

func (h *BrokerHandler) UnbindServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]
	bindingId := vars["bindingId"]

	if instanceId == "" || bindingId == "" {
		h.logger.Error("failed to delete binding when trying to get mandatory data", nil, lager.Data{"instanceId": instanceId, "bindingId": bindingId})
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body format")
		return
	}
	appId, err := h.bindingdb.GetAppIdByBindingId(bindingId)
	if err == sql.ErrNoRows {
		h.logger.Info("binding does not exist", nil, lager.Data{"instanceId": instanceId, "bindingId": bindingId})
		writeErrorResponse(w, http.StatusGone, "Binding does not exist")
		return
	}
	if err != nil {
		h.logger.Error("failed to get appId by bindingId", err, lager.Data{"instanceId": instanceId, "bindingId": bindingId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting service binding")
		return
	}
	h.logger.Info("deleting policy json", lager.Data{"appId": appId})
	err = h.policydb.DeletePolicy(appId)
	if err != nil {
		h.logger.Error("failed to delete policy for unbinding", err, lager.Data{"appId": appId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting policy")
		return
	}
	h.logger.Info("deleting schedules", lager.Data{"appId": appId})
	err = h.schedulerUtil.DeleteSchedule(appId)
	if err != nil {
		h.logger.Info("failed to delete schedules for unbinding", lager.Data{"appId": appId})
	}
	err = h.bindingdb.DeleteServiceBinding(bindingId)
	if err != nil {
		h.logger.Error("failed to delete binding", err, lager.Data{"bindingId": bindingId, "appId": appId})
		if err == db.ErrDoesNotExist {
			writeErrorResponse(w, http.StatusGone, "Service Binding Doesn't Exist")
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting service binding")
		return
	}
	err = custom_metrics_cred_helper.DeleteCredential(appId, h.policydb, custom_metrics_cred_helper.MaxRetry)
	if err != nil {
		h.logger.Error("failed to delete custom metrics credential for unbinding", err, lager.Data{"appId": appId})
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
