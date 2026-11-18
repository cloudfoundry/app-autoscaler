package generator

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

func canEvaluateOtherDimension(result *scalingEngineResult) bool {
	if result == nil || result.Status != models.ScalingStatusIgnored || result.CooldownExpiredAt != 0 {
		return false
	}
	switch result.Reason {
	case "priority_requires_max_memory", "priority_requires_min_memory", "priority_requires_min_instances", "priority_requires_max_instances", "unchanged_target":
		return true
	default:
		// active_deployment, cooldown, disabled app, missing policy, and any
		// unknown reason apply to the whole app, not just this dimension.
		return false
	}
}

// Keep the boundary observation as well as all points inside the breach window.
// Unlike legacy metrics, entitlementutil cannot infer a continuous breach from
// just one arbitrarily old value and one recent value. Missing, duplicate,
// future, stale or widely separated measurements fail closed. Empty values are
// retained and rejected by the normal breach comparison.
func entitlementBreachMetrics(metrics []*models.AppMetric, start, end time.Time, maxGap time.Duration) []*models.AppMetric {
	if len(metrics) == 0 || maxGap <= 0 {
		return nil
	}
	boundary := -1
	for i, metric := range metrics {
		if metric == nil || metric.Timestamp > end.UnixNano() || (i > 0 && metric.Timestamp <= metrics[i-1].Timestamp) {
			return nil
		}
		if metric.Timestamp <= start.UnixNano() {
			boundary = i
		}
	}
	if boundary == -1 || start.UnixNano()-metrics[boundary].Timestamp > int64(maxGap) ||
		end.UnixNano()-metrics[len(metrics)-1].Timestamp > int64(maxGap) {
		return nil
	}
	result := metrics[boundary:]
	if len(result) < 2 {
		return nil
	}
	for i := 1; i < len(result); i++ {
		if result[i].Timestamp-result[i-1].Timestamp > int64(maxGap) {
			return nil
		}
	}
	return result
}
