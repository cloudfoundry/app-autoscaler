package envelopeprocessor

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"golang.org/x/exp/maps"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"github.com/imdario/mergo"
)

// Received envelopes belong to a single app id
func ComputeHttpStartStop(envelopes []*loggregator_v2.Envelope, appID string, currentTimeStamp int64,
	collectionInterval time.Duration) []*models.AppInstanceMetric {
	var metrics []*models.AppInstanceMetric

	if len(envelopes) == 0 {
		throughputMetric := &models.AppInstanceMetric{
			AppId:         appID,
			InstanceIndex: 0,
			CollectedAt:   currentTimeStamp,
			Name:          models.MetricNameThroughput,
			Unit:          models.UnitRPS,
			Value:         "0",
			Timestamp:     currentTimeStamp,
		}

		responseTimeMetric := &models.AppInstanceMetric{
			AppId:         appID,
			InstanceIndex: 0,
			Name:          models.MetricNameResponseTime,
			Unit:          models.UnitMilliseconds,
			Value:         "0",
			CollectedAt:   currentTimeStamp,
			Timestamp:     currentTimeStamp,
		}

		metrics = append(metrics, throughputMetric)
		metrics = append(metrics, responseTimeMetric)
	} else {
		numRequestsPerAppIdx, sumReponseTimesPerAppIdx := calcNumReqsAndResponseTimes(envelopes)

		for _, instanceIndex := range maps.Keys(sumReponseTimesPerAppIdx) {
			numReq := numRequestsPerAppIdx[instanceIndex]
			sumReponseTime := sumReponseTimesPerAppIdx[instanceIndex]

			throughputMetric := &models.AppInstanceMetric{
				AppId:         appID,
				InstanceIndex: uint32(instanceIndex),
				Name:          models.MetricNameThroughput,
				Unit:          models.UnitRPS,
				Value:         fmt.Sprintf("%d", int(math.Ceil(float64(numReq)/collectionInterval.Seconds()))),
				CollectedAt:   currentTimeStamp,
				Timestamp:     currentTimeStamp,
			}
			metrics = append(metrics, throughputMetric)

			responseTimeMetric := &models.AppInstanceMetric{
				AppId:         appID,
				InstanceIndex: uint32(instanceIndex),
				Name:          models.MetricNameResponseTime,
				Unit:          models.UnitMilliseconds,
				Value:         fmt.Sprintf("%d", int64(math.Ceil(float64(sumReponseTime)/float64(numReq*1000*1000)))),
				CollectedAt:   currentTimeStamp,
				Timestamp:     currentTimeStamp,
			}
			metrics = append(metrics, responseTimeMetric)
		}

	}

	return metrics
}

func GetGaugeInstanceMetrics(e *loggregator_v2.Envelope, currentTimeStamp int64) []*models.AppInstanceMetric {
	if isContainerMetricEnvelope(e) {
		return processContainerMetrics(e, currentTimeStamp)
	} else {
		return processCustomMetrics(e, currentTimeStamp)
	}
}

// Aggregates statistics: Be careful: All envelopes must belong to the same app.
func calcNumReqsAndResponseTimes(envelopes []*loggregator_v2.Envelope) (numRequestsPerAppIdx map[uint64]int64, sumReponseTimesPerAppIdx map[uint64]int64) {
	numRequestsPerAppIdx = map[uint64]int64{}
	sumReponseTimesPerAppIdx = map[uint64]int64{}
	for _, envelope := range envelopes {
		instanceIdx, _ := strconv.ParseUint(envelope.InstanceId, 10, 32)

		if _, exists := numRequestsPerAppIdx[instanceIdx]; exists {
			numRequestsPerAppIdx[instanceIdx] += 1
		} else {
			numRequestsPerAppIdx[instanceIdx] = 1
		}

		if _, exists := sumReponseTimesPerAppIdx[instanceIdx]; exists {
			sumReponseTimesPerAppIdx[instanceIdx] += envelope.GetTimer().Stop - envelope.GetTimer().Start
		} else {
			sumReponseTimesPerAppIdx[instanceIdx] = envelope.GetTimer().Stop - envelope.GetTimer().Start
		}
	}

	return
}

func isContainerMetricEnvelope(e *loggregator_v2.Envelope) bool {
	_, exist := e.GetGauge().GetMetrics()["memory_quota"]
	return exist
}

func processContainerMetrics(e *loggregator_v2.Envelope, currentTimeStamp int64) []*models.AppInstanceMetric {
	var metrics []*models.AppInstanceMetric
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
		mergo.Merge(appInstanceMetric, baseAppInstanceMetric)
		metrics = append(metrics, appInstanceMetric)
	}

	if memoryQuota, exist := g.GetMetrics()["memory_quota"]; exist && memoryQuota.GetValue() != 0 {
		appInstanceMetric := getMemoryQuotaInstanceMetric(g.GetMetrics()["memory"].GetValue(), memoryQuota.GetValue())
		mergo.Merge(appInstanceMetric, baseAppInstanceMetric)
		metrics = append(metrics, appInstanceMetric)
	}

	if cpu, exist := g.GetMetrics()["cpu"]; exist {
		appInstanceMetric := getCPUInstanceMetric(cpu.GetValue())
		mergo.Merge(appInstanceMetric, baseAppInstanceMetric)
		metrics = append(metrics, appInstanceMetric)
	}

	return metrics
}

func getMemoryInstanceMetric(memoryValue float64) *models.AppInstanceMetric {
	return &models.AppInstanceMetric{
		Name:  models.MetricNameMemoryUsed,
		Unit:  models.UnitMegaBytes,
		Value: fmt.Sprintf("%d", int(math.Ceil(memoryValue/(1024*1024)))),
	}
}

func getMemoryQuotaInstanceMetric(memoryValue float64, memoryQuotaValue float64) *models.AppInstanceMetric {
	return &models.AppInstanceMetric{
		Name:  models.MetricNameMemoryUtil,
		Unit:  models.UnitPercentage,
		Value: fmt.Sprintf("%d", int(math.Ceil(memoryValue/memoryQuotaValue*100))),
	}
}

func getCPUInstanceMetric(cpuValue float64) *models.AppInstanceMetric {
	return &models.AppInstanceMetric{
		Name:  models.MetricNameCPUUtil,
		Unit:  models.UnitPercentage,
		Value: fmt.Sprintf("%d", int64(math.Ceil(cpuValue))),
	}
}

func processCustomMetrics(e *loggregator_v2.Envelope, currentTimestamp int64) []*models.AppInstanceMetric {
	var metrics []*models.AppInstanceMetric
	instanceIndex, _ := strconv.ParseInt(e.InstanceId, 10, 32)

	for n, v := range e.GetGauge().GetMetrics() {
		metrics = append(metrics, &models.AppInstanceMetric{
			AppId:         e.SourceId,
			InstanceIndex: uint32(instanceIndex),
			CollectedAt:   currentTimestamp,
			Name:          n,
			Unit:          v.Unit,
			Value:         fmt.Sprintf("%d", int64(math.Ceil(v.Value))),
			Timestamp:     e.Timestamp,
		})
	}
	return metrics
}
