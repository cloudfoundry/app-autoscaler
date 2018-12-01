package noaa

import (
	"autoscaler/models"
	"fmt"

	"github.com/cloudfoundry/sonde-go/events"
)

func NewContainerEnvelope(timestamp int64, appId string, index int32, cpu float64, memory uint64, disk uint64, memQuota uint64, diskQuota uint64) *events.Envelope {
	eventType := events.Envelope_ContainerMetric
	return &events.Envelope{
		EventType: &eventType,
		Timestamp: &timestamp,
		ContainerMetric: &events.ContainerMetric{
			ApplicationId:    &appId,
			InstanceIndex:    &index,
			CpuPercentage:    &cpu,
			MemoryBytes:      &memory,
			DiskBytes:        &disk,
			MemoryBytesQuota: &memQuota,
			DiskBytesQuota:   &diskQuota,
		},
	}
}

func NewHttpStartStopEnvelope(timestamp, startTime, stopTime int64, instanceIdx int32) *events.Envelope {
	eventType := events.Envelope_HttpStartStop
	return &events.Envelope{
		EventType: &eventType,
		Timestamp: &timestamp,
		HttpStartStop: &events.HttpStartStop{
			StartTimestamp: &startTime,
			StopTimestamp:  &stopTime,
			InstanceIndex:  &instanceIdx,
		},
	}
}

func GetMetricsFromContainerEnvelopes(collectAt int64, appId string, containerEnvelopes []*events.Envelope) []*models.AppInstanceMetric {
	metrics := []*models.AppInstanceMetric{}
	for _, event := range containerEnvelopes {
		metrics = append(metrics, GetMetricsFromContainerEnvelope(collectAt, appId, event)...)
	}
	return metrics
}

func GetMetricsFromContainerEnvelope(collectAt int64, appId string, event *events.Envelope) []*models.AppInstanceMetric {
	metrics := []*models.AppInstanceMetric{}
	cm := event.GetContainerMetric()
	if (cm != nil) && (*cm.ApplicationId == appId) {
		metrics = append(metrics, &models.AppInstanceMetric{
			AppId:         appId,
			InstanceIndex: uint32(cm.GetInstanceIndex()),
			CollectedAt:   collectAt,
			Name:          models.MetricNameMemoryUsed,
			Unit:          models.UnitMegaBytes,
			Value:         fmt.Sprintf("%d", int(float64(cm.GetMemoryBytes())/(1024*1024)+0.5)),
			Timestamp:     event.GetTimestamp(),
		})

		if cm.GetMemoryBytesQuota() != 0 {
			metrics = append(metrics, &models.AppInstanceMetric{
				AppId:         appId,
				InstanceIndex: uint32(cm.GetInstanceIndex()),
				CollectedAt:   collectAt,
				Name:          models.MetricNameMemoryUtil,
				Unit:          models.UnitPercentage,
				Value:         fmt.Sprintf("%d", int(float64(cm.GetMemoryBytes())/float64(cm.GetMemoryBytesQuota())*100+0.5)),
				Timestamp:     event.GetTimestamp(),
			})
		}
		metrics = append(metrics, &models.AppInstanceMetric{
			AppId:         appId,
			InstanceIndex: uint32(cm.GetInstanceIndex()),
			CollectedAt:   collectAt,
			Name:          models.MetricNameCPUUtil,
			Unit:          models.UnitPercentage,
			Value:         fmt.Sprintf("%d", int(float64(cm.GetCpuPercentage()+0.5))),
			Timestamp:     event.GetTimestamp(),
		})
	}
	return metrics
}
