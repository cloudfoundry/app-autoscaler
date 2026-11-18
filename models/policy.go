package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
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
	policyCfg        policyConfiguration
	rawConfiguration json.RawMessage

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
	object := map[string]json.RawMessage{}
	if !sp.useDefaultPolicyDef {
		data, err := sp.policyDef.ToRawJSON()
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, err
		}
	}
	if len(sp.rawConfiguration) != 0 {
		object["configuration"] = sp.rawConfiguration
	} else if sp.policyCfg != defaultPolicyConfiguration() {
		data, err := json.Marshal(sp.policyCfg)
		if err != nil {
			return nil, err
		}
		object["configuration"] = data
	}
	if len(object) == 0 {
		return nil, nil
	}
	return encodePolicyJSON(object)
}

func ScalingPolicyFromRawJSON(data json.RawMessage) (*ScalingPolicy, error) {
	if len(data) == 0 {
		return NewScalingPolicy(DefaultCustomMetricsStrategy, nil), nil
	}
	// Decode configuration separately: an embedded PolicyDefinition's JSON methods would
	// otherwise consume the configuration as well.
	var spRaw struct {
		PolicyConfiguration *policyConfiguration `json:"configuration"`
	}
	if err := json.Unmarshal(data, &spRaw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ScalingPolicy: %w", err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, err
	}
	var definition *PolicyDefinition
	if _, present := object["instance_min_count"]; present {
		definition = &PolicyDefinition{}
		if err := json.Unmarshal(data, definition); err != nil {
			return nil, err
		}
	}

	var cms CustomMetricsStrategy
	if spRaw.PolicyConfiguration == nil ||
		spRaw.PolicyConfiguration.CustomMetricsCfg.MetricSubmissionStrategy.AllowFrom.value == "" {
		cms = DefaultCustomMetricsStrategy // Default strategy if not set.
	} else {
		cms = spRaw.PolicyConfiguration.CustomMetricsCfg.MetricSubmissionStrategy.AllowFrom
	}

	policy := NewScalingPolicy(cms, definition)
	// Preserve extension fields in configuration without retaining binding-only app_guid.
	if raw := object["configuration"]; len(raw) != 0 {
		var configuration map[string]json.RawMessage
		if err := json.Unmarshal(raw, &configuration); err != nil {
			return nil, err
		}
		delete(configuration, "app_guid")
		canonical, _ := json.Marshal(policy.policyCfg)
		var known map[string]any
		var actual map[string]any
		_ = json.Unmarshal(canonical, &known)
		cleaned, err := encodePolicyJSON(configuration)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal(cleaned, &actual)
		if configurationHasExtensions(actual, known) {
			policy.rawConfiguration = cleaned
		}
	}
	return policy, nil
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
	additionalFields     map[string]json.RawMessage
	InstanceMin          int               `json:"instance_min_count"`
	InstanceMax          int               `json:"instance_max_count"`
	ScalingRules         []*ScalingRule    `json:"scaling_rules,omitempty"`
	Schedules            *ScalingSchedules `json:"schedules,omitempty"`
	MinMemoryMB          int               `json:"min_memory_mb,omitempty"`
	MaxMemoryMB          int               `json:"max_memory_mb,omitempty"`
	VerticalScalingRules []*ScalingRule    `json:"vertical_scaling_rules,omitempty"`
	ScalingPriority      ScalingDimension  `json:"scaling_priority,omitempty"`
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

func (pd PolicyDefinition) String() string {
	aJson, _ := pd.ToRawJSON()
	return string(aJson)
}

var _ fmt.Stringer = &PolicyDefinition{}

type ScalingRule struct {
	additionalFields      map[string]json.RawMessage
	MetricType            string `json:"metric_type"`
	BreachDurationSeconds int    `json:"breach_duration_secs,omitempty"`
	Threshold             int64  `json:"threshold"`
	Operator              string `json:"operator"`
	CoolDownSeconds       int    `json:"cool_down_secs,omitempty"`
	Adjustment            string `json:"adjustment"`
}

type ScalingSchedules struct {
	additionalFields      map[string]json.RawMessage
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
	additionalFields      map[string]json.RawMessage
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
	additionalFields      map[string]json.RawMessage
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

// ScalingDimension identifies the resource changed by a scaling action.
// An omitted dimension in legacy triggers and history means horizontal scaling.
type ScalingDimension string

const (
	ScalingDimensionHorizontal ScalingDimension = "horizontal"
	ScalingDimensionVertical   ScalingDimension = "vertical"
)

type Trigger struct {
	ScalingDimension      ScalingDimension `json:"scaling_dimension,omitempty"`
	AppId                 string           `json:"app_id"`
	MetricType            string           `json:"metric_type"`
	MetricUnit            string           `json:"metric_unit"`
	BreachDurationSeconds int              `json:"breach_duration_secs"`
	Threshold             int64            `json:"threshold"`
	Operator              string           `json:"operator"`
	CoolDownSeconds       int              `json:"cool_down_secs"`
	Adjustment            string           `json:"adjustment"`
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

// ValidateVerticalScaling validates relationships shared by public policies, binding
// requests and default policies. JSON schemas validate the individual field types.
func (pd *PolicyDefinition) ValidateVerticalScaling() error {
	if pd == nil {
		return nil
	}
	horizontal, vertical := len(pd.ScalingRules) > 0, len(pd.VerticalScalingRules) > 0
	if pd.ScalingPriority != "" {
		if pd.ScalingPriority != ScalingDimensionHorizontal && pd.ScalingPriority != ScalingDimensionVertical {
			return fmt.Errorf("scaling_priority must be horizontal or vertical")
		}
		if !horizontal || !vertical {
			return fmt.Errorf("scaling_priority requires nonempty scaling_rules and vertical_scaling_rules")
		}
	} else if horizontal && vertical {
		return fmt.Errorf("scaling_priority is required when both dynamic rule classes are nonempty")
	}
	if vertical || pd.MinMemoryMB != 0 || pd.MaxMemoryMB != 0 {
		if pd.MinMemoryMB <= 0 || pd.MaxMemoryMB <= 0 || pd.MinMemoryMB > pd.MaxMemoryMB {
			return fmt.Errorf("memory bounds must be positive and min_memory_mb must not exceed max_memory_mb")
		}
	}
	for i, rule := range pd.VerticalScalingRules {
		if rule == nil {
			return fmt.Errorf("vertical_scaling_rules[%d] must be an object", i)
		}
		if rule.MetricType != "entitlementutil" {
			return fmt.Errorf("vertical_scaling_rules[%d].metric_type must be entitlementutil", i)
		}
		if rule.Threshold < 0 {
			return fmt.Errorf("vertical_scaling_rules[%d].threshold must be nonnegative", i)
		}
		switch rule.Operator {
		case "<", "<=", ">", ">=":
		default:
			return fmt.Errorf("vertical_scaling_rules[%d].operator is invalid", i)
		}
		if rule.BreachDurationSeconds < 0 || rule.CoolDownSeconds < 0 {
			return fmt.Errorf("vertical_scaling_rules[%d] durations must be positive or omitted", i)
		}
		if !verticalAdjustmentPattern.MatchString(rule.Adjustment) {
			return fmt.Errorf("vertical_scaling_rules[%d].adjustment must be a signed positive integer, optionally followed by %%", i)
		}
		if strings.HasPrefix(rule.Adjustment, "-") && strings.HasSuffix(rule.Adjustment, "%") {
			magnitude := strings.TrimSuffix(rule.Adjustment[1:], "%")
			// Compare decimal strings to avoid overflow for arbitrarily large inputs.
			if len(magnitude) > 3 || len(magnitude) == 3 && magnitude > "100" {
				return fmt.Errorf("vertical_scaling_rules[%d].adjustment must not shrink by more than 100%%", i)
			}
		}
	}
	return nil
}

var verticalAdjustmentPattern = regexp.MustCompile(`^[-+][1-9][0-9]*%?$`)

func configurationHasExtensions(actual, known map[string]any) bool {
	for key, value := range actual {
		canonical, present := known[key]
		if !present {
			return true
		}
		if child, ok := value.(map[string]any); ok {
			if reference, ok := canonical.(map[string]any); !ok || configurationHasExtensions(child, reference) {
				return true
			}
		}
	}
	return false
}

func encodePolicyJSON(value any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func unmarshalPolicyObject(data []byte, value any, known ...string) (map[string]json.RawMessage, error) {
	if err := json.Unmarshal(data, value); err != nil {
		return nil, err
	}
	var extra map[string]json.RawMessage
	if err := json.Unmarshal(data, &extra); err != nil {
		return nil, err
	}
	for _, key := range known {
		delete(extra, key)
	}
	if len(extra) == 0 {
		return nil, nil
	}
	return extra, nil
}

func marshalPolicyObject(value any, extra map[string]json.RawMessage) ([]byte, error) {
	data, err := encodePolicyJSON(value)
	if err != nil || len(extra) == 0 {
		return data, err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, err
	}
	for key, value := range extra {
		if _, known := object[key]; !known {
			object[key] = value
		}
	}
	return encodePolicyJSON(object)
}

func (pd *PolicyDefinition) UnmarshalJSON(data []byte) error {
	type plain PolicyDefinition
	var value plain
	extra, err := unmarshalPolicyObject(data, &value, "instance_min_count", "instance_max_count", "scaling_rules", "schedules", "min_memory_mb", "max_memory_mb", "vertical_scaling_rules", "scaling_priority", "configuration", "schema-version", "credential-type", "role", "is_admin", "is_sso")
	if err == nil {
		*pd = PolicyDefinition(value)
		pd.additionalFields = extra
	}
	return err
}

func (pd PolicyDefinition) MarshalJSON() ([]byte, error) {
	type plain PolicyDefinition
	return marshalPolicyObject(plain(pd), pd.additionalFields)
}

func (r *ScalingRule) UnmarshalJSON(data []byte) error {
	type plain ScalingRule
	var value plain
	extra, err := unmarshalPolicyObject(data, &value, "metric_type", "breach_duration_secs", "threshold", "operator", "cool_down_secs", "adjustment", "stats_window_secs")
	if err == nil {
		*r = ScalingRule(value)
		r.additionalFields = extra
	}
	return err
}

func (r ScalingRule) MarshalJSON() ([]byte, error) {
	type plain ScalingRule
	return marshalPolicyObject(plain(r), r.additionalFields)
}

func (s *ScalingSchedules) UnmarshalJSON(data []byte) error {
	type plain ScalingSchedules
	var value plain
	extra, err := unmarshalPolicyObject(data, &value, "timezone", "recurring_schedule", "specific_date")
	if err == nil {
		*s = ScalingSchedules(value)
		s.additionalFields = extra
	}
	return err
}

func (s ScalingSchedules) MarshalJSON() ([]byte, error) {
	type plain ScalingSchedules
	return marshalPolicyObject(plain(s), s.additionalFields)
}

func (s *RecurringSchedule) UnmarshalJSON(data []byte) error {
	type plain RecurringSchedule
	var value plain
	extra, err := unmarshalPolicyObject(data, &value, "start_time", "end_time", "days_of_week", "days_of_month", "start_date", "end_date", "instance_min_count", "instance_max_count", "initial_min_instance_count")
	if err == nil {
		*s = RecurringSchedule(value)
		s.additionalFields = extra
	}
	return err
}

func (s RecurringSchedule) MarshalJSON() ([]byte, error) {
	type plain RecurringSchedule
	return marshalPolicyObject(plain(s), s.additionalFields)
}

func (s *SpecificDateSchedule) UnmarshalJSON(data []byte) error {
	type plain SpecificDateSchedule
	var value plain
	extra, err := unmarshalPolicyObject(data, &value, "start_date_time", "end_date_time", "instance_min_count", "instance_max_count", "initial_min_instance_count")
	if err == nil {
		*s = SpecificDateSchedule(value)
		s.additionalFields = extra
	}
	return err
}

func (s SpecificDateSchedule) MarshalJSON() ([]byte, error) {
	type plain SpecificDateSchedule
	return marshalPolicyObject(plain(s), s.additionalFields)
}

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
