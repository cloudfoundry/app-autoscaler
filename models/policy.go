package models

import (
	"bytes"
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
//     Pass nil to use the default policy.
//   - customMetricsStrategy: The strategy for custom metrics submission, which determines which
//     app submits custom metrics.
//
// Returns:
//   - *ScalingPolicy: A properly initialized ScalingPolicy instance
func NewScalingPolicy(
	customMetricsStrategy CustomMetricsStrategy,
	policyDefinition *PolicyDefinition,
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

// GetCustomMetricsStrategy returns the custom metrics strategy configured for this scaling policy.
// This determines which applications are allowed to submit custom metrics for scaling decisions.
//
// The strategy can be either "bound_app" (only the bound application can submit metrics) or
// "same_app" (default strategy where the application submits its own metrics).
//
// Returns:
//
//	CustomMetricsStrategy: The configured strategy for custom metrics submission
func (sp ScalingPolicy) GetCustomMetricsStrategy() CustomMetricsStrategy {
	return sp.policyCfg.CustomMetricsCfg.MetricSubmissionStrategy.AllowFrom
}

// GetPolicyDefinition returns the scaling policy for the binding and nil if no one has been set (which
// means, the default-policy is used).
func (sp *ScalingPolicy) GetPolicyDefinition() (p *PolicyDefinition) {
	if sp.useDefaultPolicyDef {
		p = nil // No scaling policy has been set, so we return nil.
	} else {
		p = &sp.policyDef
	}

	return p
}

func (sp *ScalingPolicy) IsDefaultScalingPolicy() bool {
	return sp.useDefaultPolicyDef && sp.policyCfg == defaultPolicyConfiguration()
}

// -------------------- Deserialisation and serialisation --------------------

// Json-serialized example of a ScalingPolicy:
// {
//   "configuration": {
//     "custom_metrics": {
//       "metric_submission_strategy": {
//         "allow_from": "bound_app"
//       }
//     }
//   },
//   "instance_min_count": 1,
//   "instance_max_count": 5,
//   "scaling_rules": [
//     {
//       "metric_type": "memoryused",
//       "threshold": 30,
//       "operator": "<",
//       "adjustment": "-1"
//     }
//   ]
// }

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

	var data json.RawMessage
	// If both fields are nil (i.e. they represent just the default), we return an empty JSON
	// object.
	definitionAndConfigAreBothDefault := spRaw.PolicyConfiguration == nil && spRaw.PolicyDefinition == nil
	if definitionAndConfigAreBothDefault {
		data = nil
	} else {
		var err error
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		encoder.SetEscapeHTML(false)

		err = encoder.Encode(spRaw)
		if err != nil {
			return nil, err
		}

		data = json.RawMessage(bytes.TrimSpace(buf.Bytes()))
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

func (sp ScalingPolicy) MarshalJSON() ([]byte, error) {
	rawJson, err := sp.ToRawJSON()
	if err != nil {
		return nil, err
	} else if rawJson == nil {
		// If the rawJson is nil, we return an empty JSON object.
		rawJson = []byte("{}")
	}

	return []byte(rawJson), nil
}

// ================================================================================
// policyConfiguration
// ================================================================================

type policyConfiguration struct {
	CustomMetricsCfg customMetricsConfig `json:"custom_metrics,omitempty"` // nil value represents null-value (i.e. not set).
}

type customMetricsConfig struct {
	MetricSubmissionStrategy metricsSubmissionStrategy `json:"metric_submission_strategy"`
}

type metricsSubmissionStrategy struct {
	AllowFrom CustomMetricsStrategy `json:"allow_from"`
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

// CustomMetricsStrategy defines the strategy for submitting custom metrics. It can be either
// "bound_app" or "same_app".
//
// ⛔ Do not create CustomMetricsStrategy values directly via `CustomMetricsStrategy{}` because it
// can lead to undefined behaviour due to bypassing all validations.  Use the predefined constants
// instead.
type CustomMetricsStrategy struct {
	value string // Not exported to prohibit construction of CustomMetricsStrategy values outside
	// this package.
}

var (
	CustomMetricsBoundApp = CustomMetricsStrategy{"bound_app"}

	// CustomMetricsSameApp default value if not specified
	CustomMetricsSameApp         = CustomMetricsStrategy{"same_app"}
	DefaultCustomMetricsStrategy = CustomMetricsSameApp
)

func (s CustomMetricsStrategy) String() string {
	return s.value
}

var _ fmt.Stringer = CustomMetricsStrategy{}

func ParseCustomMetricsStrategy(value string) (*CustomMetricsStrategy, error) {
	switch value {
	case "bound_app":
		return &CustomMetricsBoundApp, nil
	case "same_app":
		return &CustomMetricsSameApp, nil
	case "":
		return &DefaultCustomMetricsStrategy, nil
	default:
		return nil, fmt.Errorf("unsupported CustomMetricsStrategy: %s", value)
	}
}

func (s CustomMetricsStrategy) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.value)
}

func (s *CustomMetricsStrategy) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	switch value {
	case "bound_app":
		*s = CustomMetricsBoundApp
	case "same_app":
		*s = CustomMetricsSameApp
	default:
		return fmt.Errorf("unsupported CustomMetricsStrategy: %s", value)
	}

	return nil
}

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

func (pd PolicyDefinition) ToRawJSON() (json.RawMessage, error) {
	var err error
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)

	err = encoder.Encode(pd)
	if err != nil {
		return nil, err
	}

	data := json.RawMessage(bytes.TrimSpace(buf.Bytes()))

	return data, nil
}

func FromRawJSON(data json.RawMessage) (PolicyDefinition, error) {
	var scalingPolicy PolicyDefinition
	if err := json.Unmarshal(data, &scalingPolicy); err != nil {
		return PolicyDefinition{}, fmt.Errorf("failed to unmarshal ScalingPolicy: %w", err)
	}
	return scalingPolicy, nil
}

func (pd PolicyDefinition) String() string {
	aJson, _ := pd.ToRawJSON()
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
