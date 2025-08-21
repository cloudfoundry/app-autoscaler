package models_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BindingParameters", func() {
	var (
		bindingParameters *BindingParameters
		err               error
	)

	Context("NewBindingParameters", func() {
		var (
			scalingPolicy ScalingPolicy
			config        BindingConfig
		)

		JustBeforeEach(func() {
			bindingParameters = NewBindingParameters(config, scalingPolicy)
		})

		Context("with nil scaling policy", func() {
			It("should create binding parameters with default policy", func() {
				Expect(bindingParameters).NotTo(BeNil())
				Expect(bindingParameters.GetConfiguration()).To(Equal(config))
				Expect(bindingParameters.GetScalingPolicy()).To(BeNil())
			})
		})

		Context("with valid scaling policy", func() {
			BeforeEach(func() {
				scalingPolicy = *NewScalingPolicy(DefaultCustomMetricsStrategy, nil)
			})

			It("should create binding parameters with provided scaling policy", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingParameters).NotTo(BeNil())
				Expect(bindingParameters.GetConfiguration()).To(Equal(config))
				Expect(bindingParameters.GetScalingPolicy()).To(Equal(scalingPolicy))
			})
		})
	})

	Context("BindingParametersFromRawJSON",func() {
		var (
			rawJSON []byte
			config  BindingConfig
			scalingPolicy *PolicyDefinition
		)

		JustBeforeEach(func() {
			bindingParameters, err = BindingParametersFromRawJSON(rawJSON)
		})

		Context("with valid configuration and without policy", func() {
			BeforeEach(func() {
				rawJSON = []byte(`{"binding_cfg": {"app_guid": "test-app-guid"}}`)
				config = *NewBindingConfig(GUID("test-app-guid"), DefaultCustomMetricsStrategy)
				scalingPolicy = nil
			})

			It("should create binding parameters successfully", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingParameters).NotTo(BeNil())
				Expect(bindingParameters.GetConfiguration()).To(Equal(config))
			})
		})

		Context("with valid configuration and scaling policy", func() {
			BeforeEach(func() {
				rawJSON = []byte(`
{
  "configuration": {
	  "app_guid": "test-app-guid",
	  "custom_metrics": {
		"metric_submission_strategy": {
			"allow_from": "bound_app"
		}
	  }
  },

  "instance_min_count": 1,
  "instance_max_count": 4,
  "scaling_rules": [
	{
	  "metric_type": "memoryutil",
	  "breach_duration_secs": 600,
	  "threshold": 30,
	  "operator": "<",
	  "cool_down_secs": 300,
	  "adjustment": "-1"
	},
	{
	  "metric_type": "memoryutil",
	  "breach_duration_secs": 600,
	  "threshold": 90,
	  "operator": ">=",
	  "cool_down_secs": 300,
	  "adjustment": "+1"
	}
  ]
}`)
				config = *NewBindingConfig(GUID("test-app-guid"), CustomMetricsBoundApp)
				scalingPolicy = &PolicyDefinition{
					InstanceMin: 1,
					InstanceMax: 4,
					ScalingRules: []*ScalingRule{
						{
							MetricType:            "memoryutil",
							BreachDurationSeconds: 600,
							Threshold:             30,
							Operator:              "<",
							CoolDownSeconds:       300,
							Adjustment:            "-1",
						},
						{
							MetricType:            "memoryutil",
							BreachDurationSeconds: 600,
							Threshold:             90,
							Operator:              ">=",
							CoolDownSeconds:       300,
							Adjustment:            "+1",
						},
					},
				}
			})

			It("should create binding parameters successfully", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingParameters).NotTo(BeNil())
				Expect(bindingParameters.GetConfiguration()).To(Equal(config))
				Expect(bindingParameters.GetScalingPolicy()).To(Equal(scalingPolicy))
			})
		})

		Context("with default configuration and with scaling policy", func() {
			BeforeEach(func() {
				rawJSON = []byte(`
{
  "instance_min_count": 1,
  "instance_max_count": 4,
  "scaling_rules": [
	{
	  "metric_type": "memoryutil",
	  "breach_duration_secs": 600,
	  "threshold": 30,
	  "operator": "<",
	  "cool_down_secs": 300,
	  "adjustment": "-1"
	},
	{
	  "metric_type": "memoryutil",
	  "breach_duration_secs": 600,
	  "threshold": 90,
	  "operator": ">=",
	  "cool_down_secs": 300,
	  "adjustment": "+1"
	}
  ]
}`)
				config = *DefaultBindingConfig()
				scalingPolicy = &PolicyDefinition{
					InstanceMin: 1,
					InstanceMax: 4,
					ScalingRules: []*ScalingRule{
						{
							MetricType:            "memoryutil",
							BreachDurationSeconds: 600,
							Threshold:             30,
							Operator:              "<",
							CoolDownSeconds:       300,
							Adjustment:            "-1",
						},
						{
							MetricType:            "memoryutil",
							BreachDurationSeconds: 600,
							Threshold:             90,
							Operator:              ">=",
							CoolDownSeconds:       300,
							Adjustment:            "+1",
						},
					},
				}
			})

			It("should create binding parameters successfully", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingParameters).NotTo(BeNil())
				Expect(bindingParameters.GetConfiguration()).To(Equal(config))
				Expect(bindingParameters.GetScalingPolicy()).To(Equal(scalingPolicy))
			})
		})
	})
})
