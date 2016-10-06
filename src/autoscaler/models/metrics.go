package models

import (
	"fmt"
	"time"

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

type Metric struct {
	Name      string `json:"name"`
	Unit      string `json:"unit"`
	AppId     string `json:"app_id"`
	Timestamp int64  `json:"timestamp"`
	Instances []InstanceMetric
}

type InstanceMetric struct {
	Timestamp int64  `json:"timestamp"`
	Index     uint32 `json:"index"`
	Value     string `json:"value"`
}

func GetMemoryMetricFromContainerMetrics(appId string, containerMetrics []*events.Envelope) *Metric {
	insts := []InstanceMetric{}
	for _, e := range containerMetrics {
		cm := e.ContainerMetric
		if *cm.ApplicationId == appId {
			insts = append(insts, InstanceMetric{
				Timestamp: e.GetTimestamp(),
				Index:     uint32(cm.GetInstanceIndex()),
				Value:     fmt.Sprintf("%d", cm.GetMemoryBytes()),
			})
		}
	}

	return &Metric{
		Name:      MetricNameMemory,
		Unit:      UnitBytes,
		AppId:     appId,
		Timestamp: time.Now().UnixNano(),
		Instances: insts,
	}
}
