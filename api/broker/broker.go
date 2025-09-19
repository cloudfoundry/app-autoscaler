package broker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/plancheck"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/policyvalidator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/schedulerclient"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/brokerapi/v13/domain"
	"code.cloudfoundry.org/brokerapi/v13/domain/apiresponses"
	"code.cloudfoundry.org/lager/v3"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

var _ domain.ServiceBroker = &Broker{}

type Broker struct {
	logger          lager.Logger
	conf            *config.Config
	bindingdb       db.BindingDB
	policydb        db.PolicyDB
	policyValidator *policyvalidator.PolicyValidator
	schedulerUtil   *schedulerclient.Client
	catalog         []domain.Service
	PlanChecker     plancheck.PlanChecker
	credentials     cred_helper.Credentials
}

var (
	emptyJSONObject                 = regexp.MustCompile(`^\s*{\s*}\s*$`)
	ErrCreatingServiceBinding       = errors.New("error creating service binding")
	ErrUpdatingServiceInstance      = errors.New("error updating service instance")
	ErrDeleteSchedulesForUnbinding  = errors.New("failed to delete schedules for unbinding")
	ErrBindingDoesNotExist          = errors.New("service binding does not exist")
	ErrDeletePolicyForUnbinding     = errors.New("failed to delete policy for unbinding")
	ErrDeleteServiceBinding         = errors.New("error deleting service binding")
	ErrCredentialNotDeleted         = errors.New("failed to delete custom metrics credential for unbinding")
	ErrInvalidCredentialType        = errors.New("invalid credential type provided: allowed values are [binding-secret, x509]")
	ErrInvalidConfigurations        = errors.New("invalid binding configurations provided")
	ErrInvalidCustomMetricsStrategy = errors.New("error: custom metrics strategy not supported")
)

type Errors []error

func (e Errors) Error() string {
	var theErrors []string
	for _, err := range e {
		theErrors = append(theErrors, err.Error())
	}
	return strings.Join(theErrors, ", ")
}

var _ error = Errors{}

func New(logger lager.Logger, conf *config.Config, bindingDb db.BindingDB, policyDb db.PolicyDB, catalog []domain.Service, credentials cred_helper.Credentials) *Broker {
	broker := &Broker{
		logger:    logger,
		conf:      conf,
		bindingdb: bindingDb,
		policydb:  policyDb,
		catalog:   catalog,
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
		PlanChecker:   plancheck.NewPlanChecker(conf.PlanCheck, logger),
		credentials:   credentials,
	}
	return broker
}

// Services gets the catalog of services offered by the service broker
// GET /v2/catalog
func (b *Broker) Services(_ context.Context) ([]domain.Service, error) {
	return b.catalog, nil
}

// Provision creates a new service instance
// PUT /v2/service_instances/{instance_id}
func (b *Broker) Provision(ctx context.Context, instanceID string, details domain.ProvisionDetails, _ bool) (domain.ProvisionedServiceSpec, error) {
	result := domain.ProvisionedServiceSpec{}

	logger := b.logger.Session("provision", lager.Data{"instanceID": instanceID, "provisionDetails": details})
	logger.Info("begin")
	defer logger.Info("end")

	fields := map[string]string{
		"instanceID":        instanceID,
		"organization_guid": details.OrganizationGUID,
		"space_guid":        details.SpaceGUID,
		"plan_id":           details.PlanID,
	}

	for name, value := range fields {
		if value == "" {
			err := fmt.Errorf("missing %s", name)
			logger.Error("missing-mandatory-field", err)
			return result, apiresponses.NewFailureResponse(err, http.StatusBadRequest, "missing-mandatory-field")
		}
	}

	parameters, err := parseInstanceParameters(details.RawParameters)
	if err != nil {
		return result, err
	}

	var policyJson json.RawMessage
	if parameters.DefaultPolicy != nil {
		policyJson = parameters.DefaultPolicy
	}

	policy, err := b.getPolicyFromJsonRawMessage(policyJson, instanceID, details.PlanID)
	if err != nil {
		// The input may be not parsable, hence we use the original string.
		policyStr := string(policyJson)
		b.logger.Error("setting default policy", err, lager.Data{"policy": policyStr})
		return result, err
	}

	var policyStr, policyGuidStr string
	if policy != nil {
		policyGuidStr = uuid.NewString()

		// We generate the json again on our own to ensure a consistent formatting.
		policyAsJson, err := policy.ToRawJSON()
		if err != nil {
			const msg = "⚠️ this should never happen as the policy has been parsed from json before."
			logger.Error("failed-marshalling-policy", err, lager.Data{
				"policy":  policy,
				"message": msg,
			})
			return result, fmt.Errorf(
				"policy not serialised into json\n\t%s\n\terror: %w", msg, err)
		}

		policyStr = string(policyAsJson)
	}

	instance := models.ServiceInstance{
		ServiceInstanceId: instanceID,
		OrgId:             details.OrganizationGUID,
		SpaceId:           details.SpaceGUID,
		DefaultPolicy:     policyStr,     // empty string if no default policy is set (i.e. nil)
		DefaultPolicyGuid: policyGuidStr, // empty string if no default policy is set (i.e. nil)
	}
	err = b.bindingdb.CreateServiceInstance(ctx, instance)
	switch {
	case err == nil:
		result.DashboardURL = GetDashboardURL(b.conf, instanceID)
	case errors.Is(err, db.ErrAlreadyExists):
		logger.Error("failed to create service instance: service instance already exists", err, lager.Data{"instanceID": instanceID, "orgGuid": details.OrganizationGUID, "spaceGuid": details.SpaceGUID})
		result.DashboardURL = GetDashboardURL(b.conf, instanceID)
		result.AlreadyExists = true
		err = nil
	case errors.Is(err, db.ErrConflict):
		logger.Error("failed to create service instance: conflicting service instance exists", err, lager.Data{"instanceID": instanceID, "orgGuid": details.OrganizationGUID, "spaceGuid": details.SpaceGUID})
		err = apiresponses.ErrInstanceAlreadyExists
	default:
		logger.Error("failed to create service instance", err, lager.Data{"instanceID": instanceID, "orgGuid": details.OrganizationGUID, "spaceGuid": details.SpaceGUID})
		err = apiresponses.NewFailureResponse(errors.New("error creating service instance"), http.StatusInternalServerError, "failed to create service instance")
	}
	return result, err
}

func (b *Broker) getPolicyFromJsonRawMessage(policyJson json.RawMessage, instanceID string, planID string) (*models.ScalingPolicy, error) {
	if isEmptyPolicy := len(policyJson) <= 0; isEmptyPolicy { // no nil-check needed: `len(nil) == 0`
		return nil, nil
	}

	return b.validateAndCheckPolicy(policyJson, instanceID, planID)
}

func (b *Broker) validateAndCheckPolicy(rawJson json.RawMessage, instanceID string, planID string) (*models.ScalingPolicy, error) {
	policy, errResults := b.policyValidator.ParseAndValidatePolicy(rawJson)
	logger := b.logger.Session("validate-and-check-policy", lager.Data{"instanceID": instanceID, "policy": policy, "planID": planID, "errResults": errResults})

	if errResults != nil {
		// 🚫 The subsequent log-message is a strong assumption about the context of the caller. But
		// how can we actually know here that we operate on a default-policy? In fact, when we are
		// in the call-stack of `Bind` then we are *not* called with a default-policy.
		logger.Info("got-invalid-default-policy")
		resultsJson, err := json.Marshal(errResults)
		if err != nil {
			logger.Error("failed-marshalling-errors", err)
		}
		return policy, apiresponses.NewFailureResponse(fmt.Errorf("invalid policy provided: %s", string(resultsJson)), http.StatusBadRequest, "failed-to-validate-policy")
	}
	if err := b.planDefinitionExceeded(policy.GetPolicyDefinition(), planID, instanceID); err != nil {
		return policy, err
	}
	return policy, nil
}

// Deprovision deletes an existing service instance
// DELETE /v2/service_instances/{instance_id}
func (b *Broker) Deprovision(ctx context.Context, instanceID string, details domain.DeprovisionDetails, _ bool) (domain.DeprovisionServiceSpec, error) {
	result := domain.DeprovisionServiceSpec{}

	logger := b.logger.Session("deprovision", lager.Data{"instanceID": instanceID, "deprovisionDetails": details})
	logger.Info("begin")
	defer logger.Info("end")

	serviceInstanceDeletionError := errors.New("error deleting service instance")
	// fetch and delete service bindings
	bindingIds, err := b.bindingdb.GetBindingIdsByInstanceId(ctx, instanceID)
	if err != nil {
		logger.Error("list-bindings-of-service-instance-to-be-deleted", err)
		return result, apiresponses.NewFailureResponse(serviceInstanceDeletionError, http.StatusInternalServerError, "list-bindings-of-service-instance-to-be-deleted")
	}

	for _, bindingId := range bindingIds {
		err = b.deleteBinding(ctx, bindingId, instanceID)
		wrappedError := fmt.Errorf("service binding deletion failed: %w", err)
		if err != nil && (errors.Is(err, ErrDeleteServiceBinding) ||
			errors.Is(err, ErrDeletePolicyForUnbinding) ||
			errors.Is(err, ErrDeleteSchedulesForUnbinding) ||
			errors.Is(err, ErrCredentialNotDeleted)) {
			logger.Error("delete-bindings-of-service-instance-to-be-deleted", wrappedError)
			return result, apiresponses.NewFailureResponse(serviceInstanceDeletionError, http.StatusInternalServerError, "delete-bindings-of-service-instance-to-be-deleted")
		}
	}

	err = b.bindingdb.DeleteServiceInstance(ctx, instanceID)
	if err != nil {
		if errors.Is(err, db.ErrDoesNotExist) {
			logger.Error("failed to delete service instance: service instance does not exist", err)
			return result, apiresponses.ErrInstanceDoesNotExist
		}
		logger.Error("delete-service-instance", err)
		return result, apiresponses.NewFailureResponse(serviceInstanceDeletionError, http.StatusInternalServerError, "delete-service-instance")
	}

	return result, nil
}

// GetInstance fetches information about a service instance
// GET /v2/service_instances/{instance_id}
func (b *Broker) GetInstance(ctx context.Context, instanceID string, details domain.FetchInstanceDetails) (domain.GetInstanceDetailsSpec, error) {
	logger := b.logger.Session("get-instance", lager.Data{"instanceID": instanceID, "fetchInstanceDetails": details})
	logger.Info("begin")
	defer logger.Info("end")

	serviceInstance, err := b.getServiceInstance(ctx, instanceID)
	if err != nil {
		return domain.GetInstanceDetailsSpec{}, err
	}

	result := domain.GetInstanceDetailsSpec{
		ServiceID:    details.ServiceID,
		PlanID:       details.PlanID,
		DashboardURL: GetDashboardURL(b.conf, instanceID),
	}

	if serviceInstance.DefaultPolicy != "" {
		policy := json.RawMessage(serviceInstance.DefaultPolicy)
		result.Parameters = models.InstanceParameters{
			DefaultPolicy: policy,
		}
	}

	return result, nil
}

// Update modifies an existing service instance
// PATCH /v2/service_instances/{instance_id}
func (b *Broker) Update(ctx context.Context, instanceID string, details domain.UpdateDetails, _ bool) (domain.UpdateServiceSpec, error) {
	logger := b.logger.Session("update", lager.Data{"instanceID": instanceID, "updateDetails": details})
	logger.Info("begin")
	defer logger.Info("end")

	result := domain.UpdateServiceSpec{}

	serviceInstance, err := b.getServiceInstance(ctx, instanceID)
	if err != nil {
		return result, err
	}

	servicePlan, servicePlanIsNew, err := b.getExistingOrUpdatedServicePlan(instanceID, details)
	if err != nil {
		return result, err
	}

	parameters, err := parseInstanceParameters(details.RawParameters)
	if err != nil {
		return result, err
	}

	// determine new default policy if any
	defaultPolicy, defaultPolicyGuid, defaultPolicyIsNew, err := b.determineDefaultPolicy(parameters, serviceInstance, servicePlan)
	if err != nil {
		return result, err
	}

	if !servicePlanIsNew && !defaultPolicyIsNew {
		logger.Info("no-changes-requested")
		return result, nil
	}

	logger.Info("update-service-instance", lager.Data{"instanceID": instanceID, "serviceId": details.ServiceID, "planId": details.PlanID, "defaultPolicy": defaultPolicy})

	allBoundApps, err := b.bindingdb.GetAppIdsByInstanceId(ctx, serviceInstance.ServiceInstanceId)
	if err != nil {
		logger.Error("failed to retrieve bound apps", err, lager.Data{"instanceID": instanceID})
		return result, apiresponses.NewFailureResponse(ErrUpdatingServiceInstance, http.StatusInternalServerError, "failed to retrieve bound apps")
	}

	if servicePlanIsNew {
		if err := b.checkScalingPoliciesUnderNewPlan(ctx, allBoundApps, servicePlan, instanceID); err != nil {
			return result, err
		}
	}

	if defaultPolicyIsNew {
		if err := b.applyDefaultPolicyUpdate(ctx, allBoundApps, serviceInstance, defaultPolicy, defaultPolicyGuid); err != nil {
			return result, err
		}
		defaultPolicyBytes := []byte("")
		if defaultPolicy != nil {
			defaultPolicyBytes, err = json.Marshal(defaultPolicy)
			logger.Info("saving default policy", lager.Data{"policy": defaultPolicy, "policyStr": string(defaultPolicyBytes), "err": err})
			if err != nil {
				return result, err
			}
		}
		// persist the changes to the default policy
		// NOTE: As the plan is not persisted, we do not need to this if we are only performing a plan change!
		updatedServiceInstance := models.ServiceInstance{
			ServiceInstanceId: serviceInstance.ServiceInstanceId,
			OrgId:             serviceInstance.OrgId,
			SpaceId:           serviceInstance.SpaceId,
			DefaultPolicy:     string(defaultPolicyBytes),
			DefaultPolicyGuid: defaultPolicyGuid,
		}

		err = b.bindingdb.UpdateServiceInstance(ctx, updatedServiceInstance)
		if err != nil {
			logger.Error("failed to update service instance", err, lager.Data{"instanceID": instanceID})
			return result, apiresponses.NewFailureResponse(ErrUpdatingServiceInstance, http.StatusInternalServerError, "update service instance")
		}
	}

	return result, nil
}

func (b *Broker) applyDefaultPolicyUpdate(ctx context.Context, allBoundApps []string, serviceInstance *models.ServiceInstance, defaultPolicy *models.PolicyDefinition, defaultPolicyGuid string) error {
	if defaultPolicy == nil {
		// default policy was present and will now be removed
		return b.removeDefaultPolicyFromApps(ctx, serviceInstance)
	}
	return b.setDefaultPolicyOnApps(ctx, defaultPolicy, defaultPolicyGuid, allBoundApps, serviceInstance)
}

func parseInstanceParameters(rawParameters json.RawMessage) (*models.InstanceParameters, error) {
	parameters := &models.InstanceParameters{}
	if rawParameters != nil {
		err := json.Unmarshal(rawParameters, parameters)
		if err != nil {
			return nil, apiresponses.ErrRawParamsInvalid
		}
	}
	return parameters, nil
}

func (b *Broker) getServiceInstance(ctx context.Context, instanceID string) (*models.ServiceInstance, error) {
	serviceInstance, err := b.bindingdb.GetServiceInstance(ctx, instanceID)
	if err != nil {
		if errors.Is(err, db.ErrDoesNotExist) {
			b.logger.Error("failed to find service instance to update", err, lager.Data{"instanceID": instanceID})
			return nil, apiresponses.ErrInstanceDoesNotExist
		} else {
			b.logger.Error("failed to retrieve service instance", err, lager.Data{"instanceID": instanceID})
			return nil, apiresponses.NewFailureResponse(errors.New("failed to retrieve service instance"), http.StatusInternalServerError, "retrieving-instance-for-update")
		}
	}
	return serviceInstance, nil
}

func (b *Broker) setDefaultPolicyOnApps(ctx context.Context, updatedDefaultPolicy *models.PolicyDefinition, updatedDefaultPolicyGuid string, allBoundApps []string, serviceInstance *models.ServiceInstance) error {
	instanceID := serviceInstance.ServiceInstanceId
	b.logger.Info("update-service-instance-set-or-update", lager.Data{"instanceID": instanceID, "updatedDefaultPolicy": updatedDefaultPolicy, "updatedDefaultPolicyGuid": updatedDefaultPolicyGuid, "allBoundApps": allBoundApps, "serviceInstance": serviceInstance})

	appIds, err := b.policydb.SetOrUpdateDefaultAppPolicy(ctx, allBoundApps, serviceInstance.DefaultPolicyGuid, updatedDefaultPolicy, updatedDefaultPolicyGuid)
	if err != nil {
		b.logger.Error("failed to set default policies", err, lager.Data{"instanceID": instanceID})
		return apiresponses.NewFailureResponse(errors.New("failed to set default policy"), http.StatusInternalServerError, "updating-default-policy")
	}
	var errs Errors
	for _, appId := range appIds {
		err = b.schedulerUtil.CreateOrUpdateSchedule(ctx, appId, updatedDefaultPolicy, updatedDefaultPolicyGuid)
		if err != nil {
			b.logger.Error("failed to create/update schedules", err, lager.Data{"appId": appId, "policyGuid": updatedDefaultPolicyGuid, "policy": updatedDefaultPolicy})
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (b *Broker) removeDefaultPolicyFromApps(ctx context.Context, serviceInstance *models.ServiceInstance) error {
	updatedAppIds, err := b.policydb.DeletePoliciesByPolicyGuid(ctx, serviceInstance.DefaultPolicyGuid)
	if err != nil {
		b.logger.Error("failed to delete default policies", err, lager.Data{"instanceID": serviceInstance.ServiceInstanceId})
		return apiresponses.NewFailureResponse(errors.New("failed to delete default policy"), http.StatusInternalServerError, "deleting-default-policy")
	}
	// there is synchronization between policy and schedule, so errors deleting schedules should not break
	// the whole update process
	for _, appId := range updatedAppIds {
		if err = b.schedulerUtil.DeleteSchedule(ctx, appId); err != nil {
			b.logger.Error("failed to delete schedules", err, lager.Data{"appId": appId})
		}
	}
	return nil
}

func (b *Broker) checkScalingPoliciesUnderNewPlan(ctx context.Context, allBoundApps []string, servicePlan string, instanceID string) error {
	var existingPolicy *models.PolicyDefinition
	var err error
	for _, appId := range allBoundApps {
		existingPolicy, err = b.policydb.GetAppPolicy(ctx, appId)
		if err != nil {
			b.logger.Error("failed to retrieve policy from db", err, lager.Data{"appId": appId})
			return apiresponses.NewFailureResponse(ErrUpdatingServiceInstance, http.StatusInternalServerError, "failed to retrieve policy from db")
		}

		err := b.planDefinitionExceeded(existingPolicy, servicePlan, instanceID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Broker) determineDefaultPolicy(parameters *models.InstanceParameters,
	serviceInstance *models.ServiceInstance, planID string,
) (defaultPolicy *models.PolicyDefinition, defaultPolicyGuid string, defaultPolicyIsNew bool, err error) {
	if serviceInstance.DefaultPolicy != "" {
		err = json.Unmarshal([]byte(serviceInstance.DefaultPolicy), &defaultPolicy)
		if err != nil {
			return nil, "", false, fmt.Errorf("unmarhaling default policy failed: %w", err)
		}
	}

	defaultPolicyGuid = serviceInstance.DefaultPolicyGuid
	defaultPolicyIsNew = false

	if string(parameters.DefaultPolicy) == "" {
		return defaultPolicy, defaultPolicyGuid, false, nil
	}

	newDefaultPolicy := parameters.DefaultPolicy
	if emptyJSONObject.MatchString(string(newDefaultPolicy)) {
		// accept an empty json object "{}" as a default policy update to specify the removal of the default policy
		if defaultPolicy != nil {
			defaultPolicy = nil
			defaultPolicyGuid = ""
			defaultPolicyIsNew = true
		}
	} else {
		newPolicy, err := b.validateAndCheckPolicy(newDefaultPolicy, serviceInstance.ServiceInstanceId, planID)
		if err != nil {
			return nil, "", false, err
		}
		newPolicyDefinition := newPolicy.GetPolicyDefinition()
		policyGuid := uuid.NewString()
		defaultPolicy = newPolicyDefinition
		defaultPolicyGuid = policyGuid
		defaultPolicyIsNew = true
	}

	return defaultPolicy, defaultPolicyGuid, defaultPolicyIsNew, nil
}

// LastOperation fetches last operation state for a service instance
// GET /v2/service_instances/{instance_id}/last_operation
func (b *Broker) LastOperation(_ context.Context, instanceID string, details domain.PollDetails) (domain.LastOperation, error) {
	logger := b.logger.Session("last-operation", lager.Data{"instanceID": instanceID, "pollDetails": details})
	logger.Info("begin")
	defer logger.Info("end")

	err := errors.New("error: last-operation is not implemented and this endpoint should not have been called as all broker operations are synchronous")
	logger.Error("last-operation-is-not-implemented", err)
	return domain.LastOperation{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, "last-operation-is-not-implemented")
}

// Bind creates a new service binding
// PUT /v2/service_instances/{instance_id}/service_bindings/{binding_id}
func (b *Broker) Bind(
	ctx context.Context, instanceID string, bindingID string, details domain.BindDetails, _ bool,
) (domain.Binding, error) {
	logger := b.logger.Session("bind", lager.Data{"instanceID": instanceID, "bindingID": bindingID, "bindDetails": details})
	logger.Info("begin")
	defer logger.Info("end")

	result := domain.Binding{}

	var scalingPolicyRaw json.RawMessage
	if details.RawParameters != nil {
		scalingPolicyRaw = details.RawParameters
	}

	// This just gets used for legacy-reasons. The actually parsing happens in the step
	// afterwards. But it still does not validate against the schema, which is done here.
	_, err := b.getPolicyFromJsonRawMessage(scalingPolicyRaw, instanceID, details.PlanID)
	if err != nil {
		logger.Error("get-default-policy", err)
		return result, err
	}

	scalingPolicy, err := b.getScalingPolicyFromRequest(scalingPolicyRaw, logger)
	if err != nil {
		logger.Error("get-scaling-policy-configuration-from-request", err)
		return result, err
	}
	policyGuidStr := uuid.NewString()

	// // 🚧 To-do: Check if exactly one is provided. We don't want to accept both to be present.
	// requestAppGuid := details.BindResource.AppGuid
	// paramsAppGuid := bindingConfig.Configuration.AppGUID
	var appGUID string
	if details.BindResource != nil {
		appGUID = details.BindResource.AppGuid
	} else {
		// 👎 Access to `details.AppGUID` has been deprecated, see:
		// <https://github.com/openservicebrokerapi/servicebroker/blob/v2.17/spec.md#request-creating-a-service-binding>
		appGUID = details.AppGUID
	}

	// 🚧 To-do: Implement feature: service-key-creation; Use appID from `bindingConfig`!
	if appGUID == "" {
		err := errors.New("error: service must be bound to an application - service key creation is not supported")
		logger.Error("check-required-app-guid", err)
		return result, apiresponses.NewFailureResponseBuilder(
			err, http.StatusUnprocessableEntity, "check-required-app-guid").
			WithErrorKey("RequiresApp").Build()
	}

	// 💡🚧 To-do: We should fail during startup if this does not work. Because then the
	// configuration of the service is corrupted.
	var defaultCustomMetricsCredentialType *models.CustomMetricsBindingAuthScheme
	defaultCustomMetricsCredentialType, err = models.ParseCustomMetricsBindingAuthScheme(
		b.conf.DefaultCustomMetricsCredentialType)
	if err != nil {
		programmingError := &models.InvalidArgumentError{
			Param: "default-credential-type",
			Value: b.conf.DefaultCustomMetricsCredentialType,
			Msg:   "error parsing default credential type",
		}
		logger.Error("parse-default-credential-type", programmingError,
			lager.Data{
				"default-credential-type": b.conf.DefaultCustomMetricsCredentialType,
			})
		return result, apiresponses.NewFailureResponse(err, http.StatusInternalServerError,
			"parse-default-credential-type")
	}
	// 🏚️ Subsequently we assume that this credential-type-configuration is part of the
	// scaling-policy and check it accordingly. However this is legacy and not in line with the
	// current terminology of “PolicyDefinition”, “ScalingPolicy”, “BindingConfig” and
	// “AppScalingConfig”.
	customMetricsBindingAuthScheme, err := getOrDefaultCredentialType(scalingPolicyRaw,
		defaultCustomMetricsCredentialType, logger)
	if err != nil {
		return result, err
	}

	// To-do: 🚧 Factor everything that is involved in this creation out into an own
	// helper-function. Consider a function analogous to `getScalingPolicyFromRequest` that is
	// defined within this file.
	appScalingConfig := models.NewAppScalingConfig(
		*models.NewBindingConfig(models.GUID(appGUID), customMetricsBindingAuthScheme),
		*scalingPolicy)

	if err := b.handleExistingBindingsResiliently(ctx, instanceID, appGUID, logger); err != nil {
		return result, err
	}
	err = createServiceBinding(
		// First time in this function and its recursions, that `instanceID` is used for something
		// different but logging:
		ctx, b.bindingdb, bindingID, instanceID,
		appScalingConfig.GetConfiguration().GetAppGUID(),
		appScalingConfig.GetScalingPolicy().GetCustomMetricsStrategy())

	if err != nil {
		actionCreateServiceBinding := "create-service-binding"
		logger.Error(actionCreateServiceBinding, err)
		if errors.Is(err, db.ErrAlreadyExists) {
			return result, apiresponses.NewFailureResponse(
				errors.New("error: an autoscaler service instance is already bound to the application and multiple bindings are not supported"),
				http.StatusConflict, actionCreateServiceBinding)
		}
		if errors.Is(err, ErrInvalidCustomMetricsStrategy) {
			return result, apiresponses.NewFailureResponse(
				err, http.StatusBadRequest, actionCreateServiceBinding)
		}
		return result, apiresponses.NewFailureResponse(
			ErrCreatingServiceBinding, http.StatusInternalServerError, actionCreateServiceBinding)
	}

	customMetricsCredentials := &models.CustomMetricsCredentials{
		MtlsUrl: b.conf.MetricsForwarder.MetricsForwarderMtlsUrl,
	}

	if customMetricsBindingAuthScheme == &models.BindingSecret {
		// create credentials
		cred, err := b.credentials.Create(ctx, appGUID, nil)
		if err != nil {
			//revert binding creating
			logger.Error("create-credentials", err)

			err = b.bindingdb.DeleteServiceBindingByAppId(ctx, appGUID)
			if err != nil {
				logger.Error("revert-binding-creation-due-to-credentials-creation-failure", err)
			}
			return result, apiresponses.NewFailureResponse(ErrCreatingServiceBinding, http.StatusInternalServerError, "revert-binding-creation-due-to-credentials-creation-failure")
		}
		customMetricsCredentials.URL = &b.conf.MetricsForwarder.MetricsForwarderUrl
		customMetricsCredentials.Credential = cred
	}

	if err := b.attachPolicyOrDefaultPolicyToApp(ctx,
		instanceID, appScalingConfig.GetConfiguration().GetAppGUID(),
		appScalingConfig.GetScalingPolicy().GetPolicyDefinition(), policyGuidStr,
		logger); err != nil {
		return result, err
	}

	result.Credentials = models.Credentials{
		CustomMetrics: *customMetricsCredentials,
	}
	return result, nil
}

func (b *Broker) getScalingPolicyFromRequest(
	scalingPolicyRaw json.RawMessage, logger lager.Logger,
) (*models.ScalingPolicy, error) {
	scalingPolicy, err := models.ScalingPolicyFromRawJSON(scalingPolicyRaw)
	if err != nil {
		actionReadScalingPolicy := "read-scaling-policy"
		logger.Error("unmarshal-scaling-policy", err)
		return nil, apiresponses.NewFailureResponseBuilder(
			ErrInvalidConfigurations, http.StatusBadRequest, actionReadScalingPolicy).
			WithErrorKey(actionReadScalingPolicy).
			Build()
	}
	logger.Debug("getScalingPolicyFromRequest", lager.Data{"scalingPolicy": scalingPolicy})
	return scalingPolicy, nil
}

func getOrDefaultCredentialType(
	policyJson json.RawMessage, defaultCredentialType *models.CustomMetricsBindingAuthScheme,
	logger lager.Logger,
) (*models.CustomMetricsBindingAuthScheme, error) {
	credentialType := defaultCredentialType

	if len(policyJson) > 0 {
		var policy struct {
			CredentialType string `json:"credential-type,omitempty"`
		}
		err := json.Unmarshal(policyJson, &policy)
		if err != nil {
			logger.Error("error: unmarshal-credential-type", err)
			return nil, apiresponses.NewFailureResponse(ErrCreatingServiceBinding,
				http.StatusInternalServerError, "error-unmarshal-credential-type")
		}

		if policy.CredentialType != "" {
			parsedCredentialType, err := models.ParseCustomMetricsBindingAuthScheme(policy.CredentialType)
			if err != nil {
				logger.Error("error: parse-credential-type", err)
				return nil, apiresponses.NewFailureResponseBuilder(
					ErrInvalidCredentialType, http.StatusBadRequest, "error-parse-credential-type").
					WithErrorKey("validate-credential-type").Build()
				// For backwards-compatibility we use "validate-credential-type" here.
			}
			credentialType = parsedCredentialType
		}
	}

	logger.Debug("getOrDefaultCredentialType", lager.Data{"credential-type": credentialType})
	return credentialType, nil
}

func (b *Broker) attachPolicyOrDefaultPolicyToApp(
	ctx context.Context,
	instanceID string, appGUID models.GUID,
	policyDefinition *models.PolicyDefinition, policyGuidStr string,
	logger lager.Logger,
) error {
	logger = logger.Session("saving-policy-json", lager.Data{"policy": policyDefinition})

	if policyProvided := policyDefinition != nil; policyProvided {
		logger.Info("saving policy")

		if err := attachPolicyToApp(ctx, b, appGUID, policyDefinition, policyGuidStr, logger); err != nil {
			return err
		}
	} else {
		// Try to save the default-policy of the service-instance, if any.
		logger.Info("no-policy-json-provided")

		serviceInstance, err := b.bindingdb.GetServiceInstance(ctx, instanceID)
		if err != nil {
			logger.Error("get-service-instance", err)
			return apiresponses.NewFailureResponse(ErrCreatingServiceBinding, http.StatusInternalServerError, "get-service-instance")
		}

		if serviceInstance.DefaultPolicy != "" {
			// 🚧 To-do: Check if we should actually just parse a models.PolicyDefinition here!
			defaultScalingPolicy, err := models.ScalingPolicyFromRawJSON(json.RawMessage(serviceInstance.DefaultPolicy))
			if err != nil {
				errRsp := apiresponses.NewFailureResponse(
					fmt.Errorf("unmarshalling default policy: '%s' failed: %w",
						serviceInstance.DefaultPolicy, err),
					http.StatusInternalServerError, "unmarshal default policy")
				return errRsp
			}
			defaultPolicyGuid := serviceInstance.DefaultPolicyGuid

			if err := attachPolicyToApp(ctx, b, appGUID, defaultScalingPolicy.GetPolicyDefinition(), defaultPolicyGuid, logger); err != nil {
				return err
			}
		} else {
			logger.Info("no-default-policy-to-apply")
		}
	}

	return nil
}

func attachPolicyToApp(
	ctx context.Context, b *Broker, appGUID models.GUID,
	policyDefinition *models.PolicyDefinition, policyGuidStr string,
	logger lager.Logger,
) error {
	if policyDefinition == nil {
		return &models.InvalidArgumentError{
			Param: "policyDefinition",
			Value: nil,
			Msg:   "No policy definition provided",
		}
	}
	appGUIDStr := string(appGUID)

	if err := b.policydb.SaveAppPolicy(ctx, appGUIDStr, policyDefinition, policyGuidStr); err != nil {
		logger.Error("save-appGUID-policy", err)
		//failed to save policy, so revert creating binding and custom metrics credential
		err = b.credentials.Delete(ctx, appGUIDStr)
		if err != nil {
			logger.Error("revert-custom-metrics-credential-due-to-failed-to-save-policy", err)
		}
		err = b.bindingdb.DeleteServiceBindingByAppId(ctx, appGUIDStr)
		if err != nil {
			logger.Error("revert-binding-due-to-failed-to-save-policy", err)
		}
		return apiresponses.NewFailureResponse(
			ErrCreatingServiceBinding, http.StatusInternalServerError, "save-appGUID-policy",
		)
	}

	logger.Info("creating/updating schedules")
	if err := b.schedulerUtil.CreateOrUpdateSchedule(ctx, appGUIDStr, policyDefinition, policyGuidStr); err != nil {
		return fmt.Errorf("attachPolicyToApp failed update/create: %w", err)
	}

	return nil
}

func (b *Broker) handleExistingBindingsResiliently(ctx context.Context, instanceID string, appGUID string, logger lager.Logger) error {
	// fetch and all service bindings for the service instance
	logger = logger.Session("handleExistingBindingsResiliently", lager.Data{"app_id": appGUID, "instance_id": instanceID})
	bindingIds, err := b.bindingdb.GetBindingIdsByInstanceId(ctx, instanceID)
	if err != nil {
		logger.Error("get-existing-service-bindings-before-binding", err)
		return apiresponses.NewFailureResponse(ErrCreatingServiceBinding, http.StatusInternalServerError, "get-existing-service-bindings-before-binding")
	}

	for _, existingBindingId := range bindingIds {
		// get the service binding for the appGUID
		fetchedAppID, err := b.bindingdb.GetAppIdByBindingId(ctx, existingBindingId)
		if err != nil {
			logger.Error("Binding does not belong to app", err, lager.Data{"existingBindingID": existingBindingId, "fetched_app_id": fetchedAppID})
			return apiresponses.NewFailureResponse(ErrCreatingServiceBinding, http.StatusInternalServerError, "get-existing-service-binding-before-binding")
		}

		//select the binding-id for the appGUID
		if fetchedAppID == appGUID {
			err = b.deleteBinding(ctx, existingBindingId, instanceID)

			if err != nil {
				logger.Error("failed to deleteBinding", err, lager.Data{"existingBindingID": existingBindingId})
				if errors.Is(err, ErrDeleteServiceBinding) ||
					errors.Is(err, ErrDeletePolicyForUnbinding) ||
					errors.Is(err, ErrDeleteSchedulesForUnbinding) ||
					errors.Is(err, ErrCredentialNotDeleted) {
					return apiresponses.NewFailureResponse(
						ErrCreatingServiceBinding, http.StatusInternalServerError,
						"delete-existing-service-binding-before-binding")
				}
			}
		}
	}
	return nil
}

// Unbind deletes an existing service binding
// DELETE /v2/service_instances/{instance_id}/service_bindings/{binding_id}
func (b *Broker) Unbind(ctx context.Context, instanceID string, bindingID string, details domain.UnbindDetails, _ bool) (domain.UnbindSpec, error) {
	logger := b.logger.Session("unbind", lager.Data{"instanceID": instanceID, "bindingID": bindingID, "unbindDetails": details})
	logger.Info("begin")
	defer logger.Info("end")

	result := domain.UnbindSpec{}
	err := b.deleteBinding(ctx, bindingID, instanceID)
	if err != nil {
		logger.Error("unbind failed", err)
		if errors.Is(err, ErrBindingDoesNotExist) {
			return result, apiresponses.ErrBindingDoesNotExist
		}
		return result, apiresponses.NewFailureResponse(fmt.Errorf("unbind failed: %w", ErrDeleteServiceBinding), http.StatusInternalServerError, "delete-binding")
	}
	return result, nil
}

// GetBinding fetches an existing service binding
// GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}
func (b *Broker) GetBinding(ctx context.Context, instanceID string, bindingID string, details domain.FetchBindingDetails) (domain.GetBindingSpec, error) {
	logger := b.logger.Session("get-binding",
		lager.Data{"instanceID": instanceID, "bindingID": bindingID, "fetchBindingDetails": details})
	logger.Info("begin")
	defer logger.Info("end")

	result := domain.GetBindingSpec{}
	serviceBinding, err := b.getServiceBinding(ctx, bindingID)
	if err != nil {
		return result, err
	}

	policyDef, err := b.policydb.GetAppPolicy(ctx, serviceBinding.AppID)
	if err != nil {
		b.logger.Error("get-binding", err, lager.Data{"instanceID": instanceID, "bindingID": bindingID, "fetchBindingDetails": details})
		return domain.GetBindingSpec{}, apiresponses.NewFailureResponse(errors.New("failed to retrieve scaling policy"), http.StatusInternalServerError, "get-policy")
	}
	customMetricStrategy, err := models.ParseCustomMetricsStrategy(serviceBinding.CustomMetricsStrategy)
	if err != nil {
		// This should never happen, as we validate this on creation of the binding.
		b.logger.Error("parser-custom-metrics-strategy", err,
			lager.Data{"instanceID": instanceID, "bindingID": bindingID, "fetchBindingDetails": details})
	}

	scalingPolicy := models.NewScalingPolicy(*customMetricStrategy, policyDef)
	if !scalingPolicy.IsDefaultScalingPolicy() {
		// The scaling policy should only be serialised into the response if it differs in some way
		// from the default-value.
		result.Parameters = scalingPolicy
	}

	return result, nil
}

func (b *Broker) getServiceBinding(ctx context.Context, bindingID string) (*models.ServiceBinding, error) {
	logger := b.logger.Session("get-service-binding", lager.Data{"bindingID": bindingID})

	serviceBinding, err := b.bindingdb.GetServiceBinding(ctx, bindingID)
	if err != nil {
		if errors.Is(err, db.ErrDoesNotExist) {
			logger.Error("failed to find service binding", err)
			return nil, apiresponses.ErrBindingDoesNotExist
		} else {
			logger.Error("failed to retrieve service binding", err)
			return nil, apiresponses.NewFailureResponse(errors.New("failed to retrieve service binding"), http.StatusInternalServerError, "retrieving-service-binding")
		}
	}
	return serviceBinding, nil
}

// LastBindingOperation fetches last operation state for a service binding
// GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation
func (b *Broker) LastBindingOperation(_ context.Context, instanceID string, bindingID string, details domain.PollDetails) (domain.LastOperation, error) {
	logger := b.logger.Session("last-binding-operation", lager.Data{"instanceID": instanceID, "bindingID": bindingID, "pollDetails": details})
	logger.Info("begin")
	defer logger.Info("end")

	err := errors.New("error: last-binding-operation is not implemented and this endpoint should not have been called as all broker operations are synchronous")
	logger.Error("last-binding-operation-is-not-implemented", err)
	return domain.LastOperation{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, "last-binding-operation-is-not-implemented")
}

func (b *Broker) planDefinitionExceeded(policy *models.PolicyDefinition, planID string, instanceID string) error {
	ok, checkResult, err := b.PlanChecker.CheckPlan(policy, planID)
	if err != nil {
		b.logger.Error("failed to check policy for plan adherence", err, lager.Data{"instanceID": instanceID, "policy": policy})
		return apiresponses.NewFailureResponse(errors.New("error validating policy"), http.StatusInternalServerError, "failed to check policy for plan adherence")
	}
	if !ok {
		b.logger.Error("policy did not adhere to plan", errors.New(checkResult), lager.Data{"instanceID": instanceID, "policy": policy})
		return apiresponses.NewFailureResponse(fmt.Errorf("error: policy did not adhere to plan: %s", checkResult), http.StatusBadRequest, "policy did not adhere to plan")
	}
	return nil
}

func (b *Broker) getService(serviceID string) (domain.Service, error) {
	serviceIndex := slices.IndexFunc(b.catalog, func(s domain.Service) bool { return s.ID == serviceID })
	if serviceIndex == -1 {
		return domain.Service{}, apiresponses.NewFailureResponse(fmt.Errorf("error: unknown service with GUID '%s'specified", serviceID), http.StatusBadRequest, "retrieving-service")
	}
	return b.catalog[serviceIndex], nil
}

func (b *Broker) getServicePlan(serviceID string, planID string) (domain.ServicePlan, error) {
	service, err := b.getService(serviceID)
	if err != nil {
		return domain.ServicePlan{}, err
	}

	planIndex := slices.IndexFunc(service.Plans, func(s domain.ServicePlan) bool { return s.ID == planID })
	if planIndex == -1 {
		return domain.ServicePlan{}, apiresponses.NewFailureResponse(fmt.Errorf("error: unknown service plan with GUID '%s' specified", planID), http.StatusBadRequest, "retrieving-service-plan")
	}
	return service.Plans[planIndex], nil
}

func (b *Broker) getExistingOrUpdatedServicePlan(instanceID string, updateDetails domain.UpdateDetails) (string, bool, error) {
	existingServicePlan := updateDetails.PreviousValues.PlanID
	updateToPlan := updateDetails.PlanID

	servicePlan := existingServicePlan
	servicePlanIsNew := false

	var brokerErr error
	if updateToPlan != "" {
		if _, err := b.getServicePlan(updateDetails.ServiceID, updateToPlan); err != nil {
			return "", false, err
		}

		servicePlanIsNew = servicePlan != updateToPlan
		servicePlan = updateToPlan
		if existingServicePlan != updateToPlan {
			isPlanUpdatable, err := b.PlanChecker.IsPlanUpdatable(existingServicePlan)
			if err != nil {
				b.logger.Error("checking-broker-plan-updatable", err, lager.Data{"instanceID": instanceID, "existingServicePlan": existingServicePlan, "newServicePlan": updateToPlan})
				brokerErr = apiresponses.NewFailureResponse(errors.New("error checking if the broker plan is updatable"), http.StatusInternalServerError, "checking-broker-plan-updatable")
			} else if !isPlanUpdatable {
				b.logger.Info("plan-not-updatable", lager.Data{"instanceID": instanceID, "existingServicePlan": existingServicePlan, "newServicePlan": updateToPlan})
				brokerErr = apiresponses.ErrPlanChangeNotSupported
			}
		}
	}

	return servicePlan, servicePlanIsNew, brokerErr
}

func GetDashboardURL(conf *config.Config, instanceID string) string {
	if conf.DashboardRedirectURI != "" {
		return fmt.Sprintf("%s/manage/%s", conf.DashboardRedirectURI, instanceID)
	}
	return ""
}

func (b *Broker) deleteBinding(ctx context.Context, bindingId string, serviceInstanceId string) error {
	appId, err := b.bindingdb.GetAppIdByBindingId(ctx, bindingId)
	logger := b.logger.Session("deleteBinding", lager.Data{"app_id": appId, "binding_id": bindingId, "service_instance": serviceInstanceId})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("missing binding: %w", ErrBindingDoesNotExist)
		}
		return fmt.Errorf("failed to get biding info: %w", ErrDeleteServiceBinding)
	}

	logger.Info("deleting policy json")
	err = b.policydb.DeletePolicy(ctx, appId)
	if err != nil {
		logger.Error("failed to delete policy for unbinding", err)
		return ErrDeletePolicyForUnbinding
	}

	logger.Info("deleting schedules")
	err = b.schedulerUtil.DeleteSchedule(ctx, appId)
	if err != nil {
		logger.Info("failed to delete schedules for unbinding")
		return ErrDeleteSchedulesForUnbinding
	}
	err = b.bindingdb.DeleteServiceBinding(ctx, bindingId)
	if err != nil {
		logger.Error("failed to delete binding", err)
		if errors.Is(err, db.ErrDoesNotExist) {
			return ErrBindingDoesNotExist
		}

		return ErrDeleteServiceBinding
	}

	err = b.credentials.Delete(ctx, appId)
	if err != nil {
		logger.Error("failed to delete custom metrics credential for unbinding", err)
		return ErrCredentialNotDeleted
	}
	return nil
}

func createServiceBinding(
	ctx context.Context,
	bindingDB db.BindingDB, bindingID, instanceID string,
	appGUID models.GUID, customMetricsStrategy models.CustomMetricsStrategy,
) error {
	switch customMetricsStrategy {
	case models.CustomMetricsBoundApp, models.CustomMetricsSameApp:
		err := bindingDB.CreateServiceBinding(ctx, bindingID, instanceID, appGUID, customMetricsStrategy)
		if err != nil {
			return err
		}
	default:
		return ErrInvalidCustomMetricsStrategy
	}

	return nil
}
