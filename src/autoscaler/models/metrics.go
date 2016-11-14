package models

import (
	"fmt"

	"github.com/cloudfoundry/sonde-go/events"
)

const (
	UnitPercentage   = "percentage"
	UnitBytes        = "bytes"
	UnitNum          = "num"
	UnitMilliseconds = "milliseconds"
	UnitRPS          = "rps"
)

const MetricNameMemory = "memorybytes"

type AppInstanceMetric struct {
	AppId         string `json:"app_id"`
	InstanceIndex uint32 `json:"instance_index"`
	CollectedAt   int64  `json:"collected_at"`
	Name          string `json:"name"`
	Unit          string `json:"unit"`
	Value         string `json:"value"`
	Timestamp     int64  `json:"timestamp"`
}

func GetInstanceMemoryMetricFromContainerEnvelopes(collectAt int64, appId string, containerEnvelopes []*events.Envelope) []*AppInstanceMetric {
	metrics := []*AppInstanceMetric{}
	for _, e := range containerEnvelopes {
		cm := e.ContainerMetric
		if *cm.ApplicationId == appId {
			metrics = append(metrics, &AppInstanceMetric{
				AppId:         appId,
				InstanceIndex: uint32(cm.GetInstanceIndex()),
				CollectedAt:   collectAt,
				Name:          MetricNameMemory,
				Unit:          UnitBytes,
				Value:         fmt.Sprintf("%d", cm.GetMemoryBytes()),
				Timestamp:     e.GetTimestamp(),
			})
		}
	}
	return metrics
}
