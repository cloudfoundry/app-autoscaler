package envelopeprocessor

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"fmt"
	"github.com/imdario/mergo"
	"math"
	"strconv"
)

// Received envelopes belong to a single app id
func ComputeHttpStartStop(envelops []*loggregator_v2.Envelope, appID string, currentTimeStamp int64) []*models.AppInstanceMetric {

	//numRequests := map[uint32]int64{}
	//sumReponseTimes := map[uint32]int64{}

	//for _, envelope := range envelops {
	//	instanceIndex, _ := strconv.ParseInt(envelope.InstanceId, 10, 32)

	//	if numRequests == nil {
	//		numRequests = map[uint32]int64{}
	//	}
	//	if sumReponseTimes == nil {
	//		sumReponseTimes = map[uint32]int64{}
	//	}

	//	numRequests[uint32(instanceIndex)]++
	//	sumReponseTimes[uint32(instanceIndex)] += envelope.GetTimer().Stop - envelope.GetTimer().Start
	//}

	if len(envelops) == 0 {
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
			CollectedAt:   currentTimeStamp,
			Name:          models.MetricNameResponseTime,
			Unit:          models.UnitMilliseconds,
			Value:         "0",
			Timestamp:     currentTimeStamp,
		}

		var metrics []*models.AppInstanceMetric

		metrics = append(metrics, throughputMetric)
		metrics = append(metrics, responseTimeMetric)
		return metrics
	}
	return nil
}

func GetGaugeInstanceMetrics(e *loggregator_v2.Envelope, currentTimeStamp int64) []*models.AppInstanceMetric {
	if isContainerMetricEnvelope(e) {
		return processContainerMetrics(e, currentTimeStamp)
	} else {
		return processCustomMetrics(e, currentTimeStamp)
	}
}

func isContainerMetricEnvelope(e *loggregator_v2.Envelope) bool {
	_, exist := e.GetGauge().GetMetrics()["memory_quota"]
	return exist
}

func processContainerMetrics(e *loggregator_v2.Envelope, currentTimeStamp int64) []*models.AppInstanceMetric {
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

	var metrics []*models.AppInstanceMetric

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
