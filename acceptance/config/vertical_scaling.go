package config

// VerticalScalingConfig is deliberately disabled by default. The operator must
// confirm deployment scaling and Diego readiness support on a disposable test
// foundation; a CAPI V3 API version is not a capi-release version.
type VerticalScalingConfig struct {
	Enabled                    bool `json:"enabled"`
	DeploymentScalingSupported bool `json:"deployment_scaling_supported"`
	SafetyScenariosEnabled     bool `json:"safety_scenarios_enabled"`
	BreachDurationSeconds      int  `json:"breach_duration_secs"`
	CoolDownSeconds            int  `json:"cool_down_secs"`
}

func (c *VerticalScalingConfig) validate(terminateSuite TerminateSuite) {
	if c.BreachDurationSeconds == 0 {
		c.BreachDurationSeconds = 60
	}
	if c.CoolDownSeconds == 0 {
		c.CoolDownSeconds = 60
	}
	if !c.Enabled {
		return
	}
	// Match the existing policy validator's global duration bounds, not just
	// the positive-duration constraint of vertical rules.
	if c.BreachDurationSeconds < 60 || c.BreachDurationSeconds > 3600 {
		terminateSuite("vertical_scaling_test.breach_duration_secs must be between 60 and 3600")
	}
	if c.CoolDownSeconds < 60 || c.CoolDownSeconds > 3600 {
		terminateSuite("vertical_scaling_test.cool_down_secs must be between 60 and 3600")
	}
}
