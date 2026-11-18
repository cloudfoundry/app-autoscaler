package envelopeprocessor

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
)

// sourceTag does not guess identities from the numeric instance index. Both
// current and deprecated Loggregator tags are supported; conflicting tags are
// deliberately treated as unknown.
func sourceTag(e *loggregator_v2.Envelope, name string) string {
	value := e.GetTags()[name]
	if deprecated := e.GetDeprecatedTags()[name]; deprecated != nil {
		oldValue := deprecated.GetText()
		if value != "" && value != oldValue {
			return ""
		}
		value = oldValue
	}
	return value
}

func containerID(e *loggregator_v2.Envelope) string {
	var parts []string
	for _, name := range []string{"process_instance_id", "instance_id", "container_id"} {
		if value := sourceTag(e, name); value != "" {
			parts = append(parts, fmt.Sprintf("%s=%q", name, value))
		}
	}
	return strings.Join(parts, ",")
}

func validSourceTags(e *loggregator_v2.Envelope) bool {
	for _, name := range []string{"process_type", "process_id", "process_instance_id", "instance_id", "container_id"} {
		if value, exists := e.GetTags()[name]; exists {
			if old := e.GetDeprecatedTags()[name]; old != nil && old.GetText() != value {
				return false
			}
		}
	}
	return true
}

type entitlementSource struct {
	appID, instance, processType, processID, containerID string
	timestamp                                         int64
}

type entitlementInputs struct {
	values map[string]*loggregator_v2.GaugeValue
	invalid bool
}

// GetEntitlementInstanceMetrics joins the raw inputs BEFORE any averaging or
// lossy compaction. Only an exact app/process/container/index/timestamp match is
// permitted. Missing/duplicate inputs produce an empty value (never a zero).
func GetEntitlementInstanceMetrics(envelopes []*loggregator_v2.Envelope, collectedAt int64) []models.AppInstanceMetric {
	groups := make(map[entitlementSource]*entitlementInputs)
	for _, envelope := range envelopes {
		if envelope == nil || envelope.GetGauge() == nil {
			continue
		}
		gauges := envelope.GetGauge().GetMetrics()
		_, memory := gauges["memory"]
		_, quota := gauges["memory_quota"]
		_, cpu := gauges["cpu_entitlement"]
		if !memory && !quota && !cpu {
			continue
		}
		processType := sourceTag(envelope, "process_type")
		if processType != "" && processType != "web" {
			continue
		}
		source := entitlementSource{
			appID: envelope.SourceId, instance: envelope.InstanceId, timestamp: envelope.Timestamp,
			processType: processType, processID: sourceTag(envelope, "process_id"), containerID: containerID(envelope),
		}
		inputs := groups[source]
		if inputs == nil {
			inputs = &entitlementInputs{values: make(map[string]*loggregator_v2.GaugeValue)}
			groups[source] = inputs
		}
		inputs.invalid = inputs.invalid || !validSourceTags(envelope)
		for _, name := range []string{"memory", "memory_quota", "cpu_entitlement"} {
			if value, exists := gauges[name]; exists {
				if _, duplicate := inputs.values[name]; duplicate {
					inputs.invalid = true
				}
				inputs.values[name] = value
			}
		}
	}

	metrics := make([]models.AppInstanceMetric, 0, len(groups))
	for source, inputs := range groups {
		index, indexErr := strconv.ParseUint(source.instance, 10, 64)
		metric := models.AppInstanceMetric{
			AppId: source.appID, InstanceIndex: index, Timestamp: source.timestamp, CollectedAt: collectedAt,
			Name: models.MetricNameEntitlementUtil, Unit: models.UnitPercentage,
			ProcessType: source.processType, ProcessID: source.processID, ContainerID: source.containerID,
		}
		memory, quota, cpu := inputs.values["memory"], inputs.values["memory_quota"], inputs.values["cpu_entitlement"]
		if !inputs.invalid && indexErr == nil && source.appID != "" && source.timestamp > 0 &&
			source.processType == "web" && source.processID != "" && source.containerID != "" &&
			validGauge(memory, "bytes", false) && validGauge(quota, "bytes", true) && validGauge(cpu, "percentage", false) {
			// The existing percent metrics round up. max(ceil(a), ceil(b)) is
			// ceil(max(a,b)); CPU entitlement can legitimately exceed 100%.
			value := math.Ceil(math.Max(memory.Value/quota.Value*100, cpu.Value))
			if !math.IsInf(value, 0) && value < math.MaxInt64 {
				metric.Value = fmt.Sprintf("%.0f", value)
				metric.MemoryQuotaBytes = quota.Value
			}
		}
		metrics = append(metrics, metric)
	}
	return metrics
}

func validGauge(value *loggregator_v2.GaugeValue, unit string, positive bool) bool {
	return value != nil && value.Unit == unit && !math.IsNaN(value.Value) && !math.IsInf(value.Value, 0) &&
		value.Value >= 0 && (!positive || value.Value > 0)
}
