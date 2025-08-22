package envelopeprocessor

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
	"golang.org/x/exp/maps"
)

type EnvelopeProcessorCreator interface {
	NewProcessor(logger lager.Logger) Processor
}

type EnvelopeProcessor interface {
	GetGaugeMetrics(envelopes []*loggregator_v2.Envelope, currentTimeStamp int64) []models.AppInstanceMetric
}

var _ EnvelopeProcessor = &Processor{}

type Processor struct {
	logger lager.Logger
}

func NewProcessor(logger lager.Logger) Processor {
	return Processor{
		logger: logger.Session("EnvelopeProcessor"),
	}
}

func (p Processor) GetGaugeMetrics(envelopes []*loggregator_v2.Envelope, currentTimeStamp int64) []models.AppInstanceMetric {
	p.logger.Debug("GetGaugeMetrics")
	compactedEnvelopes := p.CompactEnvelopes(envelopes)
	p.logger.Debug("Compacted envelopes", lager.Data{"compactedEnvelopes": compactedEnvelopes})
	return GetGaugeInstanceMetrics(compactedEnvelopes, currentTimeStamp)
}

// Log cache returns instance metrics such as cpu and memory in serparate envelopes, this was not the case with
// loggregator. We compact this message by matching source_id and timestamp to facilitate metrics calulations.
func (p Processor) CompactEnvelopes(envelopes []*loggregator_v2.Envelope) []*loggregator_v2.Envelope {
	result := map[string]*loggregator_v2.Envelope{}
	for i := range envelopes {
		key := fmt.Sprintf("%s-%s-%d", envelopes[i].SourceId, envelopes[i].InstanceId, envelopes[i].Timestamp)
		if result[key] == nil {
			result[key] = &loggregator_v2.Envelope{}
			result[key] = envelopes[i]
		} else {
			metrics := envelopes[i].GetGauge().GetMetrics()
			for k, v := range metrics {
				result[key].GetGauge().Metrics[k] = v
			}
		}
	}

	return maps.Values(result)
}

func GetGaugeInstanceMetrics(envelopes []*loggregator_v2.Envelope, currentTimeStamp int64) []models.AppInstanceMetric {
	var metrics = []models.AppInstanceMetric{}

	for _, envelope := range envelopes {
		if isContainerMetricEnvelope(envelope) {
			containerMetrics := processContainerMetrics(envelope, currentTimeStamp)
			metrics = append(metrics, containerMetrics...)
		} else {
			customMetrics := processCustomMetrics(envelope, currentTimeStamp)
			metrics = append(metrics, customMetrics...)
		}
	}
	return metrics
}

func isContainerMetricEnvelope(e *loggregator_v2.Envelope) bool {
	metrics := maps.Keys(e.GetGauge().GetMetrics())

	// these metrics are required for the supported metric types
	relevantContainerMetrics := []string{"memory_quota", "memory", "cpu", "cpu_entitlement", "disk", "disk_quota"}

	for _, m := range metrics {
		if slices.Contains(relevantContainerMetrics, m) {
			return true
		}
	}
	return false
}

func processContainerMetrics(e *loggregator_v2.Envelope, currentTimeStamp int64) []models.AppInstanceMetric {
	var metrics []models.AppInstanceMetric
	appID := e.SourceId
	instanceIndex, _ := strconv.ParseUint(e.InstanceId, 10, 64)
	g := e.GetGauge()
	timestamp := e.Timestamp

	baseAppInstanceMetric := models.AppInstanceMetric{
		AppId:         appID,
		InstanceIndex: instanceIndex,
		CollectedAt:   currentTimeStamp,
		Timestamp:     timestamp,
	}

	if memory, exist := g.GetMetrics()["memory"]; exist {
		appInstanceMetric := getMemoryInstanceMetric(memory.GetValue())
		copyBaseAttributes(&appInstanceMetric, &baseAppInstanceMetric)
		metrics = append(metrics, appInstanceMetric)
	}

	if memoryQuota, exist := g.GetMetrics()["memory_quota"]; exist && memoryQuota.GetValue() != 0 {
		appInstanceMetric := getMemoryQuotaInstanceMetric(g.GetMetrics()["memory"].GetValue(), memoryQuota.GetValue())
		copyBaseAttributes(&appInstanceMetric, &baseAppInstanceMetric)
		metrics = append(metrics, appInstanceMetric)
	}

	if cpu, exist := g.GetMetrics()["cpu"]; exist {
		appInstanceMetric := getCPUInstanceMetric(cpu.GetValue())
		copyBaseAttributes(&appInstanceMetric, &baseAppInstanceMetric)
		metrics = append(metrics, appInstanceMetric)
	}

	if cpuEntitlement, exist := g.GetMetrics()["cpu_entitlement"]; exist {
		appInstanceMetric := getCPUEntitlementInstanceMetric(cpuEntitlement.GetValue())
		copyBaseAttributes(&appInstanceMetric, &baseAppInstanceMetric)
		metrics = append(metrics, appInstanceMetric)
	}

	if diskQuota, exist := g.GetMetrics()["disk_quota"]; exist && diskQuota.GetValue() != 0 {
		appInstanceMetric := getDiskQuotaInstanceMetric(g.GetMetrics()["disk"].GetValue(), diskQuota.GetValue())
		copyBaseAttributes(&appInstanceMetric, &baseAppInstanceMetric)
		metrics = append(metrics, appInstanceMetric)
	}

	if memory, exist := g.GetMetrics()["disk"]; exist {
		appInstanceMetric := getDiskInstanceMetric(memory.GetValue())
		copyBaseAttributes(&appInstanceMetric, &baseAppInstanceMetric)
		metrics = append(metrics, appInstanceMetric)
	}

	return metrics
}

func getMemoryInstanceMetric(memoryValue float64) models.AppInstanceMetric {
	return models.AppInstanceMetric{
		Name:  models.MetricNameMemoryUsed,
		Unit:  models.UnitMegaBytes,
		Value: fmt.Sprintf("%d", int(math.Ceil(memoryValue/(1024*1024)))),
	}
}

func getMemoryQuotaInstanceMetric(memoryValue float64, memoryQuotaValue float64) models.AppInstanceMetric {
	return models.AppInstanceMetric{
		Name:  models.MetricNameMemoryUtil,
		Unit:  models.UnitPercentage,
		Value: fmt.Sprintf("%d", int(math.Ceil(memoryValue/memoryQuotaValue*100))),
	}
}

func getDiskInstanceMetric(diskValue float64) models.AppInstanceMetric {
	return models.AppInstanceMetric{
		Name:  models.MetricNameDisk,
		Unit:  models.UnitMegaBytes,
		Value: fmt.Sprintf("%d", int(math.Ceil(diskValue/(1024*1024)))),
	}
}

func getDiskQuotaInstanceMetric(diskValue float64, diskQuotaValue float64) models.AppInstanceMetric {
	return models.AppInstanceMetric{
		Name:  models.MetricNameDiskUtil,
		Unit:  models.UnitPercentage,
		Value: fmt.Sprintf("%d", int(math.Ceil(diskValue/diskQuotaValue*100))),
	}
}

func getCPUInstanceMetric(cpuValue float64) models.AppInstanceMetric {
	return models.AppInstanceMetric{
		Name:  models.MetricNameCPU,
		Unit:  models.UnitPercentage,
		Value: fmt.Sprintf("%d", int64(math.Ceil(cpuValue))),
	}
}

func getCPUEntitlementInstanceMetric(cpuEntitlementValue float64) models.AppInstanceMetric {
	return models.AppInstanceMetric{
		Name:  models.MetricNameCPUUtil,
		Unit:  models.UnitPercentage,
		Value: fmt.Sprintf("%d", int64(math.Ceil(cpuEntitlementValue))),
	}
}

func processCustomMetrics(e *loggregator_v2.Envelope, currentTimestamp int64) []models.AppInstanceMetric {
	var metrics []models.AppInstanceMetric
	instanceIndex, _ := strconv.ParseUint(e.InstanceId, 10, 64)

	for n, v := range e.GetGauge().GetMetrics() {
		metrics = append(metrics, models.AppInstanceMetric{
			AppId:         e.SourceId,
			InstanceIndex: instanceIndex,
			CollectedAt:   currentTimestamp,
			Name:          n,
			Unit:          v.Unit,
			Value:         fmt.Sprintf("%d", int64(math.Ceil(v.Value))),
			Timestamp:     e.Timestamp,
		})
	}
	return metrics
}

func copyBaseAttributes(dst *models.AppInstanceMetric, src *models.AppInstanceMetric) {
	dst.AppId = src.AppId
	dst.InstanceIndex = src.InstanceIndex
	dst.CollectedAt = src.CollectedAt
	dst.Timestamp = src.Timestamp
}
