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
	TimeStamp int64  `json:"timestamp"`
	Instances []InstanceMetric
}

type InstanceMetric struct {
	Timestamp int64  `json:"timestamp"`
	Index     uint32 `json:"index"`
	Value     string `json:"value"`
}

func GetMemoryMetricFromContainerMetrics(appId string, containerMetrics []*events.ContainerMetric) *Metric {
	insts := []InstanceMetric{}
	for _, cm := range containerMetrics {
		if *cm.ApplicationId == appId {
			insts = append(insts, InstanceMetric{
				Index: uint32(cm.GetInstanceIndex()),
				Value: fmt.Sprintf("%d", cm.GetMemoryBytes()),
			})
		}
	}

	return &Metric{
		Name:      MetricNameMemory,
		Unit:      UnitBytes,
		AppId:     appId,
		TimeStamp: time.Now().UnixNano(),
		Instances: insts,
	}
}
