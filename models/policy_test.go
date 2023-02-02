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
	var policyStr = `
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
	var policyStrAnother = `
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
				p1 = &PolicyJson{PolicyStr: policyStr}
				p2 = &PolicyJson{PolicyStr: policyStr}
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
				p1 = &PolicyJson{PolicyStr: policyStr}
				p2 = &PolicyJson{PolicyStr: policyStrAnother}
			})
			It("should return false", func() {
				Expect(p1.Equals(p2)).To(Equal(false))
			})
		})

	})
	Context("GetAppPolicy", func() {

		BeforeEach(func() {
			policyJson = &PolicyJson{AppId: testAppId, PolicyStr: policyStr}
			policy, err = policyJson.GetAppPolicy()
		})
		It("should return a policy", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(policy).To(Equal(&AppPolicy{
				AppId: testAppId,
				ScalingPolicy: &ScalingPolicy{
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
				policyJson = &PolicyJson{AppId: testAppId, PolicyStr: policyStr}
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
