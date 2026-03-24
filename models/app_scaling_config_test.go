package models_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppScalingConfig", func() {
	var (
		appScalingCfg *AppScalingConfig
		testAppGUID   = GUID("test-app-guid")
	)

	Context("NewAppScalingConfig", func() {
		var (
			scalingPolicy ScalingPolicy
			config        BindingConfig
		)

		JustBeforeEach(func() {
			appScalingCfg = NewAppScalingConfig(config, scalingPolicy)
		})

		Context("with default scaling policy", func() {
			BeforeEach(func() {
				config = *NewBindingConfig(testAppGUID, nil)
				scalingPolicy = *NewScalingPolicy(DefaultCustomMetricsStrategy, nil)
			})

			It("should create app scaling config with default policy", func() {
				Expect(appScalingCfg).NotTo(BeNil())
				Expect(appScalingCfg.GetConfiguration()).To(Equal(&config))
				Expect(appScalingCfg.GetScalingPolicy()).NotTo(BeNil())
				Expect(appScalingCfg.GetScalingPolicy().GetPolicyDefinition()).To(BeNil())
			})
		})

		Context("with valid scaling policy", func() {
			BeforeEach(func() {
				config = *NewBindingConfig(testAppGUID, &X509Certificate)
				policyDef := &PolicyDefinition{
					InstanceMin: 2,
					InstanceMax: 8,
					ScalingRules: []*ScalingRule{
						{
							MetricType:            "memoryutil",
							BreachDurationSeconds: 300,
							Threshold:             80,
							Operator:              ">",
							CoolDownSeconds:       300,
							Adjustment:            "+1",
						},
					},
				}
				scalingPolicy = *NewScalingPolicy(CustomMetricsBoundApp, policyDef)
			})

			It("should create app scaling config with provided policy", func() {
				Expect(appScalingCfg).NotTo(BeNil())
				Expect(appScalingCfg.GetConfiguration()).To(Equal(&config))
				Expect(appScalingCfg.GetScalingPolicy()).NotTo(BeNil())
				Expect(appScalingCfg.GetScalingPolicy().GetPolicyDefinition()).NotTo(BeNil())
				Expect(appScalingCfg.GetScalingPolicy().GetCustomMetricsStrategy()).To(Equal(CustomMetricsBoundApp))
			})
		})
	})
})
