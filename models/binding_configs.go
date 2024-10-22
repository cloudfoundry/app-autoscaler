package models

// BindingConfig
/* The configuration object received as part of the binding parameters. Example config:
{
  "configuration": {
    "custom_metrics": {
      "auth": {
        "credential_type": "binding_secret"
      },
      "metric_submission_strategy": {
        "allow_from": "bound_app or same_app"
      }
    }
  }
*/

const (
	CustomMetricsBoundApp = "bound_app"
	CustomMetricsSameApp  = "same_app"
)

type BindingConfig struct {
	Configuration Configuration `json:"configuration"`
}
type Configuration struct {
	CustomMetrics CustomMetricsConfig `json:"custom_metrics"`
}

type CustomMetricsConfig struct {
	MetricSubmissionStrategy MetricsSubmissionStrategy `json:"metric_submission_strategy"`
}

type MetricsSubmissionStrategy struct {
	AllowFrom string `json:"allow_from"`
}

func (b *BindingConfig) GetCustomMetricsStrategy() string {
	return b.Configuration.CustomMetrics.MetricSubmissionStrategy.AllowFrom
}

func (b *BindingConfig) SetCustomMetricsStrategy(allowFrom string) {
	b.Configuration.CustomMetrics.MetricSubmissionStrategy.AllowFrom = allowFrom
}
