package noaa

import (
	"autoscaler/models"
	"fmt"
	"github.com/cloudfoundry/sonde-go/events"
)

func NewContainerEnvelope(timestamp int64, appId string, index int32, cpu float64, memory uint64, disk uint64) *events.Envelope {
	eventType := events.Envelope_ContainerMetric
	return &events.Envelope{
		EventType: &eventType,
		Timestamp: &timestamp,
		ContainerMetric: &events.ContainerMetric{
			ApplicationId: &appId,
			InstanceIndex: &index,
			CpuPercentage: &cpu,
			MemoryBytes:   &memory,
			DiskBytes:     &disk,
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

func GetInstanceMemoryMetricFromContainerEnvelopes(collectAt int64, appId string, containerEnvelopes []*events.Envelope) []*models.AppInstanceMetric {
	metrics := []*models.AppInstanceMetric{}
	for _, event := range containerEnvelopes {
		instanceMetric := GetInstanceMemoryMetricFromContainerMetricEvent(collectAt, appId, event)
		if instanceMetric != nil {
			metrics = append(metrics, instanceMetric)
		}
	}
	return metrics
}

func GetInstanceMemoryMetricFromContainerMetricEvent(collectAt int64, appId string, event *events.Envelope) *models.AppInstanceMetric {
	cm := event.GetContainerMetric()
	if (cm != nil) && (*cm.ApplicationId == appId) {
		return &models.AppInstanceMetric{
			AppId:         appId,
			InstanceIndex: uint32(cm.GetInstanceIndex()),
			CollectedAt:   collectAt,
			Name:          models.MetricNameMemory,
			Unit:          models.UnitMegaBytes,
			Value:         fmt.Sprintf("%d", int(float64(cm.GetMemoryBytes())/(1024*1024)+0.5)),
			Timestamp:     event.GetTimestamp(),
		}
	}
	return nil
}
