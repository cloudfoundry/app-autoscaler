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
	Username string `json:"username"`
	Password string `json:"password"`
}

//custom metrics response
//  {
//     credentials: {
//         custom_metrics:{
//           username: username,
//           password: password,
//           url: settings.customMetricsUrl
//         }
//     }
// }

type CustomMetrics struct {
	*CustomMetricCredentials
	URL string `json:"url"`
}
type Credentials struct {
	CustomMetrics CustomMetrics `json:"custom_metrics"`
}
type CredentialResponse struct {
	Credentials Credentials `json:"credentials"`
}
