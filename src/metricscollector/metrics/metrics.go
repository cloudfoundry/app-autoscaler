package metrics

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

const MemoryMetricName = "memorybytes"

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
		Name:      MemoryMetricName,
		Unit:      UnitBytes,
		AppId:     appId,
		TimeStamp: time.Now().UnixNano(),
		Instances: insts,
	}
}
