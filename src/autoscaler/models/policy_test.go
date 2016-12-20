package models_test

import (
	. "autoscaler/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Policy", func() {

	var testAppId = "testAppId"
	var testAppIdAnother = "testAppIdAnother"
	var policy *AppPolicy
	var policyStr = `
   {
   "instance_min_count":1,
   "instance_max_count":5,
   "scaling_rules":[
      {
         "metric_type":"MemoryUsage",
         "stat_window_secs":300,
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
         "metric_type":"MemoryUsage",
         "stat_window_secs":300,
         "breach_duration_secs":600,
         "threshold":30,
         "operator":"<",
         "cool_down_secs":300,
         "adjustment":"-1"
      }
   ]
}`
	var p1, p2, policyJson *PolicyJson
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
			policy = policyJson.GetAppPolicy()
		})
		It("should return a policy", func() {
			Expect(policy).To(Equal(&AppPolicy{
				AppId: testAppId,
				ScalingPolicy: &ScalingPolicy{
					InstanceMax: 5,
					InstanceMin: 1,
					ScalingRules: []*ScalingRule{
						&ScalingRule{
							MetricType:            "MemoryUsage",
							StatWindowSeconds:     300,
							BreachDurationSeconds: 300,
							CoolDownSeconds:       300,
							Threshold:             30,
							Operator:              "<",
							Adjustment:            "-1",
						}}}}))
		})

	})

})
