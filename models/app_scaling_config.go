package models

// AppScalingConfig contains all the necessary data to establish the relationship between
// application binding configuration and its associated scaling behavior.
//
// ⛔ Do not create `AppScalingConfig` values directly via `AppScalingConfig{}` because it can lead to
// undefined behaviour due to bypassing all validations. Use the constructor-functions instead!
type AppScalingConfig struct {
	// configuration contains the binding configuration settings.
	configuration BindingConfig

	// scalingPolicy defines the scaling behavior and rules for the binding.
	scalingPolicy ScalingPolicy
}

func NewAppScalingConfig(
	configuration BindingConfig, scalingPolicy ScalingPolicy,
) (bps *AppScalingConfig) {
	return &AppScalingConfig{
		configuration: configuration,
		scalingPolicy: scalingPolicy,
	}
}

func (asc *AppScalingConfig) GetConfiguration() *BindingConfig {
	return &asc.configuration
}

// GetScalingPolicy returns the scaling policy for the binding and nil if no one has been set (which
// means, the default-policy is used).
func (asc *AppScalingConfig) GetScalingPolicy() (p *ScalingPolicy) {
	return &asc.scalingPolicy
}
