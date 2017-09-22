package models

const (
	UnitPercentage   = "%"
	UnitMegaBytes    = "MB"
	UnitNum          = ""
	UnitMilliseconds = "ms"
	UnitRPS          = "rps"
)

const MetricNameCpuPercentage = "cpuPercentage"
const MetricNameMemoryUtil = "memoryutil"
const MetricNameMemoryUsed = "memoryused"
const MetricNameThroughput = "throughput"
const MetricNameResponseTime = "responsetime"

type AppInstanceMetric struct {
	AppId         string `json:"app_id"`
	InstanceIndex uint32 `json:"instance_index"`
	CollectedAt   int64  `json:"collected_at"`
	Name          string `json:"name"`
	Unit          string `json:"unit"`
	Value         string `json:"value"`
	Timestamp     int64  `json:"timestamp"`
}
