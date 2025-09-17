package models_test

import (
	"encoding/json"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppScalingConfig", func() {
	var (
		appScalingCfg *AppScalingConfig
		err           error
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

	Context("ToRawJSON", func() {
		var rawJSON json.RawMessage
		var rawJSONString string

		Context("with default configuration and policy", func() {
			BeforeEach(func() {
				config := *NewBindingConfig(testAppGUID, nil)
				scalingPolicy := *NewScalingPolicy(DefaultCustomMetricsStrategy, nil)
				appScalingCfg = NewAppScalingConfig(config, scalingPolicy)
				rawJSON, err = appScalingCfg.ToRawJSON()
				rawJSONString = string(rawJSON)
			})

			It("should serialize to raw JSON without error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rawJSON).NotTo(BeNil())
				Expect(rawJSONString).To(ContainSubstring(`"binding-configuration"`))
				Expect(rawJSONString).NotTo(ContainSubstring(`"scaling-policy"`))
			})

			It("should not include configuration section for default custom metrics strategy", func() {
				// // We would love to check the absence of the substing "configuration" here, but it
				// // always will be present, because we have a binding-configuration section.
				// Expect(rawJSONString).NotTo(ContainSubstring("configuration"))
				Expect(rawJSONString).NotTo(ContainSubstring("custom_metrics"))
			})
		})

		Context("with custom configuration and policy", func() {
			BeforeEach(func() {
				config := *NewBindingConfig(testAppGUID, &BindingSecret)
				policyDef := &PolicyDefinition{
					InstanceMin: 1,
					InstanceMax: 5,
					ScalingRules: []*ScalingRule{
						{
							MetricType:            "cpu",
							BreachDurationSeconds: 600,
							Threshold:             75,
							Operator:              ">",
							CoolDownSeconds:       400,
							Adjustment:            "+2",
						},
					},
				}
				scalingPolicy := *NewScalingPolicy(CustomMetricsBoundApp, policyDef)
				appScalingCfg = NewAppScalingConfig(config, scalingPolicy)
				rawJSON, err = appScalingCfg.ToRawJSON()
				rawJSONString = string(rawJSON)
			})

			It("should serialize to raw JSON without error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rawJSON).NotTo(BeNil())
				Expect(rawJSONString).To(ContainSubstring(`"binding-configuration"`))
				Expect(rawJSONString).To(ContainSubstring(`"scaling-policy"`))
			})

			It("should include configuration section for non-default custom metrics strategy", func() {
				Expect(rawJSONString).To(ContainSubstring("configuration"))
				Expect(rawJSONString).To(ContainSubstring("custom_metrics"))
				Expect(rawJSONString).To(ContainSubstring("bound_app"))
			})
		})
	})

	Context("AppScalingConfigFromRawJSON", func() {
		var rawJSON json.RawMessage

		Context("with valid configuration and policy", func() {
			BeforeEach(func() {
				rawJSON = []byte(`{
  "binding-configuration": {
	"app_guid": "test-app-guid",
	"credential-type": "x509"
  },
  "scaling-policy": {
	"configuration": {
	  "custom_metrics": {
		"metric_submission_strategy": {
		  "allow_from": "bound_app"
		}
	  }
	},
	"instance_min_count": 2,
	"instance_max_count": 6,
	"scaling_rules": [
	  {
		"metric_type": "memoryutil",
		"breach_duration_secs": 400,
		"threshold": 85,
		"operator": ">",
		"cool_down_secs": 350,
		"adjustment": "+1"
	  }
	]
  }
}`)
			})

			It("should deserialize from raw JSON without error", func() {
				appScalingCfg, err = AppScalingConfigFromRawJSON(rawJSON)
				Expect(err).NotTo(HaveOccurred())
				Expect(appScalingCfg).NotTo(BeNil())

				config := appScalingCfg.GetConfiguration()
				Expect(config.GetAppGUID()).To(Equal(testAppGUID))
				Expect(*config.GetCustomMetricStrategy()).To(Equal(X509Certificate))

				policy := appScalingCfg.GetScalingPolicy()
				Expect(policy.GetCustomMetricsStrategy()).To(Equal(CustomMetricsBoundApp))

				policyDef := policy.GetPolicyDefinition()
				Expect(policyDef).NotTo(BeNil())
				Expect(policyDef.InstanceMin).To(Equal(2))
				Expect(policyDef.InstanceMax).To(Equal(6))
				Expect(len(policyDef.ScalingRules)).To(Equal(1))
				Expect(policyDef.ScalingRules[0].MetricType).To(Equal("memoryutil"))
			})
		})

		Context("with minimal configuration", func() {
			BeforeEach(func() {
				rawJSON = []byte(`{
  "binding-configuration": {
	"app_guid": "test-app-guid"
  },
  "scaling-policy": {}
}`)
			})

			It("should deserialize with default values", func() {
				appScalingCfg, err = AppScalingConfigFromRawJSON(rawJSON)
				Expect(err).NotTo(HaveOccurred())
				Expect(appScalingCfg).NotTo(BeNil())

				config := appScalingCfg.GetConfiguration()
				Expect(config.GetAppGUID()).To(Equal(testAppGUID))
				Expect(config.GetCustomMetricStrategy()).To(BeNil())

				policy := appScalingCfg.GetScalingPolicy()
				Expect(policy.GetCustomMetricsStrategy()).To(Equal(DefaultCustomMetricsStrategy))
				Expect(policy.GetPolicyDefinition()).To(BeNil())
			})
		})

		Context("with invalid JSON", func() {
			BeforeEach(func() {
				rawJSON = []byte(`{"invalid_json"}`)
			})

			It("should return an error", func() {
				appScalingCfg, err = AppScalingConfigFromRawJSON(rawJSON)
				Expect(err).To(HaveOccurred())
				Expect(appScalingCfg).To(BeNil())
			})
		})
	})

	Context("ToRawJSON and AppScalingConfigFromRawJSON", func() {
		When("executed in succession", func() {
			var appScalingCfg1, appScalingCfg2 *AppScalingConfig
			var rawJSON json.RawMessage

			BeforeEach(func() {
				config := *NewBindingConfig(testAppGUID, &BindingSecret)
				policyDef := &PolicyDefinition{
					InstanceMin: 3,
					InstanceMax: 7,
					ScalingRules: []*ScalingRule{
						{
							MetricType:            "cpu",
							BreachDurationSeconds: 500,
							Threshold:             70,
							Operator:              ">",
							CoolDownSeconds:       300,
							Adjustment:            "+1",
						},
					},
				}
				scalingPolicy := *NewScalingPolicy(CustomMetricsBoundApp, policyDef)
				appScalingCfg1 = NewAppScalingConfig(config, scalingPolicy)

				rawJSON, err = appScalingCfg1.ToRawJSON()
				Expect(err).NotTo(HaveOccurred())

				appScalingCfg2, err = AppScalingConfigFromRawJSON(rawJSON)
			})

			It("should return an equivalent AppScalingConfig", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appScalingCfg2).NotTo(BeNil())

				// Compare configurations
				config1 := appScalingCfg1.GetConfiguration()
				config2 := appScalingCfg2.GetConfiguration()
				Expect(config2).To(Equal(config1))

				// Compare scaling policies
				policy1 := appScalingCfg1.GetScalingPolicy()
				policy2 := appScalingCfg2.GetScalingPolicy()
				Expect(policy2.GetCustomMetricsStrategy()).To(Equal(policy1.GetCustomMetricsStrategy()))

				policyDef1 := policy1.GetPolicyDefinition()
				policyDef2 := policy2.GetPolicyDefinition()
				if policyDef1 == nil {
					Expect(policyDef2).To(BeNil())
				} else {
					Expect(policyDef2).NotTo(BeNil())
					Expect(*policyDef2).To(Equal(*policyDef1))
				}
			})
		})
	})
})
