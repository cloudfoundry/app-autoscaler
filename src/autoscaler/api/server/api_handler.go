package server

import (
	"autoscaler/api/config"
	"autoscaler/api/policyvalidator"
	"autoscaler/api/schedulerutil"
	"autoscaler/db"
	"autoscaler/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	uuid "github.com/nu7hatch/gouuid"
)

type ApiHandler struct {
	logger          lager.Logger
	conf            *config.Config
	bindingdb       db.BindingDB
	policydb        db.PolicyDB
	policyValidator *policyvalidator.PolicyValidator
	schedulerUtil   *schedulerutil.SchedulerUtil
}

func NewApiHandler(logger lager.Logger, conf *config.Config, bindingdb db.BindingDB, policydb db.PolicyDB) *ApiHandler {

	return &ApiHandler{
		logger:          logger,
		conf:            conf,
		bindingdb:       bindingdb,
		policydb:        policydb,
		policyValidator: policyvalidator.NewPolicyValidator(conf.PolicySchemaPath),
		schedulerUtil:   schedulerutil.NewSchedulerUtil(conf, logger),
	}
}

func (h *ApiHandler) GetBrokerCatalog(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	catalog, err := ioutil.ReadFile(h.conf.CatalogPath)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to load catalog"})
		return
	}
	w.Write([]byte(catalog))
}

func (h *ApiHandler) CreateServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]

	body := &models.InstanceCreationRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to read request body"})
		return
	}

	if instanceId == "" || body.OrgGUID == "" || body.SpaceGUID == "" || body.ServiceID == "" || body.PlanID == "" {
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "Malformed or missing mandatory data",
		})
		return
	}

	err = h.bindingdb.CreateServiceInstance(instanceId, body.OrgGUID, body.SpaceGUID)
	if err != nil {
		if err == db.ErrAlreadyExists {
			w.Write(nil)
			return
		}
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

func (h *ApiHandler) DeleteServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]

	body := &models.BrokerCommonRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to read request body"})
		return
	}

	if instanceId == "" || body.ServiceID == "" || body.PlanID == "" {
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "Malformed or missing mandatory data",
		})
		return
	}

	err = h.bindingdb.DeleteServiceInstance(instanceId)
	if err != nil {
		if err == db.ErrDoesNotExist {
			handlers.WriteJSONResponse(w, http.StatusGone, models.ErrorResponse{
				Code:    "Gone",
				Message: "Service Instance Doesn't Exist"})
			return
		}

		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error deleting service instance"})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

func (h *ApiHandler) BindServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]
	bindingId := vars["bindingId"]

	body := &models.BindingRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to read request body"})
		return
	}

	if body.AppID == "" || instanceId == "" || bindingId == "" || body.ServiceID == "" || body.PlanID == "" {
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "Malformed or missing mandatory data",
		})
		return
	}

	if body.Policy == "" {
		h.logger.Info("no policy json provided", lager.Data{})
	} else {
		errResults, valid := h.policyValidator.ValidatePolicy(body.Policy)

		if !valid {
			handlers.WriteJSONResponse(w, http.StatusBadRequest, errResults)
			return
		}

		policyGuid, err := uuid.NewV4()
		if err != nil {
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Interal-Server-Error",
				Message: "Error generating policy guid"})
			return
		}

		h.logger.Info("saving policy json", lager.Data{"policy": body.Policy})
		err = h.policydb.SaveAppPolicy(body.AppID, body.Policy, policyGuid.String())
		if err != nil {
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Interal-Server-Error",
				Message: "Error saving policy"})
			return
		}

		h.logger.Info("creating/updating schedules", lager.Data{"policy": body.Policy})
		err = h.schedulerUtil.CreateOrUpdateSchedule(body.AppID, body.Policy, policyGuid.String())
		if err != nil {
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Interal-Server-Error",
				Message: "Error creating/updating schedules"})
			return
		}
	}

	err = h.bindingdb.CreateServiceBinding(bindingId, instanceId, body.AppID)
	if err != nil {
		if err == db.ErrAlreadyExists {
			w.Write(nil)
			return
		}
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error creating service binding"})
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(nil)
}

func (h *ApiHandler) UnbindServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]
	bindingId := vars["bindingId"]

	body := &models.UnbindingRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to read request body"})
		return
	}

	if instanceId == "" || bindingId == "" || body.ServiceID == "" || body.PlanID == "" {
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "Malformed or missing mandatory data",
		})
		return
	}

	h.logger.Info("deleting policy json", lager.Data{"appId": body.AppID})
	err = h.policydb.DeletePolicy(body.AppID)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error deleting policy"})
		return
	}

	err = h.bindingdb.DeleteServiceBinding(bindingId)
	if err != nil {
		if err == db.ErrDoesNotExist {
			handlers.WriteJSONResponse(w, http.StatusGone, models.ErrorResponse{
				Code:    "Gone",
				Message: "Service Binding Doesn't Exist"})
			return
		}
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error creating service binding"})
		return
	}

	h.logger.Info("deleting schedules", lager.Data{"appId": body.AppID})
	err = h.schedulerUtil.DeleteSchedule(body.AppID)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error deleting schedules"})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
