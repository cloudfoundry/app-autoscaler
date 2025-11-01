package bindingrequestparser

import (
	"encoding/json"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/policyvalidator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/brokerapi/v13/domain"
)

// ================================================================================
// Parser-interface
// ================================================================================

type BindRequestParser = interface {
	Parse(details domain.BindDetails) (models.AppScalingConfig, error)
}

// This error type is used, when the passed binding-request fails to validate against the schema.
type BindReqNoAppGuid struct{} // 🚧 To-do: Generalise for every appGuidError (as well if two are provided)

func (e BindReqNoAppGuid) Error() string {
	return "error: service must be bound to an application; Please provide a GUID of an app!"
}

var _ error = BindReqNoAppGuid{}

// ================================================================================
// Parser-implementation
// ================================================================================

type bindRequestParser struct {
	policyValidator                    *policyvalidator.PolicyValidator
	defaultCustomMetricsCredentialType models.CustomMetricsBindingAuthScheme
}

var _ BindRequestParser = &bindRequestParser{
	policyValidator: policyvalidator.NewPolicyValidator(
		"🚸 This is a dummy and never executed!", 0, 0, 0, 0, 0, 0, 0, 0,
	)}

func NewBindRequestParser(policyValidator *policyvalidator.PolicyValidator, defaultCredentialType models.CustomMetricsBindingAuthScheme) BindRequestParser {
	return &bindRequestParser{
		policyValidator:                    policyValidator,
		defaultCustomMetricsCredentialType: defaultCredentialType,
	}
}

func (brp *bindRequestParser) Parse(details domain.BindDetails) (models.AppScalingConfig, error) {
	var scalingPolicyRaw json.RawMessage
	if details.RawParameters != nil {
		scalingPolicyRaw = details.RawParameters
	}

	var err error

	// This just gets used for legacy-reasons. The actually parsing happens in the step
	// afterwards. But it still does not validate against the schema, which is done here.
	_, err = brp.getPolicyFromJsonRawMessage(scalingPolicyRaw)
	if err != nil {
		err := fmt.Errorf("validation-error against the json-schema:\n\t%w", err)
		return models.AppScalingConfig{}, err
	}

	scalingPolicy, err := models.ScalingPolicyFromRawJSON(scalingPolicyRaw)
	if err != nil {
		err := fmt.Errorf("could not parse scaling policy from request:\n\t%w", err)
		return models.AppScalingConfig{}, err
		// // 🚧 ⚠️ I need to be run on the receiver-side.
		// return nil, apiresponses.NewFailureResponseBuilder(
		//	ErrInvalidConfigurations, http.StatusBadRequest, actionReadScalingPolicy).
		//	WithErrorKey(actionReadScalingPolicy).
		//	Build()
	}

	// 🏚️ Subsequently we assume that this credential-type-configuration is part of the
	// scaling-policy and check it accordingly. However this is legacy and not in line with the
	// current terminology of “PolicyDefinition”, “ScalingPolicy”, “BindingConfig” and
	// “AppScalingConfig”.
	customMetricsBindingAuthScheme, err := brp.getOrDefaultCredentialType(scalingPolicyRaw)
	if err != nil {
		return models.AppScalingConfig{}, err
	}

	// 🏚️ Subsequently we assume that this app-guid is part of the
	// scaling-policy and check it accordingly. However this is legacy and not in line with the
	// current terminology of “PolicyDefinition”, “ScalingPolicy”, “BindingConfig” and
	// “AppScalingConfig”.
	appGuidFromBindingConfig, err := brp.getAppGuidFromBindingConfig(scalingPolicyRaw)
	if err != nil {
		return models.AppScalingConfig{}, err
	}

	var appGuid models.GUID
	appGuidIsFromCC := details.BindResource != nil && details.BindResource.AppGuid != ""
	appGuidIsFromCCDeprField := details.AppGUID != ""
	appGuidIsFromBindingConfig := appGuidFromBindingConfig != ""
	switch {
	case (appGuidIsFromCC || appGuidIsFromCCDeprField) && appGuidIsFromBindingConfig:
		msg := "error: app GUID provided in both, binding resource and binding configuration"
		err := fmt.Errorf("%s:\n\tfrom binding-request: %s", msg, appGuidFromBindingConfig)
		return models.AppScalingConfig{}, err
	case appGuidIsFromCC:
		appGuid = models.GUID(details.BindResource.AppGuid)
	case appGuidIsFromCCDeprField:
		// 👎 Access to `details.AppGUID` has been deprecated, see:
		// <https://github.com/openservicebrokerapi/servicebroker/blob/v2.17/spec.md#request-creating-a-service-binding>
		appGuid = models.GUID(details.AppGUID)
	case appGuidIsFromBindingConfig:
		appGuid = appGuidFromBindingConfig
	default:
		return models.AppScalingConfig{}, &BindReqNoAppGuid{}
	}

	appScalingConfig := models.NewAppScalingConfig(
		*models.NewBindingConfig(appGuid, customMetricsBindingAuthScheme),
		*scalingPolicy)

	return *appScalingConfig, nil
}

func (brp *bindRequestParser) getPolicyFromJsonRawMessage(policyJson json.RawMessage) (*models.ScalingPolicy, error) {
	if isEmptyPolicy := len(policyJson) <= 0; isEmptyPolicy { // no nil-check needed: `len(nil) == 0`
		return nil, nil
	}

	policy, errResults := brp.policyValidator.ParseAndValidatePolicy(policyJson)
	if len(errResults) > 0 {
		return policy, &errResults
	}

	return policy, nil
}

func (brp *bindRequestParser) getOrDefaultCredentialType(
	policyJson json.RawMessage,
) (*models.CustomMetricsBindingAuthScheme, error) {
	credentialType := &brp.defaultCustomMetricsCredentialType

	if len(policyJson) > 0 {
		var policy struct {
			CredentialType string `json:"credential-type,omitempty"`
		}
		err := json.Unmarshal(policyJson, &policy)
		if err != nil {
			// 🚸 This can not happen because the input at this point has already been checked
			// against the json-schema.
			return nil, fmt.Errorf("could not parse scaling policy to get credential type: %w", err)
		}

		if policy.CredentialType != "" {
			parsedCredentialType, err := models.ParseCustomMetricsBindingAuthScheme(policy.CredentialType)
			if err != nil {
				// 🚸 This can not happen because the input at this point has already been checked
				// against the json-schema.
				return nil, fmt.Errorf("could not parse credential type from scaling policy: %w", err)
			}
			credentialType = parsedCredentialType
		}
	}

	return credentialType, nil
}

func (brp *bindRequestParser) getAppGuidFromBindingConfig(policyJson json.RawMessage) (models.GUID, error) {
	if len(policyJson) <= 0 {
		return "", nil
	}

	var appGuidRawStruct struct {
		BindingConfig struct {
			AppGUID string `json:"app_guid,omitempty"`
		} `json:"configuration,omitempty"`
	}
	err := json.Unmarshal(policyJson, &appGuidRawStruct)
	if err != nil {
		// 🚸 This can not happen because the input at this point has already been checked
		// against the json-schema.
		return "", fmt.Errorf("could not parse scaling policy to get app-guid from binding-configuration: %w", err)
	}

	appGuid := models.GUID(appGuidRawStruct.BindingConfig.AppGUID)

	return appGuid, nil
}
