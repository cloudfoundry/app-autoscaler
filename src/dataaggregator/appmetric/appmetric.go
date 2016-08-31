package appmetric

import (
	"time"
)

type AppMonitor struct {
	AppId      string
	MetricType string
	StatWindow time.Duration
}
type AppMetric struct {
	AppId      string
	MetricType string
	Value      int64
	Unit       string
	Timestamp  int64
}
