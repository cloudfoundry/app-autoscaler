package policy_test

import (
	. "dataaggregator/policy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Policy", func() {

	var testAppId = "testAppId"
	var testAppIdAnother = "testAppIdAnother"
	var trigger *Trigger
	var policyStr = `
   {
   "instance_min_count":1,
   "instance_max_count":5,
   "scaling_rules":[
      {
         "metric_type":"MemoryUsage",
         "stat_window":300,
         "breach_duration":300,
         "threshold":30,
         "operator":"<",
         "cool_down_duration":300,
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
         "stat_window":300,
         "breach_duration":600,
         "threshold":30,
         "operator":"<",
         "cool_down_duration":300,
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
	Context("GetTriggerFromPolicy", func() {

		BeforeEach(func() {
			policyJson = &PolicyJson{AppId: testAppId, PolicyStr: policyStr}
			trigger = policyJson.GetTrigger()
		})
		It("should return a trigger", func() {
			Expect(trigger).To(Equal(&Trigger{AppId: testAppId, TriggerRecord: &TriggerRecord{InstanceMaxCount: 5, InstanceMinCount: 1, ScalingRules: []*ScalingRule{&ScalingRule{MetricType: "MemoryUsage", StatWindow: 300, BreachDuration: 300, CoolDownDuration: 300, Threshold: 30, Operator: "<", Adjustment: "-1"}}}}))
		})

	})

})
