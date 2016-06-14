package metrics

import (
	"fmt"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
)

const (
	UNIT_PERCENTAGE  = "percentage"
	UNIT_BYTES       = "bytes"
	UNIT_NUM         = "num"
	UNIT_MILISECONDS = "milliseconds"
	UNIT_RPS         = "rps"
)

const MEMORY_METRIC_NAME = "memorybytes"

type Metric struct {
	Name      string `json:"name"`
	Unit      string `json:"unit"`
	AppId     string `json:"app_id"`
	TimeStamp int64  `json:"time_stamp"`
	Instances []InstanceMetric
}

type InstanceMetric struct {
	Index int32  `json:"index"`
	Value string `json:"value"`
}

func GetMemoryMetricFromContainerMetrics(appId string, containerMetrics []*events.ContainerMetric) *Metric {
	insts := []InstanceMetric{}
	for _, cm := range containerMetrics {
		if *cm.ApplicationId == appId {
			insts = append(insts, InstanceMetric{
				Index: cm.GetInstanceIndex(),
				Value: fmt.Sprintf("%d", cm.GetMemoryBytes()),
			})
		}
	}

	return &Metric{
		Name:      MEMORY_METRIC_NAME,
		Unit:      UNIT_BYTES,
		AppId:     appId,
		TimeStamp: time.Now().UnixNano(),
		Instances: insts,
	}
}
