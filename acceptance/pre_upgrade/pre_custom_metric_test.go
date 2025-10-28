package pre_upgrade_test

import (
	. "acceptance/helpers"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AutoScaler custom metrics policy", func() {
	var (
		appName string
		appGUID string
		policy  string
		err     error
	)

	BeforeEach(func() {
		policy = GenerateDynamicScaleOutAndInPolicy(1, 2, "test_metric", 500, 500)
		appName = CreateTestApp(cfg, "node-custom-metric", 1)
		appGUID, err = GetAppGuid(cfg, appName)
		Expect(err).NotTo(HaveOccurred())
		_ = CreatePolicy(cfg, appName, appGUID, policy)
		StartApp(appName, cfg.CfPushTimeoutDuration())
	})

	AfterEach(func() { DebugInfo(cfg, setup, appName) })

	Context("when scaling by custom metrics", func() {
		It("should scale out and scale in", func() {
			By("Scale out to 2 instances")
			scaleOut := func() (int, error) {
				SendMetric(cfg, appName, 550)
				return RunningInstances(appGUID, 5*time.Second)
			}
			Eventually(scaleOut, 5*time.Minute, 15*time.Second).Should(Equal(2))

			By("Scale in to 1 instances")
			scaleIn := func() (int, error) {
				SendMetric(cfg, appName, 100)
				return RunningInstances(appGUID, 5*time.Second)
			}
			Eventually(scaleIn, 5*time.Minute, 15*time.Second).Should(Equal(1))
		})
	})
})
