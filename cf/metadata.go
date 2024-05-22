package cf

type Metadata struct {
	Labels `json:"labels"`
}
type Labels struct {
	DisableAutoscaling *string `json:"app-autoscaler.cloudfoundry.org/disable-autoscaling"`
}
