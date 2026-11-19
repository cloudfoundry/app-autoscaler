package aggregator

import (
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

func entitlementMetric(index uint64, timestamp int64, value string) models.AppInstanceMetric {
	return models.AppInstanceMetric{
		AppId: "app", InstanceIndex: index, Name: models.MetricNameEntitlementUtil, Unit: models.UnitPercentage,
		Value: value, Timestamp: timestamp, ProcessType: "web", ProcessID: "process", ContainerID: valueContainer(index), MemoryQuotaBytes: 100,
	}
}

func valueContainer(index uint64) string {
	if index == 0 { return "container-0" }
	return "container-1"
}

func TestEntitlementAverageAfterPerInstanceMaximum(t *testing.T) {
	now := time.Now().Add(-time.Second).UnixNano()
	// Memory pressure on index 0 and CPU pressure on index 1: avg(max)
	// is 90, whereas max(avg(memory),avg(cpu)) would incorrectly be 50.
	metrics := []models.AppInstanceMetric{entitlementMetric(0, now, "90"), entitlementMetric(1, now, "90")}
	result := aggregateEntitlement("app", metrics)
	if result.Value != "90" || result.Timestamp != now || result.Unit != models.UnitPercentage {
		t.Fatalf("incorrect web instance aggregation: %+v", result)
	}
	metrics = append(metrics, entitlementMetric(0, now-int64(time.Second), "10"))
	result = aggregateEntitlement("app", metrics)
	if result.Value != "90" {
		t.Fatalf("unequal sample counts biased instance weighting: %+v", result)
	}
}

func TestEntitlementAggregationFailsClosed(t *testing.T) {
	now := time.Now().Add(-time.Second).UnixNano()
	for _, tc := range []struct {
		name string
		change func([]models.AppInstanceMetric) []models.AppInstanceMetric
	}{
		{"empty", func(m []models.AppInstanceMetric) []models.AppInstanceMetric { return nil }},
		{"missing input", func(m []models.AppInstanceMetric) []models.AppInstanceMetric { m[1].Value = ""; return m }},
		{"malformed value", func(m []models.AppInstanceMetric) []models.AppInstanceMetric { m[1].Value = "NaN"; return m }},
		{"missing identity", func(m []models.AppInstanceMetric) []models.AppInstanceMetric { m[1].ContainerID = ""; return m }},
		{"unknown process", func(m []models.AppInstanceMetric) []models.AppInstanceMetric { m[1].ProcessType = ""; return m }},
		{"different generation", func(m []models.AppInstanceMetric) []models.AppInstanceMetric { m[1].ProcessID = "new-process"; return m }},
		{"different memory quota", func(m []models.AppInstanceMetric) []models.AppInstanceMetric { m[1].MemoryQuotaBytes = 200; return m }},
		{"duplicate sample", func(m []models.AppInstanceMetric) []models.AppInstanceMetric { return append(m, m[0]) }},
		{"old and new container at one index", func(m []models.AppInstanceMetric) []models.AppInstanceMetric {
			other := m[0]; other.Timestamp++; other.ContainerID = "new-container"; return append(m, other)
		}},
		{"future sample", func(m []models.AppInstanceMetric) []models.AppInstanceMetric { m[1].Timestamp = time.Now().Add(time.Hour).UnixNano(); return m }},
		{"old invalid sample is not hidden by latest valid one", func(m []models.AppInstanceMetric) []models.AppInstanceMetric {
			other := m[0]; other.Timestamp--; other.Value = ""; return append(m, other)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metrics := tc.change([]models.AppInstanceMetric{entitlementMetric(0, now, "10"), entitlementMetric(1, now, "90")})
			if result := aggregateEntitlement("app", metrics); result.Value != "" {
				t.Fatalf("invalid web cohort became a numeric metric: %+v", result)
			}
		})
	}
}
