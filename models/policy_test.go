package models_test

import (
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Policy", func() {
	const (
		DefaultStatWindowSecs     = 400
		DefaultBreachDurationSecs = 450
		DefaultCoolDownSecs       = 500
	)

	var testAppId = "testAppId"
	var testAppIdAnother = "testAppIdAnother"
	var policy *AppPolicy
	var policyDefStr = `
   {
   "instance_min_count":1,
   "instance_max_count":5,
   "scaling_rules":[
	  {
		 "metric_type":"memoryused",
		 "breach_duration_secs":300,
		 "threshold":30,
		 "operator":"<",
		 "cool_down_secs":300,
		 "adjustment":"-1"
	  }
   ]
}`
	var policyDefStrAnother = `
   {
   "instance_min_count":2,
   "instance_max_count":5,
   "scaling_rules":[
	  {
		 "metric_type":"memoryused",
		 "breach_duration_secs":600,
		 "threshold":30,
		 "operator":"<",
		 "cool_down_secs":300,
		 "adjustment":"-1"
	  }
   ]
}`

	var policyStrMinimalScalingRuleParameter = `
{
"instance_min_count":2,
"instance_max_count":5,
"scaling_rules":[
   {
	  "metric_type":"memoryused",
	  "threshold":30,
	  "operator":"<",
	  "adjustment":"-1"
   }
]
}`
	var p1, p2, policyJson *PolicyJson
	var err error

	Context("JsonSerialisation", func() {
		// Schreibe Tests, die Überprüfen, ob eine ScalingPolicy korrekt deserialisiert wird.
		// Schreibe außerdem Tests, die überprüfen, dass eine ScalingPolicy, deren CustomMetricsStrategy der Standardstrategie entspricht, ohne policyCfg-Teil serialisiert wird.

		var scalingPolicy *ScalingPolicy
		var serializedData []byte
		var deserializedPolicy *ScalingPolicy

		Context("when deserializing a ScalingPolicy", func() {
			Context("with default configuration", func() {
				BeforeEach(func() {
					policyDef := &PolicyDefinition{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*ScalingRule{
							{
								MetricType:            "memoryused",
								BreachDurationSeconds: 300,
								Threshold:             30,
								Operator:              "<",
								CoolDownSeconds:       300,
								Adjustment:            "-1",
							},
						},
					}
					scalingPolicy = NewScalingPolicy(DefaultCustomMetricsStrategy, policyDef)
					serializedData, err = scalingPolicy.ToRawJSON()
					Expect(err).NotTo(HaveOccurred())
				})

				It("should deserialize correctly", func() {
					deserializedPolicy, err = ScalingPolicyFromRawJSON(serializedData)
					Expect(err).NotTo(HaveOccurred())
					Expect(deserializedPolicy).NotTo(BeNil())

					originalPolicyDef := scalingPolicy.GetPolicyDefinition()
					deserializedPolicyDef := deserializedPolicy.GetPolicyDefinition()

					Expect(deserializedPolicyDef.InstanceMin).To(Equal(originalPolicyDef.InstanceMin))
					Expect(deserializedPolicyDef.InstanceMax).To(Equal(originalPolicyDef.InstanceMax))
					Expect(len(deserializedPolicyDef.ScalingRules)).To(Equal(len(originalPolicyDef.ScalingRules)))
					Expect(deserializedPolicyDef.ScalingRules[0].MetricType).To(Equal(originalPolicyDef.ScalingRules[0].MetricType))
				})
			})

			Context("with custom configuration", func() {
				BeforeEach(func() {
					policyDef := &PolicyDefinition{
						InstanceMin: 2,
						InstanceMax: 8,
						ScalingRules: []*ScalingRule{
							{
								MetricType:            "cpu",
								BreachDurationSeconds: 600,
								Threshold:             80,
								Operator:              ">",
								CoolDownSeconds:       400,
								Adjustment:            "+2",
							},
						},
					}
					scalingPolicy = NewScalingPolicy(CustomMetricsBoundApp, policyDef)
					serializedData, err = scalingPolicy.ToRawJSON()
					Expect(err).NotTo(HaveOccurred())
				})

				It("should deserialize correctly", func() {
					deserializedPolicy, err = ScalingPolicyFromRawJSON(serializedData)
					Expect(err).NotTo(HaveOccurred())
					Expect(deserializedPolicy).NotTo(BeNil())

					originalPolicyDef := scalingPolicy.GetPolicyDefinition()
					deserializedPolicyDef := deserializedPolicy.GetPolicyDefinition()

					Expect(deserializedPolicyDef.InstanceMin).To(Equal(originalPolicyDef.InstanceMin))
					Expect(deserializedPolicyDef.InstanceMax).To(Equal(originalPolicyDef.InstanceMax))
					Expect(deserializedPolicyDef.ScalingRules[0].MetricType).To(Equal(originalPolicyDef.ScalingRules[0].MetricType))
					Expect(deserializedPolicyDef.ScalingRules[0].Threshold).To(Equal(originalPolicyDef.ScalingRules[0].Threshold))
				})
			})

			Context("with nil policy definition", func() {
				BeforeEach(func() {
					scalingPolicy = NewScalingPolicy(CustomMetricsBoundApp, nil)
					serializedData, err = scalingPolicy.ToRawJSON()
					Expect(err).NotTo(HaveOccurred())
				})

				It("should deserialize to policy with nil scaling policy", func() {
					deserializedPolicy, err = ScalingPolicyFromRawJSON(serializedData)
					Expect(err).NotTo(HaveOccurred())
					Expect(deserializedPolicy).NotTo(BeNil())
					Expect(deserializedPolicy.GetPolicyDefinition()).To(BeNil())
				})
			})
		})

		Context("when serializing a ScalingPolicy with default CustomMetricsStrategy", func() {
			BeforeEach(func() {
				policyDef := &PolicyDefinition{
					InstanceMin: 1,
					InstanceMax: 3,
				}
				scalingPolicy = NewScalingPolicy(DefaultCustomMetricsStrategy, policyDef)
				serializedData, err = scalingPolicy.ToRawJSON()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should not include configuration section in JSON", func() {
				jsonString := string(serializedData)
				Expect(jsonString).NotTo(ContainSubstring("configuration"))
				Expect(jsonString).NotTo(ContainSubstring("custom_metrics"))
			})

			It("should only contain policy definition", func() {
				jsonString := string(serializedData)
				Expect(jsonString).To(ContainSubstring("instance_min_count"))
				Expect(jsonString).To(ContainSubstring("instance_max_count"))
			})
		})

		Context("when serializing a ScalingPolicy with non-default CustomMetricsStrategy", func() {
			BeforeEach(func() {
				policyDef := &PolicyDefinition{
					InstanceMin: 1,
					InstanceMax: 3,
				}
				scalingPolicy = NewScalingPolicy(CustomMetricsBoundApp, policyDef)
				serializedData, err = scalingPolicy.ToRawJSON()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should include configuration section in JSON", func() {
				jsonString := string(serializedData)
				Expect(jsonString).To(ContainSubstring("configuration"))
				Expect(jsonString).To(ContainSubstring("custom_metrics"))
				Expect(jsonString).To(ContainSubstring("bound_app"))
			})
		})
	})

	Context("PolicyJson.Equals", func() {
		Context("when p1 and p2 are all nil", func() {
			BeforeEach(func() {
				p1 = nil
				p2 = nil
			})
			It("should return true", func() {
				Expect(p1.Equals(p2)).To(Equal(true))
			})
		})

		Context("when the AppIds are the same", func() {
			BeforeEach(func() {
				p1 = &PolicyJson{AppId: testAppId}
				p2 = &PolicyJson{AppId: testAppId}
			})
			It("should return true", func() {
				Expect(p1.Equals(p2)).To(Equal(true))
			})
		})
		Context("when the PolicyStrs are the same", func() {
			BeforeEach(func() {
				p1 = &PolicyJson{PolicyStr: policyDefStr}
				p2 = &PolicyJson{PolicyStr: policyDefStr}
			})
			It("should return false", func() {
				Expect(p1.Equals(p2)).To(Equal(true))
			})
		})
		Context("when the AppId are different", func() {
			BeforeEach(func() {
				p1 = &PolicyJson{AppId: testAppId}
				p2 = &PolicyJson{AppId: testAppIdAnother}
			})
			It("should return false", func() {
				Expect(p1.Equals(p2)).To(Equal(false))
			})
		})
		Context("when the PolicyStrs are the different", func() {
			BeforeEach(func() {
				p1 = &PolicyJson{PolicyStr: policyDefStr}
				p2 = &PolicyJson{PolicyStr: policyDefStrAnother}
			})
			It("should return false", func() {
				Expect(p1.Equals(p2)).To(Equal(false))
			})
		})

	})
	Context("GetAppPolicy", func() {

		BeforeEach(func() {
			policyJson = &PolicyJson{AppId: testAppId, PolicyStr: policyDefStr}
			policy, err = policyJson.GetAppPolicy()
		})
		It("should return a policy", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(policy).To(Equal(&AppPolicy{
				AppId: testAppId,
				ScalingPolicy: &PolicyDefinition{
					InstanceMax: 5,
					InstanceMin: 1,
					ScalingRules: []*ScalingRule{
						{
							MetricType:            "memoryused",
							BreachDurationSeconds: 300,
							CoolDownSeconds:       300,
							Threshold:             30,
							Operator:              "<",
							Adjustment:            "-1",
						}}}}))
		})

	})

	DescribeTable("IsEmpty tests",
		func(schedule *ScalingSchedules, value bool) { Expect(schedule.IsEmpty()).To(Equal(value)) },
		Entry("nul", nil, true),
		Entry("nil arrays", &ScalingSchedules{}, true),
		Entry("one recurring schedule", &ScalingSchedules{RecurringSchedules: []*RecurringSchedule{{}}}, false),
		Entry("one Specific schedule", &ScalingSchedules{SpecificDateSchedules: []*SpecificDateSchedule{{}}}, false),
	)

	Context("ScalingRules", func() {
		JustBeforeEach(func() {
			policy, err = policyJson.GetAppPolicy()
		})

		Context("When scaling rule has breach_duration_secs and cool_down_secs", func() {
			BeforeEach(func() {
				Expect(err).NotTo(HaveOccurred())
				policyJson = &PolicyJson{AppId: testAppId, PolicyStr: policyDefStr}
			})
			It("should return actual breach_duration_secs and cool_down_secs", func() {
				Expect(policy.ScalingPolicy.ScalingRules[0].BreachDuration(DefaultBreachDurationSecs)).To(Equal(
					time.Duration(policy.ScalingPolicy.ScalingRules[0].BreachDurationSeconds) * time.Second))

				Expect(policy.ScalingPolicy.ScalingRules[0].CoolDown(DefaultCoolDownSecs)).To(Equal(
					time.Duration(policy.ScalingPolicy.ScalingRules[0].CoolDownSeconds) * time.Second))
			})
		})
		Context("When scaling rule doesn't have  breach_duration_secs and cool_down_secs", func() {
			BeforeEach(func() {
				policyJson = &PolicyJson{AppId: testAppId, PolicyStr: policyStrMinimalScalingRuleParameter}
			})
			It("should return default breach_duration_secs and cool_down_secs", func() {
				Expect(policy.ScalingPolicy.ScalingRules[0].BreachDuration(DefaultBreachDurationSecs)).To(Equal(
					time.Duration(DefaultBreachDurationSecs) * time.Second))

				Expect(policy.ScalingPolicy.ScalingRules[0].CoolDown(DefaultCoolDownSecs)).To(Equal(
					time.Duration(DefaultCoolDownSecs) * time.Second))
			})
		})
	})
})
