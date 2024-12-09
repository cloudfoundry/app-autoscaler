package models_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BindingConfigs", func() {

	var bindingConfig *BindingConfig

	Context("GetBindingConfigAndPolicy", func() {
		var (
			scalingPolicy        *ScalingPolicy
			customMetricStrategy string
			result               interface{}
			err                  error
		)

		JustBeforeEach(func() {
			result, err = GetBindingConfigAndPolicy(scalingPolicy, customMetricStrategy)
		})

		When("both scaling policy and custom metric strategy are present", func() {
			BeforeEach(func() {
				scalingPolicy = &ScalingPolicy{
					InstanceMax: 5,
					InstanceMin: 1,
					ScalingRules: []*ScalingRule{
						{
							MetricType:            "memoryused",
							BreachDurationSeconds: 300,
							CoolDownSeconds:       300,
							Threshold:             30,
							Operator:              ">",
							Adjustment:            "-1",
						},
					},
				}
				customMetricStrategy = CustomMetricsBoundApp
			})

			It("should return combined configuration", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeAssignableToTypeOf(&ScalingPolicyWithBindingConfig{}))
				combinedConfig := result.(*ScalingPolicyWithBindingConfig)
				Expect(combinedConfig.ScalingPolicy).To(Equal(*scalingPolicy))
				Expect(combinedConfig.BindingConfig.GetCustomMetricsStrategy()).To(Equal(customMetricStrategy))
			})
		})

		When("only scaling policy is present", func() {
			BeforeEach(func() {
				scalingPolicy = &ScalingPolicy{
					InstanceMax: 5,
					InstanceMin: 1,
					ScalingRules: []*ScalingRule{
						{
							MetricType:            "memoryused",
							BreachDurationSeconds: 300,
							CoolDownSeconds:       300,
							Threshold:             30,
							Operator:              ">",
							Adjustment:            "-1",
						},
					},
				}
				customMetricStrategy = ""
			})

			It("should return scaling policy", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(&ScalingPolicyWithBindingConfig{ScalingPolicy: *scalingPolicy, BindingConfig: nil}))
			})
		})

		When("policy is not found", func() {
			BeforeEach(func() {
				scalingPolicy = nil
				customMetricStrategy = CustomMetricsBoundApp
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("policy not found"))
			})
		})
	})

	Context("GetCustomMetricsStrategy", func() {
		It("should return the correct custom metrics strategy", func() {
			bindingConfig = &BindingConfig{
				Configuration: Configuration{
					CustomMetrics: CustomMetricsConfig{
						MetricSubmissionStrategy: MetricsSubmissionStrategy{
							AllowFrom: CustomMetricsBoundApp,
						},
					},
				},
			}
			Expect(bindingConfig.GetCustomMetricsStrategy()).To(Equal(CustomMetricsBoundApp))
		})
	})

	Context("SetCustomMetricsStrategy", func() {
		It("should set the custom metrics strategy correctly", func() {
			bindingConfig = &BindingConfig{}
			bindingConfig.SetCustomMetricsStrategy(CustomMetricsBoundApp)
			Expect(bindingConfig.Configuration.CustomMetrics.MetricSubmissionStrategy.AllowFrom).To(Equal(CustomMetricsBoundApp))
		})
	})

	Context("ValidateOrGetDefaultCustomMetricsStrategy", func() {
		var (
			validatedBindingConfig *BindingConfig
			err                    error
		)
		JustBeforeEach(func() {
			validatedBindingConfig, err = bindingConfig.ValidateOrGetDefaultCustomMetricsStrategy()
		})
		When("custom metrics strategy is empty", func() {

			BeforeEach(func() {
				bindingConfig = &BindingConfig{}
			})
			It("should set the default custom metrics strategy", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(validatedBindingConfig.GetCustomMetricsStrategy()).To(Equal(CustomMetricsSameApp))
			})
		})

		When("custom metrics strategy is unsupported", func() {
			BeforeEach(func() {
				bindingConfig = &BindingConfig{
					Configuration: Configuration{
						CustomMetrics: CustomMetricsConfig{
							MetricSubmissionStrategy: MetricsSubmissionStrategy{
								AllowFrom: "unsupported_strategy",
							},
						},
					},
				}
			})
			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("error: custom metrics strategy not supported"))
			})
		})
	})
})
