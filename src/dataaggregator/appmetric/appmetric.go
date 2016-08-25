package appmetric

type AppMonitor struct {
	AppId          string
	MetricType     string
	StatWindowSecs int64
}
type AppMetric struct {
	AppId      string
	MetricType string
	Value      int64
	Unit       string
	Timestamp  int64
}
