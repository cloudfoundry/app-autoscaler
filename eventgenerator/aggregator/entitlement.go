package aggregator

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

// Each polling window contributes one equally weighted, latest complete sample
// per observed web instance. An incomplete sample anywhere in the window, two
// containers for one index, different process generations or memory limits
// invalidate the entire point rather than biasing it towards the surviving data.
func aggregateEntitlement(appID string, metrics []models.AppInstanceMetric) *models.AppMetric {
	result := &models.AppMetric{AppId: appID, MetricType: models.MetricNameEntitlementUtil, Timestamp: time.Now().UnixNano()}
	latest := make(map[uint64]models.AppInstanceMetric)
	seen := make(map[string]bool)
	var processID string
	var quota float64
	for _, metric := range metrics {
		value, err := strconv.ParseInt(metric.Value, 10, 64)
		if metric.AppId != appID || metric.Name != models.MetricNameEntitlementUtil || metric.ProcessType != "web" ||
			metric.ProcessID == "" || metric.ContainerID == "" || metric.Timestamp <= 0 || metric.Timestamp > result.Timestamp ||
			metric.Unit != models.UnitPercentage || err != nil || value < 0 || metric.MemoryQuotaBytes <= 0 ||
			math.IsNaN(metric.MemoryQuotaBytes) || math.IsInf(metric.MemoryQuotaBytes, 0) {
			return result
		}
		if processID == "" {
			processID, quota = metric.ProcessID, metric.MemoryQuotaBytes
		}
		if processID != metric.ProcessID || quota != metric.MemoryQuotaBytes {
			return result
		}
		key := fmt.Sprintf("%d/%d", metric.InstanceIndex, metric.Timestamp)
		if seen[key] {
			return result
		}
		seen[key] = true
		previous, exists := latest[metric.InstanceIndex]
		if exists && previous.ContainerID != metric.ContainerID {
			return result
		}
		if !exists || metric.Timestamp > previous.Timestamp {
			latest[metric.InstanceIndex] = metric
		}
	}
	if len(latest) == 0 {
		return result
	}
	var sum float64
	for _, metric := range latest {
		value, _ := strconv.ParseInt(metric.Value, 10, 64)
		sum += float64(value)
		if metric.Timestamp < result.Timestamp {
			result.Timestamp = metric.Timestamp
		}
	}
	result.Value = fmt.Sprintf("%.0f", math.Ceil(sum/float64(len(latest))))
	result.Unit = models.UnitPercentage
	indices := make([]uint64, 0, len(latest))
	for index := range latest {
		indices = append(indices, index)
	}
	sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })
	var source strings.Builder
	fmt.Fprintf(&source, "%q/%g", processID, quota)
	for _, index := range indices {
		fmt.Fprintf(&source, "/%d:%q", index, latest[index].ContainerID)
	}
	result.SourceID = source.String()
	return result
}
