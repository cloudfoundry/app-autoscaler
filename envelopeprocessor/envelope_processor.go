package envelopeprocessor

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"code.cloudfoundry.org/lager/v3"

	"golang.org/x/exp/maps"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"dario.cat/mergo"
)

type EnvelopeProcessorCreator interface {
	NewProcessor(logger lager.Logger, collectionInterval time.Duration) Processor
}

type EnvelopeProcessor interface {
	GetGaugeMetrics(envelopes []*loggregator_v2.Envelope, currentTimeStamp int64) ([]models.AppInstanceMetric, error)
	GetTimerMetrics(envelopes []*loggregator_v2.Envelope, appID string, currentTimestamp int64) []models.AppInstanceMetric
}

var _ EnvelopeProcessor = &Processor{}

type Processor struct {
	logger             lager.Logger
	collectionInterval time.Duration
}

func NewProcessor(logger lager.Logger, collectionInterval time.Duration) Processor {
	return Processor{
		logger:             logger.Session("EnvelopeProcessor"),
		collectionInterval: collectionInterval,
	}
}

func (p Processor) GetGaugeMetrics(envelopes []*loggregator_v2.Envelope, currentTimeStamp int64) ([]models.AppInstanceMetric, error) {
	p.logger.Debug("GetGaugeMetrics")
	compactedEnvelopes := p.CompactEnvelopes(envelopes)
	p.logger.Debug("Compacted envelopes", lager.Data{"compactedEnvelopes": compactedEnvelopes})
	return GetGaugeInstanceMetrics(compactedEnvelopes, currentTimeStamp)
}
func (p Processor) GetTimerMetrics(envelopes []*loggregator_v2.Envelope, appID string, currentTimestamp int64) []models.AppInstanceMetric {
	p.logger.Debug("GetTimerMetrics")
	p.logger.Debug("Compacted envelopes", lager.Data{"Envelopes": envelopes})
	return GetHttpStartStopInstanceMetrics(envelopes, appID, currentTimestamp, p.collectionInterval)
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

func GetGaugeInstanceMetrics(envelopes []*loggregator_v2.Envelope, currentTimeStamp int64) ([]models.AppInstanceMetric, error) {
	var metrics = []models.AppInstanceMetric{}

	for _, envelope := range envelopes {
		if isContainerMetricEnvelope(envelope) {
			//TODO: please see https://github.com/cloudfoundry/app-autoscaler-release/issues/717
			containerMetrics, _ := processContainerMetrics(envelope, currentTimeStamp)
			metrics = append(metrics, containerMetrics...)
		} else {
			//TODO: please see https://github.com/cloudfoundry/app-autoscaler-release/issues/717
			containerMetrics, _ := processCustomMetrics(envelope, currentTimeStamp)
			metrics = append(metrics, containerMetrics...)
		}
	}
	return metrics, nil
}

func GetHttpStartStopInstanceMetrics(envelopes []*loggregator_v2.Envelope, appID string, currentTimestamp int64,
	collectionInterval time.Duration) []models.AppInstanceMetric {
	var metrics []models.AppInstanceMetric

	numRequestsPerAppIdx := calcNumReqs(envelopes)
	sumReponseTimesPerAppIdx := calcSumResponseTimes(envelopes)

	throughputMetrics := getThroughputInstanceMetrics(envelopes, appID, numRequestsPerAppIdx, collectionInterval, currentTimestamp)
	responseTimeMetric := getResponsetimeInstanceMetrics(envelopes, appID, numRequestsPerAppIdx, sumReponseTimesPerAppIdx, currentTimestamp)

	metrics = append(metrics, throughputMetrics...)
	metrics = append(metrics, responseTimeMetric...)

	return metrics
}

func getResponsetimeInstanceMetrics(envelopes []*loggregator_v2.Envelope, appID string, numRequestsPerAppIdx map[uint64]int64, sumReponseTimesPerAppIdx map[uint64]int64, currentTimestamp int64) []models.AppInstanceMetric {
	var metrics []models.AppInstanceMetric

	if len(envelopes) == 0 {
		responseTimeMetric := models.AppInstanceMetric{
			AppId:         appID,
			InstanceIndex: 0,
			Name:          models.MetricNameResponseTime,
			Unit:          models.UnitMilliseconds,
			Value:         "0",
			CollectedAt:   currentTimestamp,
			Timestamp:     currentTimestamp,
		}
		metrics = append(metrics, responseTimeMetric)
	} else {
		for _, instanceIndex := range maps.Keys(sumReponseTimesPerAppIdx) {
			numReq := numRequestsPerAppIdx[instanceIndex]
			sumResponseTime := sumReponseTimesPerAppIdx[instanceIndex]

			responseTimeMetric := models.AppInstanceMetric{
				AppId:         appID,
				InstanceIndex: uint32(instanceIndex),
				Name:          models.MetricNameResponseTime,
				Unit:          models.UnitMilliseconds,
				Value:         fmt.Sprintf("%d", int64(math.Ceil(float64(sumResponseTime)/float64(numReq*1000*1000)))),
				CollectedAt:   currentTimestamp,
				Timestamp:     currentTimestamp,
			}
			metrics = append(metrics, responseTimeMetric)
		}
	}
	return metrics
}

func getThroughputInstanceMetrics(envelopes []*loggregator_v2.Envelope, appID string, numRequestsPerAppIdx map[uint64]int64, collectionInterval time.Duration, currentTimestamp int64) []models.AppInstanceMetric {
	var metrics []models.AppInstanceMetric

	if len(envelopes) == 0 {
		throughputMetric := models.AppInstanceMetric{
			AppId:         appID,
			InstanceIndex: 0,
			Name:          models.MetricNameThroughput,
			Unit:          models.UnitRPS,
			Value:         "0",
			CollectedAt:   currentTimestamp,
			Timestamp:     currentTimestamp,
		}
		metrics = append(metrics, throughputMetric)
	} else {
		for _, instanceIndex := range maps.Keys(numRequestsPerAppIdx) {
			numReq := numRequestsPerAppIdx[instanceIndex]

			throughputMetric := models.AppInstanceMetric{
				AppId:         appID,
				InstanceIndex: uint32(instanceIndex),
				Name:          models.MetricNameThroughput,
				Unit:          models.UnitRPS,
				Value:         fmt.Sprintf("%d", int(math.Ceil(float64(numReq)/collectionInterval.Seconds()))),
				CollectedAt:   currentTimestamp,
				Timestamp:     currentTimestamp,
			}
			metrics = append(metrics, throughputMetric)
		}
	}

	return metrics
}

func calcSumResponseTimes(envelopes []*loggregator_v2.Envelope) (sumReponseTimesPerAppIdx map[uint64]int64) {
	sumReponseTimesPerAppIdx = map[uint64]int64{}
	for _, envelope := range envelopes {
		instanceIdx, _ := strconv.ParseUint(envelope.InstanceId, 10, 32)

		sumReponseTimesPerAppIdx[instanceIdx] += envelope.GetTimer().Stop - envelope.GetTimer().Start
	}
	return
}

func calcNumReqs(envelopes []*loggregator_v2.Envelope) (numRequestsPerAppIdx map[uint64]int64) {
	numRequestsPerAppIdx = map[uint64]int64{}
	for _, envelope := range envelopes {
		instanceIdx, _ := strconv.ParseUint(envelope.InstanceId, 10, 32)

		numRequestsPerAppIdx[instanceIdx] += 1
	}
	return
}

func isContainerMetricEnvelope(e *loggregator_v2.Envelope) bool {
	keys := maps.Keys(e.GetGauge().GetMetrics())

	var matchingKeys []string

	for i := range keys {
		if keys[i] == "memory_quota" || keys[i] == "memory" || keys[i] == "cpu" || keys[i] == "cpu_entitlement" {
			matchingKeys = append(matchingKeys, keys[i])
		}
	}
	return len(matchingKeys) != 0
}

func processContainerMetrics(e *loggregator_v2.Envelope, currentTimeStamp int64) ([]models.AppInstanceMetric, error) {
	var metrics []models.AppInstanceMetric
	appID := e.SourceId
	instanceIndex, _ := strconv.ParseInt(e.InstanceId, 10, 32)
	g := e.GetGauge()
	timestamp := e.Timestamp

	baseAppInstanceMetric := models.AppInstanceMetric{
		AppId:         appID,
		InstanceIndex: uint32(instanceIndex),
		CollectedAt:   currentTimeStamp,
		Timestamp:     timestamp,
	}

	if memory, exist := g.GetMetrics()["memory"]; exist {
		appInstanceMetric := getMemoryInstanceMetric(memory.GetValue())
		err := mergo.Merge(&appInstanceMetric, baseAppInstanceMetric)
		if err != nil {
			return []models.AppInstanceMetric{}, err
		}

		metrics = append(metrics, appInstanceMetric)
	}

	if memoryQuota, exist := g.GetMetrics()["memory_quota"]; exist && memoryQuota.GetValue() != 0 {
		appInstanceMetric := getMemoryQuotaInstanceMetric(g.GetMetrics()["memory"].GetValue(), memoryQuota.GetValue())
		err := mergo.Merge(&appInstanceMetric, baseAppInstanceMetric)
		if err != nil {
			return []models.AppInstanceMetric{}, err
		}
		metrics = append(metrics, appInstanceMetric)
	}

	if cpu, exist := g.GetMetrics()["cpu"]; exist {
		appInstanceMetric := getCPUInstanceMetric(cpu.GetValue())
		err := mergo.Merge(&appInstanceMetric, baseAppInstanceMetric)
		if err != nil {
			return []models.AppInstanceMetric{}, err
		}
		metrics = append(metrics, appInstanceMetric)
	}

	if cpuEntitlement, exist := g.GetMetrics()["cpu_entitlement"]; exist {
		appInstanceMetric := getCPUEntitlementInstanceMetric(cpuEntitlement.GetValue())
		err := mergo.Merge(&appInstanceMetric, baseAppInstanceMetric)
		if err != nil {
			return []models.AppInstanceMetric{}, err
		}
		metrics = append(metrics, appInstanceMetric)
	}

	return metrics, nil
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

func processCustomMetrics(e *loggregator_v2.Envelope, currentTimestamp int64) ([]models.AppInstanceMetric, error) {
	var metrics []models.AppInstanceMetric
	instanceIndex, _ := strconv.ParseInt(e.InstanceId, 10, 32)

	for n, v := range e.GetGauge().GetMetrics() {
		metrics = append(metrics, models.AppInstanceMetric{
			AppId:         e.SourceId,
			InstanceIndex: uint32(instanceIndex),
			CollectedAt:   currentTimestamp,
			Name:          n,
			Unit:          v.Unit,
			Value:         fmt.Sprintf("%d", int64(math.Ceil(v.Value))),
			Timestamp:     e.Timestamp,
		})
	}
	return metrics, nil
}
