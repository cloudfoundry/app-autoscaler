package brokerserver

import (
	"autoscaler/api/plancheck"
	"autoscaler/api/quota"
	"autoscaler/cf"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"autoscaler/api/config"
	"autoscaler/api/custom_metrics_cred_helper"
	"autoscaler/api/policyvalidator"
	"autoscaler/api/schedulerutil"
	"autoscaler/db"
	"autoscaler/models"
	"errors"

	"github.com/pivotal-cf/brokerapi/domain"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	uuid "github.com/nu7hatch/gouuid"
)

type BrokerHandler struct {
	logger                lager.Logger
	conf                  *config.Config
	bindingdb             db.BindingDB
	policydb              db.PolicyDB
	policyValidator       *policyvalidator.PolicyValidator
	schedulerUtil         *schedulerutil.SchedulerUtil
	quotaManagementClient *quota.Client
	catalog               []domain.Service
	planChecker           *plancheck.PlanChecker
	cfClient              cf.CFClient
}

var emptyJSONObject = regexp.MustCompile(`^\s*{\s*}\s*$`)
var errorBindingDoesNotExist = errors.New("Service binding does not exist")
var errorDeleteSchedulesForUnbinding = errors.New("Failed to delete schedules for unbinding")
var errorDeletePolicyForUnbinding = errors.New("Failed to delete policy for unbinding")
var errorDeleteServiceBinding = errors.New("Error deleting service binding")
var errorCredentialNotDeleted = errors.New("Failed to delete custom metrics credential for unbinding")

func NewBrokerHandler(logger lager.Logger, conf *config.Config, bindingdb db.BindingDB, policydb db.PolicyDB, catalog []domain.Service, cfClient cf.CFClient) *BrokerHandler {
	return &BrokerHandler{
		logger:                logger,
		conf:                  conf,
		bindingdb:             bindingdb,
		policydb:              policydb,
		catalog:               catalog,
		policyValidator:       policyvalidator.NewPolicyValidator(conf.PolicySchemaPath),
		schedulerUtil:         schedulerutil.NewSchedulerUtil(conf, logger),
		quotaManagementClient: quota.NewClient(conf, logger, cfClient),
		planChecker:           plancheck.NewPlanChecker(conf.PlanCheck, logger),
		cfClient:              cfClient,
	}
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	handlers.WriteJSONResponse(w, statusCode, models.ErrorResponse{
		Code:    http.StatusText(statusCode),
		Message: message})
}

func (h *BrokerHandler) GetBrokerCatalog(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
	catalog, err := ioutil.ReadFile(h.conf.CatalogPath)
	if err != nil {
		h.logger.Error("failed to read catalog file", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to load catalog")
		return
	}
	_, err = w.Write(catalog)
	if err != nil {
		h.logger.Error("unable to write body", err)
	}
}

func (h *BrokerHandler) GetHealth(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	//w.Write([]byte(`{"alive":"true"}`))
	handlers.WriteJSONResponse(w, http.StatusOK, []byte(`{"alive":"true"}`))
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

	if h.quotaExceeded(body, instanceId, w) {
		return
	}

	policyStr := ""
	if body.Parameters.DefaultPolicy != nil {
		policyStr = string(*body.Parameters.DefaultPolicy)
	}
	policyGuidStr := ""
	if policyStr != "" {
		errResults, valid := h.policyValidator.ValidatePolicy(policyStr)
		if !valid {
			h.logger.Error("failed to validate policy", err, lager.Data{"instanceId": instanceId, "policy": policyStr})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, errResults)
			return
		}

		if h.planDefinitionExceeded(policyStr, body.PlanID, instanceId, w) {
			return
		}

		policyGuid, err := uuid.NewV4()
		if err != nil {
			h.logger.Error("failed to create policy guid", err, lager.Data{"instanceId": instanceId})
			writeErrorResponse(w, http.StatusInternalServerError, "Error generating policy guid")
			return
		}
		policyGuidStr = policyGuid.String()
	}

	successResponse := func() {
		if h.conf.DashboardRedirectURI == "" {
			_, err = w.Write([]byte("{}"))
			if err != nil {
				h.logger.Error("unable to write body", err)
			}
		} else {
			_, err = w.Write([]byte(fmt.Sprintf("{\"dashboard_url\":\"%s\"}", GetDashboardURL(h.conf, instanceId))))
			if err != nil {
				h.logger.Error("unable to write body", err)
			}
		}
	}
	err = h.bindingdb.CreateServiceInstance(models.ServiceInstance{ServiceInstanceId: instanceId, OrgId: body.OrgGUID, SpaceId: body.SpaceGUID, DefaultPolicy: policyStr, DefaultPolicyGuid: policyGuidStr})
	switch {
	case err == nil:
		w.WriteHeader(http.StatusCreated)
		successResponse()
	case errors.Is(err, db.ErrAlreadyExists):
		h.logger.Error("failed to create service instance: service instance already exists", err, lager.Data{"instanceId": instanceId, "orgGuid": body.OrgGUID, "spaceGuid": body.SpaceGUID})
		successResponse()
	case errors.Is(err, db.ErrConflict):
		h.logger.Error("failed to create service instance: conflicting service instance exists", err, lager.Data{"instanceId": instanceId, "orgGuid": body.OrgGUID, "spaceGuid": body.SpaceGUID})
		writeErrorResponse(w, http.StatusConflict, fmt.Sprintf("Service instance with instance_id \"%s\" already exists with different parameters", instanceId))
	default:
		h.logger.Error("failed to create service instance", err, lager.Data{"instanceId": instanceId, "orgGuid": body.OrgGUID, "spaceGuid": body.SpaceGUID})
		writeErrorResponse(w, http.StatusInternalServerError, "Error creating service instance")
	}
}

func (h *BrokerHandler) planDefinitionExceeded(policyStr string, planID string, instanceId string, w http.ResponseWriter) bool {
	policy := models.ScalingPolicy{}
	err := json.Unmarshal([]byte(policyStr), &policy)
	if err != nil {
		h.logger.Error("failed to unmarshal policy", err, lager.Data{"instanceId": instanceId, "policyStr": policyStr})
		writeErrorResponse(w, http.StatusInternalServerError, "Error reading policy")
		return true
	}
	ok, checkResult, err := h.planChecker.CheckPlan(policy, planID)
	if err != nil {
		h.logger.Error("failed to check policy for plan adherence", err, lager.Data{"instanceId": instanceId, "policyStr": policyStr})
		writeErrorResponse(w, http.StatusInternalServerError, "Error generating validating policy")
		return true
	}
	if !ok {
		h.logger.Error("policy did not adhere to plan", fmt.Errorf(checkResult), lager.Data{"instanceId": instanceId, "policyStr": policyStr})
		writeErrorResponse(w, http.StatusBadRequest, checkResult)
		return true
	}
	return false
}

func (h *BrokerHandler) quotaExceeded(creationRequestBody *models.InstanceCreationRequestBody, instanceId string, w http.ResponseWriter) bool {
	logger := h.logger.Session("quota-check", lager.Data{"instanceId": instanceId})
	logger.Debug("start")
	defer logger.Debug("end")

	planId := creationRequestBody.PlanID
	serviceId := creationRequestBody.ServiceID
	orgGuid := creationRequestBody.OrgGUID
	spaceGuid := creationRequestBody.SpaceGUID

	serviceName := ""
	planName := ""

	for _, service := range h.catalog {
		if service.ID == serviceId {
			for _, plan := range service.Plans {
				if plan.ID == planId {
					serviceName = service.Name
					planName = plan.Name
				}
			}
		}
	}
	if serviceName == "" || planName == "" {
		h.logger.Error("failed to find selected service and plan in catalog", nil, lager.Data{"instanceId": instanceId, "orgGuid": orgGuid, "spaceGuid": spaceGuid, "serviceId": serviceId, "planId": planId, "serviceName": serviceName, "planName": planName})
		writeErrorResponse(w, http.StatusBadRequest, "Unknown service or plan")
		return true
	}
	quota, err := h.quotaManagementClient.GetQuota(orgGuid, serviceName, planName)
	logger.Debug("quota-management-client-returned", lager.Data{"quota": quota, "instanceId": instanceId, "orgGuid": orgGuid, "spaceGuid": spaceGuid, "serviceId": serviceId, "planId": planId, "serviceName": serviceName, "planName": planName})

	if err != nil {
		h.logger.Error("failed to call quota management API", err, lager.Data{"instanceId": instanceId, "orgGuid": orgGuid, "spaceGuid": spaceGuid, "serviceId": serviceId, "planId": planId, "serviceName": serviceName, "planName": planName})
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to determine available Application Autoscaler quota for your subaccount. Please try again later.")
		return true
	}
	if quota == 0 {
		h.logger.Error("failed to create service instance due to missing quota", nil, lager.Data{"instanceId": instanceId, "orgGuid": orgGuid, "spaceGuid": spaceGuid, "serviceId": serviceId, "planId": planId, "serviceName": serviceName, "planName": planName, "quota": quota})
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf(`No quota for this service "%s" and service plan "%s" has been assigned to your subaccount. Please contact your global account administrator for help on how to assign Application Autoscaler quota to your subaccount.`, serviceName, planName))
		return true
	}
	if quota > 0 {
		instances, err := h.quotaManagementClient.GetServiceInstancesInOrg(orgGuid, planId)
		logger.Debug("quota-management-client-get-service-instances-returned", lager.Data{"instancesCount": instances, "quota": quota, "instanceId": instanceId, "orgGuid": orgGuid, "spaceGuid": spaceGuid, "serviceId": serviceId, "planId": planId, "serviceName": serviceName, "planName": planName})

		if err != nil {
			h.logger.Error("failed to count currently existing service instances", err, lager.Data{"instanceId": instanceId, "orgGuid": orgGuid, "spaceGuid": spaceGuid, "serviceId": serviceId, "planId": planId, "serviceName": serviceName, "planName": planName})
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to determine used quota. Please try again later.")
			return true
		}
		if instances+1 > quota {
			h.logger.Error("failed to create service instance due to insufficient quota", nil, lager.Data{"instanceId": instanceId, "orgGuid": orgGuid, "spaceGuid": spaceGuid, "serviceId": serviceId, "planId": planId, "serviceName": serviceName, "planName": planName, "quota": quota, "instances": instances})
			writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf(`The quota of %d service instances of service '%s' with plan '%s' within this subaccount has been exceeded (%d instances already exist). Please contact your global account administrator for help on how to assign more Application Autoscaler quota to your subaccount.`, quota, serviceName, planName, instances))
			return true
		}
	}
	return false
}

func (h *BrokerHandler) UpdateServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]

	body := &models.InstanceUpdateRequestBody{}
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read service update request body", err, lager.Data{"instanceId": instanceId})
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to read request body")
		return
	}
	err = json.Unmarshal(bodyBytes, body)
	if err != nil {
		h.logger.Error("failed to unmarshal service update body", err, lager.Data{"instanceId": instanceId, "body": string(bodyBytes)})
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body format")
		return
	}

	if instanceId == "" || body.ServiceID == "" {
		h.logger.Error("failed to update service instance when trying to get mandatory data", nil, lager.Data{"instanceId": instanceId, "serviceId": body.ServiceID, "planId": body.PlanID})
		writeErrorResponse(w, http.StatusBadRequest, "Malformed or missing mandatory data")
		return
	}

	if body.Parameters == nil || body.Parameters.DefaultPolicy == nil {
		h.logger.Error("failed to update instance, only default policy updates allowed", nil, lager.Data{"instanceId": instanceId, "serviceId": body.ServiceID, "planId": body.PlanID})
		writeErrorResponse(w, http.StatusUnprocessableEntity, "Failed to update service instance: Only default_policy updates allowed")
		return
	}
	updatedDefaultPolicy := string(*body.Parameters.DefaultPolicy)

	if emptyJSONObject.MatchString(updatedDefaultPolicy) {
		// accept an empty json object "{}" as a default policy update to specify the removal of the default policy
		updatedDefaultPolicy = ""
	}

	updatedDefaultPolicyGuid := ""
	if updatedDefaultPolicy != "" {
		errResults, valid := h.policyValidator.ValidatePolicy(updatedDefaultPolicy)
		if !valid {
			h.logger.Error("failed to validate policy", err, lager.Data{"instanceId": instanceId, "policy": updatedDefaultPolicy})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, errResults)
			return
		}

		servicePlan, err := h.cfClient.GetServicePlan(instanceId)
		if err != nil {
			h.logger.Error("failed-to-retrieve-service-plan-of-service-instance", err, lager.Data{"instanceId": instanceId})
			writeErrorResponse(w, http.StatusInternalServerError, "Error validating policy")
			return
		}
		if h.planDefinitionExceeded(updatedDefaultPolicy, servicePlan, instanceId, w) {
			return
		}

		policyGuid, err := uuid.NewV4()
		if err != nil {
			h.logger.Error("failed to create policy guid", err, lager.Data{"instanceId": instanceId})
			writeErrorResponse(w, http.StatusInternalServerError, "Error generating policy guid")
			return
		}
		updatedDefaultPolicyGuid = policyGuid.String()
	}

	serviceInstance, err := h.bindingdb.GetServiceInstance(instanceId)
	if err != nil {
		if errors.Is(err, db.ErrDoesNotExist) {
			h.logger.Error("failed to find service instance to update", err, lager.Data{"instanceId": instanceId})
			writeErrorResponse(w, http.StatusNotFound, "Failed to find service instance to update")
			return
		} else {
			h.logger.Error("failed to retrieve service instance", err, lager.Data{"instanceId": instanceId})
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve service instance")
			return
		}
	}

	if updatedDefaultPolicyGuid != "" {
		allBoundApps, err := h.bindingdb.GetAppIdsByInstanceId(serviceInstance.ServiceInstanceId)
		if err != nil {
			h.logger.Error("failed to retrieve bound apps", err, lager.Data{"instanceId": instanceId})
			writeErrorResponse(w, http.StatusInternalServerError, "Error updating service instance")
			return
		}
		updatedAppIds, err := h.policydb.SetOrUpdateDefaultAppPolicy(allBoundApps, serviceInstance.DefaultPolicyGuid, updatedDefaultPolicy, updatedDefaultPolicyGuid)
		if err != nil {
			h.logger.Error("failed to set default policies", err, lager.Data{"instanceId": instanceId})
			writeErrorResponse(w, http.StatusInternalServerError, "Error updating service instance")
			return
		}

		// there is synchronization between policy and schedule, so errors creating schedules should not break
		// the whole update process
		for _, appId := range updatedAppIds {
			if err = h.schedulerUtil.CreateOrUpdateSchedule(appId, updatedDefaultPolicy, updatedDefaultPolicyGuid); err != nil {
				h.logger.Error("failed to create/update schedules", err, lager.Data{"appId": appId, "policyGuid": updatedDefaultPolicyGuid, "policy": updatedDefaultPolicy})
			}
		}
	} else {
		if serviceInstance.DefaultPolicyGuid != "" {
			// default policy was present and will now be removed
			updatedAppIds, err := h.policydb.DeletePoliciesByPolicyGuid(serviceInstance.DefaultPolicyGuid)
			if err != nil {
				h.logger.Error("failed to delete default policies", err, lager.Data{"instanceId": instanceId})
				writeErrorResponse(w, http.StatusInternalServerError, "Error updating service instance")
				return
			}
			// there is synchronization between policy and schedule, so errors creating schedules should not break
			// the whole update process
			for _, appId := range updatedAppIds {
				if err = h.schedulerUtil.DeleteSchedule(appId); err != nil {
					h.logger.Error("failed to delete schedules", err, lager.Data{"appId": appId})
				}
			}
		}
	}

	updatedServiceInstance := models.ServiceInstance{
		ServiceInstanceId: serviceInstance.ServiceInstanceId,
		OrgId:             serviceInstance.OrgId,
		SpaceId:           serviceInstance.SpaceId,
		DefaultPolicy:     updatedDefaultPolicy,
		DefaultPolicyGuid: updatedDefaultPolicyGuid,
	}

	err = h.bindingdb.UpdateServiceInstance(updatedServiceInstance)
	if err != nil {
		h.logger.Error("failed to update service instance", err, lager.Data{"instanceId": instanceId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error updating service instance")
		return
	}

	_, err = w.Write([]byte("{}"))
	if err != nil {
		h.logger.Error("unable to write body", err)
	}
}

func (h *BrokerHandler) DeleteServiceInstance(w http.ResponseWriter, _ *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]
	if instanceId == "" {
		h.logger.Error("failed to delete service instance when trying to get mandatory data", nil,
			lager.Data{"instanceId": instanceId})
		writeErrorResponse(w, http.StatusBadRequest, "Malformed or missing mandatory data")
		return
	}

	// fetch and delete service bindings
	bindingIds, err := h.bindingdb.GetBindingIdsByInstanceId(instanceId)
	if err != nil {
		h.logger.Error("failed to delete service bindings before service instance deletion", err, lager.Data{"instanceId": instanceId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting service instance")
		return
	}

	for _, bindingId := range bindingIds {
		err = deleteBinding(h, bindingId, instanceId)
		wrappedError := fmt.Errorf("service instance deletion failed: %w", err)
		if err != nil && (errors.Is(err, errorDeleteServiceBinding) ||
			errors.Is(err, errorDeletePolicyForUnbinding) ||
			errors.Is(err, errorDeleteSchedulesForUnbinding) ||
			errors.Is(err, errorCredentialNotDeleted)) {
			writeErrorResponse(w, http.StatusInternalServerError, wrappedError.Error())
			return
		}
	}

	err = h.bindingdb.DeleteServiceInstance(instanceId)
	if err != nil {
		if errors.Is(err, db.ErrDoesNotExist) {
			h.logger.Error("failed to delete service instance: service instance does not exist", err,
				lager.Data{"instanceId": instanceId})
			writeErrorResponse(w, http.StatusGone, "Service Instance Doesn't Exist")
			return
		}
		h.logger.Error("failed to delete service instance", err, lager.Data{"instanceId": instanceId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting service instance")
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("{}"))
	if err != nil {
		h.logger.Error("unable to write body", err)
	}
}

func (h *BrokerHandler) BindServiceInstance(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]
	bindingId := vars["bindingId"]
	var policyGuidStr string
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

		if h.planDefinitionExceeded(policyStr, body.PlanID, instanceId, w) {
			return
		}

		policyGuid, err := uuid.NewV4()
		if err != nil {
			h.logger.Error("failed to create policy guid", err, lager.Data{"appId": body.AppID})
			writeErrorResponse(w, http.StatusInternalServerError, "Error generating policy guid")
			return
		}
		policyGuidStr = policyGuid.String()
	}

	// fallback to default policy if no policy was provided
	if policyStr == "" {
		if serviceInstance, err := h.bindingdb.GetServiceInstance(instanceId); err != nil {
			h.logger.Error("failed to get default policy", err, lager.Data{"appId": body.AppID, "instanceId": instanceId, "bindingId": bindingId})
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, "Error reading the default policy")
			return
		} else {
			policyStr = serviceInstance.DefaultPolicy
			policyGuidStr = serviceInstance.DefaultPolicyGuid
		}
	}

	// fetch and all service bindings for the service instance
	bindingIds, err := h.bindingdb.GetBindingIdsByInstanceId(instanceId)
	if err != nil {
		h.logger.Error("failed to delete service bindings before service instance deletion", err, lager.Data{"instanceId": instanceId})
		writeErrorResponse(w, http.StatusInternalServerError, "Error deleting service instance")
		return
	}

	for _, bindingId := range bindingIds {
		// get the service binding for the app
		fetchedAppID, err := h.bindingdb.GetAppIdByBindingId(bindingId)
		if err != nil {
			h.logger.Error("unable to get appId for bindingId", err, lager.Data{"instanceId": instanceId})
			writeErrorResponse(w, http.StatusInternalServerError, "Error deleting service instance")
			return
		}

		//select the binding-id for the app
		if fetchedAppID == body.AppID {
			err = deleteBinding(h, bindingId, instanceId)
			wrappedError := fmt.Errorf("Failed to bind service: %w", err)
			if err != nil && (errors.Is(err, errorDeleteServiceBinding) ||
				errors.Is(err, errorDeletePolicyForUnbinding) ||
				errors.Is(err, errorDeleteSchedulesForUnbinding) ||
				errors.Is(err, errorCredentialNotDeleted)) {
				writeErrorResponse(w, http.StatusInternalServerError, wrappedError.Error())
				return
			}
		}
	}

	err = h.bindingdb.CreateServiceBinding(bindingId, instanceId, body.AppID)
	if err != nil {
		if errors.Is(err, db.ErrAlreadyExists) {
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
		err = h.policydb.SaveAppPolicy(body.AppID, policyStr, policyGuidStr)
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
		err = h.schedulerUtil.CreateOrUpdateSchedule(body.AppID, policyStr, policyGuidStr)
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

func (h *BrokerHandler) UnbindServiceInstance(w http.ResponseWriter, _ *http.Request, vars map[string]string) {
	instanceId := vars["instanceId"]
	bindingId := vars["bindingId"]

	if instanceId == "" || bindingId == "" {
		h.logger.Error("failed to delete binding when trying to get mandatory data", nil, lager.Data{"instanceId": instanceId, "bindingId": bindingId})
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body format")
		return
	}

	err := deleteBinding(h, bindingId, instanceId)
	wrappedError := fmt.Errorf("Failed to unbind service: %w", err)

	if err != nil && errors.Is(err, errorBindingDoesNotExist) {
		writeErrorResponse(w, http.StatusGone, wrappedError.Error())
		return
	} else if err != nil && (errors.Is(err, errorDeleteServiceBinding) ||
		errors.Is(err, errorDeletePolicyForUnbinding) ||
		errors.Is(err, errorDeleteSchedulesForUnbinding) ||
		errors.Is(err, errorCredentialNotDeleted)) {
		writeErrorResponse(w, http.StatusInternalServerError, wrappedError.Error())
		return
	} else if err != nil && errors.Is(err, errorDeleteServiceBinding) {
		writeErrorResponse(w, http.StatusGone, wrappedError.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("{}"))
	if err != nil {
		h.logger.Error("unable to write body", err)
	}
}

func deleteBinding(h *BrokerHandler, bindingId string, serviceInstanceId string) error {
	appId, err := h.bindingdb.GetAppIdByBindingId(bindingId)
	if errors.Is(err, sql.ErrNoRows) {
		h.logger.Info("binding does not exist", nil, lager.Data{"instanceId": serviceInstanceId, "bindingId": bindingId})
		return errorBindingDoesNotExist
	}
	if err != nil {
		h.logger.Error("failed to get appId by bindingId", err, lager.Data{"instanceId": serviceInstanceId, "bindingId": bindingId})
		return errorDeleteServiceBinding
	}
	h.logger.Info("deleting policy json", lager.Data{"appId": appId})
	err = h.policydb.DeletePolicy(appId)
	if err != nil {
		h.logger.Error("failed to delete policy for unbinding", err, lager.Data{"appId": appId})
		return errorDeletePolicyForUnbinding
	}

	h.logger.Info("deleting schedules", lager.Data{"appId": appId})
	err = h.schedulerUtil.DeleteSchedule(appId)
	if err != nil {
		h.logger.Info("failed to delete schedules for unbinding", lager.Data{"appId": appId})
		return errorDeleteSchedulesForUnbinding
	}
	err = h.bindingdb.DeleteServiceBinding(bindingId)
	if err != nil {
		h.logger.Error("failed to delete binding", err, lager.Data{"bindingId": bindingId, "appId": appId})
		if errors.Is(err, db.ErrDoesNotExist) {
			return errorBindingDoesNotExist
		}

		return errorDeleteServiceBinding
	}

	err = custom_metrics_cred_helper.DeleteCredential(appId, h.policydb, custom_metrics_cred_helper.MaxRetry)
	if err != nil {
		h.logger.Error("failed to delete custom metrics credential for unbinding", err, lager.Data{"appId": appId})
		return errorCredentialNotDeleted
	}

	return nil
}
