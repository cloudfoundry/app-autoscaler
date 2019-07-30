package brokerserver

import (
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

func (h *BrokerHandler) GetBrokerCatalog(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	catalog, err := ioutil.ReadFile(h.conf.CatalogPath)
	if err != nil {
		h.logger.Error("failed to read request body when get catalog", err)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to load catalog"})
		return
	}
	w.Write([]byte(catalog))
}

func (h *BrokerHandler) CreateServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]

	body := &models.InstanceCreationRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		h.logger.Error("failed to create service instance when trying to read request body", err)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to read request body"})
		return
	}

	if instanceId == "" || body.OrgGUID == "" || body.SpaceGUID == "" || body.ServiceID == "" || body.PlanID == "" {
		h.logger.Error("failed to create service instance when trying to get mandatory data", nil, lager.Data{"instanceId": instanceId, "orgGuid": body.OrgGUID, "spaceGuid": body.SpaceGUID, "serviceId": body.ServiceID, "planId": body.PlanID})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "Malformed or missing mandatory data",
		})
		return
	}

	err = h.bindingdb.CreateServiceInstance(instanceId, body.OrgGUID, body.SpaceGUID)
	if err != nil {
		if err == db.ErrAlreadyExists {
			h.logger.Error("failed to create service instance: service instance already exists", err, lager.Data{"instanaceId": instanceId, "orgGuid": body.OrgGUID, "spaceGuid": body.SpaceGUID})
			w.Write(nil)
			return
		}
		h.logger.Error("failed to create service instance", err, lager.Data{"instanaceId": instanceId, "orgGuid": body.OrgGUID, "spaceGuid": body.SpaceGUID})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error creating service instance"})
		return
	}

	if h.conf.DashboardRedirectURI == "" {
		w.WriteHeader(http.StatusCreated)
		w.Write(nil)
	} else {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf("{\"dashboard_url\":\"%s\"}", GetDashboardURL(h.conf, instanceId))))
	}
}

func (h *BrokerHandler) DeleteServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]

	body := &models.BrokerCommonRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		h.logger.Error("failed to delete service instance when trying to read request body", err)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to read request body"})
		return
	}

	if instanceId == "" || body.ServiceID == "" || body.PlanID == "" {
		h.logger.Error("failed to delete service instance when trying to get mandatory data", nil,
			lager.Data{"instanceId": instanceId, "serviceId": body.ServiceID, "planId": body.PlanID})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "Malformed or missing mandatory data",
		})
		return
	}

	err = h.bindingdb.DeleteServiceInstance(instanceId)
	if err != nil {
		if err == db.ErrDoesNotExist {
			h.logger.Error("failed to delete service instance: service instance does not exist", err,
				lager.Data{"instanaceId": instanceId})
			handlers.WriteJSONResponse(w, http.StatusGone, models.ErrorResponse{
				Code:    "Gone",
				Message: "Service Instance Doesn't Exist"})
			return
		}
		h.logger.Error("failed to delete service instance", err, lager.Data{"instanaceId": instanceId})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error deleting service instance"})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

func (h *BrokerHandler) BindServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]
	bindingId := vars["bindingId"]

	body := &models.BindingRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		h.logger.Error("failed to create binding when trying to read request body", err)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to read request body"})
		return
	}

	if body.AppID == "" || instanceId == "" || bindingId == "" || body.ServiceID == "" || body.PlanID == "" {
		h.logger.Error("failed to create binding when trying to get mandatory data", nil, lager.Data{"appId": body.AppID, "instanceId": instanceId, "bindingId": bindingId, "serviceId": body.ServiceID, "planId": body.PlanID})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "Malformed or missing mandatory data",
		})
		return
	}

	err = h.bindingdb.CreateServiceBinding(bindingId, instanceId, body.AppID)
	if err != nil {
		if err == db.ErrAlreadyExists {
			h.logger.Error("failed to create binding: binding already exists", err, lager.Data{"appId": body.AppID})
			handlers.WriteJSONResponse(w, http.StatusConflict, models.ErrorResponse{
				Code:    "Conflict",
				Message: "An autoscaler service instance is already bound to application ${applicationId}. Multiple bindings are not supported."})
			return
		}
		h.logger.Error("failed to save binding", err, lager.Data{"appId": body.AppID, "bindingId": bindingId, "instanceId": instanceId})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error creating service binding"})
		return
	}
	cred, err := custom_metrics_cred_helper.CreateCustomMetricsCredential(body.AppID, h.policydb, custom_metrics_cred_helper.MaxRetry)
	if err != nil {
		//revert binding creating
		h.logger.Error("failed to create custom metrics credential", err, lager.Data{"appId": body.AppID})
		err = h.bindingdb.DeleteServiceBindingByAppId(body.AppID)
		if err != nil {
			h.logger.Error("revert binding due to failed to create custom metrics credential", err, lager.Data{"appId": body.AppID})
		}
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error creating service binding"})
		return
	}
	if body.Policy == "" {
		h.logger.Info("no policy json provided", lager.Data{})
	} else {
		errResults, valid := h.policyValidator.ValidatePolicy(body.Policy)
		if !valid {
			h.logger.Error("failed to validate policy", err, lager.Data{"appId": body.AppID, "policy": body.Policy})
			//revert creating binding and custom metrics credential
			err = custom_metrics_cred_helper.DeleteCustomMetricsCredential(body.AppID, h.policydb, custom_metrics_cred_helper.MaxRetry)
			if err != nil {
				h.logger.Error("failed to revert custom metrics credential due to failed to validate policy", err, lager.Data{"appId": body.AppID})
			}
			err = h.bindingdb.DeleteServiceBindingByAppId(body.AppID)
			if err != nil {
				h.logger.Error("failed to revert binding due to failed to validate policy", err, lager.Data{"appId": body.AppID})
			}
			handlers.WriteJSONResponse(w, http.StatusBadRequest, errResults)
			return
		}
		policyGuid, err := uuid.NewV4()
		if err != nil {
			h.logger.Error("failed to create policy guid", err, lager.Data{"appId": body.AppID})
			//revert creating binding and custom metrics credential
			err = custom_metrics_cred_helper.DeleteCustomMetricsCredential(body.AppID, h.policydb, custom_metrics_cred_helper.MaxRetry)
			if err != nil {
				h.logger.Error("failed to revert custom metrics credential due to failed to create policy guid", err, lager.Data{"appId": body.AppID})
			}
			err = h.bindingdb.DeleteServiceBindingByAppId(body.AppID)
			if err != nil {
				h.logger.Error("failed to revert binding due to failed to create policy guid", err, lager.Data{"appId": body.AppID})
			}
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Interal-Server-Error",
				Message: "Error generating policy guid"})
			return
		}

		h.logger.Info("saving policy json", lager.Data{"policy": body.Policy})
		err = h.policydb.SaveAppPolicy(body.AppID, body.Policy, policyGuid.String())
		if err != nil {
			h.logger.Error("failed to save policy", err, lager.Data{"appId": body.AppID, "policy": body.Policy})
			//failed to save policy, so revert creating binding and custom metrics credential
			err = custom_metrics_cred_helper.DeleteCustomMetricsCredential(body.AppID, h.policydb, custom_metrics_cred_helper.MaxRetry)
			if err != nil {
				h.logger.Error("failed to revert custom metrics credential due to failed to save policy", err, lager.Data{"appId": body.AppID})
			}
			err = h.bindingdb.DeleteServiceBindingByAppId(body.AppID)
			if err != nil {
				h.logger.Error("failed to revert binding due to failed to save policy", err, lager.Data{"appId": body.AppID})
			}
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Interal-Server-Error",
				Message: "Error saving policy"})
			return
		}

		h.logger.Info("creating/updating schedules", lager.Data{"policy": body.Policy})
		err = h.schedulerUtil.CreateOrUpdateSchedule(body.AppID, body.Policy, policyGuid.String())
		//while there is synchronization between policy and schedule, so creating schedule error does not break
		//the whole creating binding process
		if err != nil {
			h.logger.Error("failed to create/update schedules", err, lager.Data{"policy": body.Policy})
		}
	}
	handlers.WriteJSONResponse(w, http.StatusCreated, models.CredentialResponse{
		Credentials: models.Credentials{
			CustomMetrics: models.CustomMetrics{
				CustomMetricCredentials: cred,
				URL:                     h.conf.MetricsForwarder.MetricsForwarderUrl,
			},
		},
	})
}

func (h *BrokerHandler) UnbindServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]
	bindingId := vars["bindingId"]

	body := &models.UnbindingRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		h.logger.Error("failed to read request body:delete binding", err)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to read request body"})
		return
	}

	if instanceId == "" || bindingId == "" || body.ServiceID == "" || body.PlanID == "" {
		h.logger.Error("failed to delete binding when trying to get mandatory data", nil, lager.Data{"appId": body.AppID, "instanceId": instanceId, "bindingId": bindingId, "serviceId": body.ServiceID, "planId": body.PlanID})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "Malformed or missing mandatory data",
		})
		return
	}

	h.logger.Info("deleting policy json", lager.Data{"appId": body.AppID})
	err = h.policydb.DeletePolicy(body.AppID)
	if err != nil {
		h.logger.Error("failed to delete policy for unbinding", err, lager.Data{"appId": body.AppID})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error deleting policy"})
		return
	}
	h.logger.Info("deleting schedules", lager.Data{"appId": body.AppID})
	err = h.schedulerUtil.DeleteSchedule(body.AppID)
	if err != nil {
		h.logger.Info("failed to delete schedules for unbinding", lager.Data{"appId": body.AppID})
	}
	err = h.bindingdb.DeleteServiceBinding(bindingId)
	if err != nil {
		h.logger.Error("failed to delete binding", err, lager.Data{"bindingId": bindingId, "appId": body.AppID})
		if err == db.ErrDoesNotExist {
			handlers.WriteJSONResponse(w, http.StatusGone, models.ErrorResponse{
				Code:    "Gone",
				Message: "Service Binding Doesn't Exist"})
			return
		}
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error deleting service binding"})
		return
	}
	err = custom_metrics_cred_helper.DeleteCustomMetricsCredential(body.AppID, h.policydb, custom_metrics_cred_helper.MaxRetry)
	if err != nil {
		h.logger.Error("failed to delete custom metrics credential for unbinding", err, lager.Data{"appId": body.AppID})
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
