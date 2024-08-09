package models

type CustomMetric struct {
	Name          string  `json:"name"`
	Value         float64 `json:"value"`
	Unit          string  `json:"unit"`
	AppGUID       string  `json:"app_guid"`
	InstanceIndex uint32  `json:"instance_index"`
}

type MetricsConsumer struct {
	InstanceIndex uint32          `json:"instance_index"`
	CustomMetrics []*CustomMetric `json:"metrics"`
}

type Credential struct {
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

type CustomMetricsCredentials struct {
	*Credential
	URL     *string `json:"url,omitempty"`
	MtlsUrl string  `json:"mtls_url"`
}
type Credentials struct {
	CustomMetrics CustomMetricsCredentials `json:"custom_metrics"`
}
type CredentialResponse struct {
	Credentials Credentials `json:"credentials"`
}

type CredentialsOptions struct {
	InstanceId string `json:"instanceId"`
	BindingId  string `json:"bindingId"`
}
