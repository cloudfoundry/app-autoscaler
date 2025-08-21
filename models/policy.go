package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// ================================================================================
// ScalingPolicy
// ================================================================================

// -------------------- Definition and functions --------------------

// A `ScalingPolicy` contains all customer-facing information that determines how an application is
// scaled by the autoscaler.
//
// ⛔ Do not create `ScalingPolicy` values directly via `ScalingPolicy{}` because it can lead to
// undefined behaviour due to bypassing all validations. Use the constructor-functions instead!
type ScalingPolicy struct {
	policyCfg policyConfiguration

	// True if and only if there has not been set any policy for that app. If true, then the content
	// of the field `policyDef` is meaningless.
	useDefaultPolicyDef bool

	policyDef PolicyDefinition
}

// NewScalingPolicy creates a new ScalingPolicy with the provided policy definition.
//
// Parameters:
//   - policyDefinition: The PolicyDefinition that defines the scaling behavior for the application.
//                       Pass nil to use the default policy.
//   - customMetricsStrategy: The strategy for custom metrics submission, which determines which
//     app submits custom metrics.
//
// Returns:
//   - *ScalingPolicy: A properly initialized ScalingPolicy instance
func NewScalingPolicy(
	customMetricsStrategy CustomMetricsStrategy, policyDefinition *PolicyDefinition,
) (scalingPolicy *ScalingPolicy) {
	pCfg := policyConfiguration{
		CustomMetricsCfg: customMetricsConfig{
			MetricSubmissionStrategy: metricsSubmissionStrategy{
				AllowFrom: customMetricsStrategy,
			},
		},
	}

	if policyDefinition == nil {
		scalingPolicy = &ScalingPolicy{
			policyCfg:           pCfg,
			useDefaultPolicyDef: true,
			policyDef:           PolicyDefinition{}, // Content does not matter
		}
	} else {
		scalingPolicy = &ScalingPolicy{
			policyCfg:           pCfg,
			useDefaultPolicyDef: false,
			policyDef:           *policyDefinition,
		}
	}

	return scalingPolicy
}

// GetScalingPolicy returns the scaling policy for the binding and nil if no one has been set (which
// means, the default-policy is used).
func (sp *ScalingPolicy) GetScalingPolicy() (p *PolicyDefinition) {
	if sp.useDefaultPolicyDef {
		p = nil // No scaling policy has been set, so we return nil.
	} else {
		p = &sp.policyDef
	}

	return p
}

// -------------------- Deserialisation and serialisation --------------------

type scalingPolicyJsonRawRepr struct {
	PolicyConfiguration *policyConfiguration `json:"configuration,omitempty"`
	*PolicyDefinition
}

func (sp ScalingPolicy) ToRawJSON() (json.RawMessage, error) {
	var policyCfg *policyConfiguration
	if sp.policyCfg == defaultPolicyConfiguration() {
		// If the policy configuration is the default one, we do not serialize it.
		policyCfg = nil
	} else {
		policyCfg = &sp.policyCfg
	}

	var policy *PolicyDefinition
	if sp.useDefaultPolicyDef {
		policy = nil // ScalingPolicy{} // Gets not serialized, which is equivalent to null in JSON.
	} else {
		policy = &sp.policyDef
	}

	spRaw := scalingPolicyJsonRawRepr{
		PolicyConfiguration: policyCfg,
		PolicyDefinition:    policy,
	}

	data, err := json.Marshal(spRaw)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func ScalingPolicyFromRawJSON(data json.RawMessage) (*ScalingPolicy, error) {
	// If the data is nil, we return a default ScalingPolicy with default configuration.
	if len(data) <= 0 {
		return NewScalingPolicy(DefaultCustomMetricsStrategy, nil), nil
	}

	var spRaw scalingPolicyJsonRawRepr
	if err := json.Unmarshal(data, &spRaw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ScalingPolicy: %w", err)
	}

	var cms CustomMetricsStrategy
	if spRaw.PolicyConfiguration == nil {
		cms = DefaultCustomMetricsStrategy // Default strategy if not set.
	} else {
		cms = spRaw.PolicyConfiguration.CustomMetricsCfg.MetricSubmissionStrategy.AllowFrom
	}

	return NewScalingPolicy(cms, spRaw.PolicyDefinition), nil
}

// ================================================================================
// policyConfiguration
// ================================================================================

type policyConfiguration struct {
	CustomMetricsCfg customMetricsConfig `json:"custom_metrics,omitempty"` // nil value represents null-value (i.e. not set).
}

func defaultPolicyConfiguration() policyConfiguration {
	return policyConfiguration{
		CustomMetricsCfg: customMetricsConfig{
			MetricSubmissionStrategy: metricsSubmissionStrategy{
				AllowFrom: DefaultCustomMetricsStrategy,
			},
		},
	}
}

// func (pc policyConfiguration) ToRawJSON() (json.RawMessage, error) {
//	data, err := json.Marshal(pc)
//	if err != nil {
//		return nil, fmt.Errorf("failed to marshal policyConfiguration: %w", err)
//	}
//	return json.RawMessage(data), nil
// }

// func policyConfigurationFromRawJSON(data json.RawMessage) (policyConfiguration, error) {
//	var pcRaw policyConfiguration
//	if err := json.Unmarshal(data, &pcRaw); err != nil {
//		return policyConfiguration{}, fmt.Errorf("failed to unmarshal policyConfiguration: %w", err)
//	}
//	return pcRaw, nil
// }



// ================================================================================
// PolicyDefinition
// ================================================================================

// 🚧 To-do: Once we switch to the parsers in the package `binding_request`, we can remove the
// json-(de)serialisation-instructions for `PolicyDefinition` throughout the file.

// `PolicyDefinition` is a customer facing entity and represents the scaling policy for an application.
// It can be created/deleted/retrieved by the user via the binding process and public api. If a change is required in the policy,
// the corresponding endpoints should be also be updated in the public api server.
type PolicyDefinition struct {
	InstanceMin  int               `json:"instance_min_count"`
	InstanceMax  int               `json:"instance_max_count"`
	ScalingRules []*ScalingRule    `json:"scaling_rules,omitempty"`
	Schedules    *ScalingSchedules `json:"schedules,omitempty"`
}

func (s PolicyDefinition) ToRawJSON() (json.RawMessage, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func FromRawJSON(data json.RawMessage) (PolicyDefinition, error) {
	var scalingPolicy PolicyDefinition
	if err := json.Unmarshal(data, &scalingPolicy); err != nil {
		return PolicyDefinition{}, fmt.Errorf("failed to unmarshal ScalingPolicy: %w", err)
	}
	return scalingPolicy, nil
}

func (s PolicyDefinition) String() string {
	aJson, _ := s.ToRawJSON()
	return string(aJson)
}

var _ fmt.Stringer = &PolicyDefinition{}

type ScalingRule struct {
	MetricType            string `json:"metric_type"`
	BreachDurationSeconds int    `json:"breach_duration_secs,omitempty"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	CoolDownSeconds       int    `json:"cool_down_secs,omitempty"`
	Adjustment            string `json:"adjustment"`
}

type ScalingSchedules struct {
	Timezone              string                  `json:"timezone"`
	RecurringSchedules    []*RecurringSchedule    `json:"recurring_schedule,omitempty"`
	SpecificDateSchedules []*SpecificDateSchedule `json:"specific_date,omitempty"`
}

func (s *ScalingSchedules) IsEmpty() bool {
	switch {
	case s == nil:
		return true
	case len(s.SpecificDateSchedules) != 0:
		return false
	case len(s.RecurringSchedules) != 0:
		return false
	default:
		return true
	}
}

type RecurringSchedule struct {
	StartTime             string `json:"start_time"`
	EndTime               string `json:"end_time"`
	DaysOfWeek            []int  `json:"days_of_week,omitempty"`
	DaysOfMonth           []int  `json:"days_of_month,omitempty"`
	StartDate             string `json:"start_date,omitempty"`
	EndDate               string `json:"end_date,omitempty"`
	ScheduledInstanceMin  int    `json:"instance_min_count"`
	ScheduledInstanceMax  int    `json:"instance_max_count"`
	ScheduledInstanceInit int    `json:"initial_min_instance_count,omitempty"`
}

type SpecificDateSchedule struct {
	StartDateTime         string `json:"start_date_time"`
	EndDateTime           string `json:"end_date_time"`
	ScheduledInstanceMin  int    `json:"instance_min_count"`
	ScheduledInstanceMax  int    `json:"instance_max_count"`
	ScheduledInstanceInit int    `json:"initial_min_instance_count,omitempty"`
}

func (r *ScalingRule) BreachDuration(defaultBreachDurationSecs int) time.Duration {
	if r.BreachDurationSeconds <= 0 {
		return time.Duration(defaultBreachDurationSecs) * time.Second
	}
	return time.Duration(r.BreachDurationSeconds) * time.Second
}

func (r *ScalingRule) CoolDown(defaultCoolDownSecs int) time.Duration {
	if r.CoolDownSeconds <= 0 {
		return time.Duration(defaultCoolDownSecs) * time.Second
	}
	return time.Duration(r.CoolDownSeconds) * time.Second
}

type Trigger struct {
	AppId                 string `json:"app_id"`
	MetricType            string `json:"metric_type"`
	MetricUnit            string `json:"metric_unit"`
	BreachDurationSeconds int    `json:"breach_duration_secs"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	CoolDownSeconds       int    `json:"cool_down_secs"`
	Adjustment            string `json:"adjustment"`
}

func (t Trigger) BreachDuration() time.Duration {
	return time.Duration(t.BreachDurationSeconds) * time.Second
}

func (t Trigger) CoolDown(defaultCoolDownSecs int) time.Duration {
	if t.CoolDownSeconds <= 0 {
		return time.Duration(defaultCoolDownSecs) * time.Second
	}
	return time.Duration(t.CoolDownSeconds) * time.Second
}

type ActiveSchedule struct {
	ScheduleId         string
	InstanceMin        int `json:"instance_min_count"`
	InstanceMax        int `json:"instance_max_count"`
	InstanceMinInitial int `json:"initial_min_instance_count"`
}

// ================================================================================
// 🏚️ Legacy-definitions
// ================================================================================

type AppPolicy struct {
	AppId         string
	ScalingPolicy *PolicyDefinition
}

type PolicyJson struct {
	AppId     string
	PolicyStr string
}

func (p *PolicyJson) Equals(p2 *PolicyJson) bool {
	if p == p2 {
		return true
	} else if p != nil && p2 != nil {
		if p.AppId == p2.AppId && p.PolicyStr == p2.PolicyStr {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}
func (p *PolicyJson) GetAppPolicy() (*AppPolicy, error) {
	scalingPolicy := PolicyDefinition{}
	err := json.Unmarshal([]byte(p.PolicyStr), &scalingPolicy)
	if err != nil {
		return nil, fmt.Errorf("policy unmarshalling failed %w", err)
	}
	return &AppPolicy{AppId: p.AppId, ScalingPolicy: &scalingPolicy}, nil
}
