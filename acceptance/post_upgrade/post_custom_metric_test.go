package post_upgrade_test

import (
	"acceptance/helpers"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AutoScaler custom metrics policy", func() {
	var (
		appName string
		appGUID string
	)

	BeforeEach(func() {
		appName, appGUID = GetAppInfo(orgGUID, spaceGUID, "node-custom-metric")
		Expect(appName).ShouldNot(Equal(""), "Unable to determine node-custom-metric from space")
	})

	It("Scales by custom metrics post upgrade", func() {
		By("should still have the same policy attached")
		policy := helpers.GetPolicy(cfg, appGUID)
		expectedPolicy := helpers.ScalingPolicy{InstanceMin: 1, InstanceMax: 2,
			ScalingRules: []*helpers.ScalingRule{
				{MetricType: "test_metric", BreachDurationSeconds: 60, Threshold: 500, Operator: ">=", Adjustment: "+1", CoolDownSeconds: 60},
				{MetricType: "test_metric", BreachDurationSeconds: 60, Threshold: 500, Operator: "<", Adjustment: "-1", CoolDownSeconds: 60},
			},
		}
		Expect(expectedPolicy).To(BeEquivalentTo(policy))

		By("Should only have instance left over from the pre update test")
		Expect(helpers.RunningInstances(appGUID, 5*time.Second)).To(Equal(1))

		By("Scaling out to 2 instances")
		scaleOut := func() (int, error) {
			helpers.SendMetric(cfg, appName, 550)
			return helpers.RunningInstances(appGUID, 5*time.Second)
		}
		Eventually(scaleOut, 5*time.Minute, 15*time.Second).Should(Equal(2))

		By("Scaling in to 1 instance")
		scaleIn := func() (int, error) {
			helpers.SendMetric(cfg, appName, 100)
			return helpers.RunningInstances(appGUID, 5*time.Second)
		}
		Eventually(scaleIn, 5*time.Minute, 15*time.Second).Should(Equal(1))
	})

})
