package models

import (
	"fmt"

	"github.com/cloudfoundry/sonde-go/events"
)

const (
	UnitPercentage   = "percentage"
	UnitMegaBytes    = "megabytes"
	UnitNum          = "num"
	UnitMilliseconds = "milliseconds"
	UnitRPS          = "rps"
)

const MetricNameMemory = "memoryused"

type AppInstanceMetric struct {
	AppId         string `json:"app_id"`
	InstanceIndex uint32 `json:"instance_index"`
	CollectedAt   int64  `json:"collected_at"`
	Name          string `json:"name"`
	Unit          string `json:"unit"`
	Value         string `json:"value"`
	Timestamp     int64  `json:"timestamp"`
}

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

func GetInstanceMemoryMetricFromContainerEnvelopes(collectAt int64, appId string, containerEnvelopes []*events.Envelope) []*AppInstanceMetric {
	metrics := []*AppInstanceMetric{}
	for _, event := range containerEnvelopes {
		instanceMetric := GetInstanceMemoryMetricFromContainerMetricEvent(collectAt, appId, event)
		if instanceMetric != nil {
			metrics = append(metrics, instanceMetric)
		}
	}
	return metrics
}

func GetInstanceMemoryMetricFromContainerMetricEvent(collectAt int64, appId string, event *events.Envelope) *AppInstanceMetric {
	cm := event.GetContainerMetric()
	if (cm != nil) && (*cm.ApplicationId == appId) {
		return &AppInstanceMetric{
			AppId:         appId,
			InstanceIndex: uint32(cm.GetInstanceIndex()),
			CollectedAt:   collectAt,
			Name:          MetricNameMemory,
			Unit:          UnitMegaBytes,
			Value:         fmt.Sprintf("%d", int(float64(cm.GetMemoryBytes())/(1024*1024)+0.5)),
			Timestamp:     event.GetTimestamp(),
		}
	}
	return nil
}
