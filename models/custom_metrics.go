package models

type CustomMetric struct {
	Name          string  `json:"name"`
	Value         float64 `json:"value"`
	Unit          string  `json:"unit"`
	AppGUID       string  `json:"app_guid"`
	InstanceIndex uint32  `json:"instance_index"`
}

type MetricsConsumer struct {
	AppGUID       string          `json:"app_guid"`
	InstanceIndex uint32          `json:"instance_index"`
	CustomMetrics []*CustomMetric `json:"metrics"`
}

type CustomMetricCredentials struct {
	Username string
	Password string
}
