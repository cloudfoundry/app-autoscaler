package models

import (
	"fmt"
)

const (
	UnitPercentage   = "%"
	UnitMegaBytes    = "MB"
	UnitNum          = ""
	UnitMilliseconds = "ms"
	UnitRPS          = "rps"

	MetricNameMemoryUtil   = "memoryutil"
	MetricNameMemoryUsed   = "memoryused"
	MetricNameThroughput   = "throughput"
	MetricNameResponseTime = "responsetime"

	MetricLabelAppID         = "app_id"
	MetricLabelInstanceIndex = "instance_index"
	MetricLabelName          = "name"
)

type AppInstanceMetric struct {
	AppId         string `json:"app_id"`
	InstanceIndex uint32 `json:"instance_index"`
	CollectedAt   int64  `json:"collected_at"`
	Name          string `json:"name"`
	Unit          string `json:"unit"`
	Value         string `json:"value"`
	Timestamp     int64  `json:"timestamp"`
}

func (m *AppInstanceMetric) GetTimestamp() int64 {
	return m.Timestamp
}

func (m *AppInstanceMetric) HasLabels(labels map[string]string) bool {
	for k, v := range labels {
		switch k {
		case MetricLabelAppID:
			if v == m.AppId {
				continue
			} else {
				return false
			}
		case MetricLabelInstanceIndex:
			if v == fmt.Sprintf("%d", m.InstanceIndex) {
				continue
			} else {
				return false
			}
		case MetricLabelName:
			if v == m.Name {
				continue
			} else {
				return false
			}
		default:
			return false
		}
	}
	return true
}

type AppMetric struct {
	AppId      string `json:"app_id"`
	MetricType string `json:"name"`
	Value      string `json:"value"`
	Unit       string `json:"unit"`
	Timestamp  int64  `json:"timestamp"`
}

func (m *AppMetric) GetTimestamp() int64 {
	return m.Timestamp
}

func (m *AppMetric) HasLabels(labels map[string]string) bool {
	for k, v := range labels {
		switch k {
		case MetricLabelAppID:
			if v == m.AppId {
				continue
			} else {
				return false
			}
		case MetricLabelName:
			if v == m.MetricType {
				continue
			} else {
				return false
			}
		default:
			return false
		}
	}
	return true
}
