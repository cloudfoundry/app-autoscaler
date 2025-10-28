package post_upgrade_test

import (
	"acceptance/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"time"
)

var _ = Describe("AutoScaler dynamic policy", func() {

	var (
		appName string
		appGUID string
	)

	JustBeforeEach(func() {
		appName, appGUID = GetAppInfo(orgGUID, spaceGUID, "nodeapp-cpu")
		Expect(appName).ShouldNot(Equal(""), "Unable to determine nodeapp-cpu from space")
	})

	// To check existing aggregated cpu metrics do: cf asm APP_NAME cpu
	// See: https://www.ibm.com/docs/de/cloud-private/3.2.0?topic=SSBS6K_3.2.0/cloud_foundry/integrating/cfee_autoscaler.html
	Context("when scaling by cpu", func() {
		It("when cpu is greater than scaling out threshold", func() {
			By("should have a policy attached")
			policy := helpers.GetPolicy(cfg, appGUID)
			expectedPolicy := helpers.ScalingPolicy{InstanceMin: 1, InstanceMax: 2,
				ScalingRules: []*helpers.ScalingRule{
					{MetricType: "cpu", BreachDurationSeconds: helpers.TestBreachDurationSeconds,
						Threshold: 10, Operator: ">=", Adjustment: "+1", CoolDownSeconds: helpers.TestCoolDownSeconds},
					{MetricType: "cpu", BreachDurationSeconds: helpers.TestBreachDurationSeconds,
						Threshold: 5, Operator: "<", Adjustment: "-1", CoolDownSeconds: helpers.TestCoolDownSeconds},
				},
			}
			Expect(expectedPolicy).To(BeEquivalentTo(policy))

			By("should scale out to 2 instances")
			helpers.StartCPUUsage(cfg, appName, 50, 5)
			helpers.WaitForNInstancesRunning(appGUID, 2, 10*time.Minute)

			By("should scale in to 1 instance after cpu usage is reduced")
			//only hit the one instance that was asked to run hot.
			helpers.StopCPUUsage(cfg, appName, 0)

			helpers.WaitForNInstancesRunning(appGUID, 1, 10*time.Minute)
		})
	})
})
