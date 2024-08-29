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
	MetricNameCPU          = "cpu"
	MetricNameCPUUtil      = "cpuutil"
	MetricNameThroughput   = "throughput"
	MetricNameResponseTime = "responsetime"
	MetricNameDiskUtil     = "diskutil"
	MetricNameDisk         = "disk"

	MetricLabelAppID         = "app_id"
	MetricLabelInstanceIndex = "instance_index"
	MetricLabelName          = "name"
)

type AppInstanceMetric struct {
	AppId         string `json:"app_id" db:"app_id"`
	InstanceIndex uint64 `json:"instance_index" db:"instance_index"`
	CollectedAt   int64  `json:"collected_at" db:"collected_at"`
	Name          string `json:"name" db:"name"`
	Unit          string `json:"unit" db:"unit"`
	Value         string `json:"value" db:"value"`
	Timestamp     int64  `json:"timestamp" db:"timestamp"`
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
	AppId      string `json:"app_id" db:"app_id"`
	MetricType string `json:"name" db:"metric_type"`
	Value      string `json:"value" db:"value"`
	Unit       string `json:"unit" db:"unit"`
	Timestamp  int64  `json:"timestamp" db:"timestamp"`
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
