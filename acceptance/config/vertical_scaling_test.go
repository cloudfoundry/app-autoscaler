package config_test

import (
	. "acceptance/config"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Vertical scaling acceptance configuration", func() {
	BeforeEach(func() {
		GinkgoT().Setenv("ACCEPTANCE_CONFIG_JSON", config())
	})

	It("requires explicit opt-in and foundation confirmation", func() {
		cfg := LoadConfig(AbortSuite)
		Expect(cfg.VerticalScaling.Enabled).To(BeFalse())
		Expect(cfg.VerticalScaling.DeploymentScalingSupported).To(BeFalse())
		Expect(cfg.VerticalScaling.SafetyScenariosEnabled).To(BeFalse())
		Expect(cfg.VerticalScaling.BreachDurationSeconds).To(Equal(60))
		Expect(cfg.VerticalScaling.CoolDownSeconds).To(Equal(60))
	})

	It("loads independent safety opt-in and policy durations", func() {
		Expect(os.Setenv("ACCEPTANCE_CONFIG_JSON", configWith(`"vertical_scaling_test": {
			"enabled": true, "deployment_scaling_supported": true,
			"safety_scenarios_enabled": true, "breach_duration_secs": 120, "cool_down_secs": 300
		}`))).To(Succeed())
		cfg := LoadConfig(AbortSuite)
		Expect(cfg.VerticalScaling.Enabled).To(BeTrue())
		Expect(cfg.VerticalScaling.DeploymentScalingSupported).To(BeTrue())
		Expect(cfg.VerticalScaling.SafetyScenariosEnabled).To(BeTrue())
		Expect(cfg.VerticalScaling.BreachDurationSeconds).To(Equal(120))
		Expect(cfg.VerticalScaling.CoolDownSeconds).To(Equal(300))
	})

	DescribeTable("honours global policy duration bounds", func(field string, duration int) {
		content := configWith(`"vertical_scaling_test": {"enabled": true, "` + field + `": ` + strconv.Itoa(duration) + `}`)
		Expect(os.Setenv("ACCEPTANCE_CONFIG_JSON", content)).To(Succeed())
		loadConfigExpectSuiteTerminationWith("vertical_scaling_test." + field + " must be between 60 and 3600")
	},
		Entry("short breach", "breach_duration_secs", 1),
		Entry("negative cooldown", "cool_down_secs", -1),
		Entry("long breach", "breach_duration_secs", 3601),
		Entry("long cooldown", "cool_down_secs", 3601),
	)
})
