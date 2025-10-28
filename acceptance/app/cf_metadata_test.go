package app_test

import (
	. "acceptance/helpers"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AutoScaler CF metadata support", func() {
	var (
		policy string
		err    error
	)
	BeforeEach(func() {
		policy = GenerateDynamicScaleOutAndInPolicy(1, 2, "test_metric", 500, 500)
		appToScaleName = CreateTestApp(cfg, "labeled-go_app", 1)
		appToScaleGUID, err = GetAppGuid(cfg, appToScaleName)
		Expect(err).NotTo(HaveOccurred())
		instanceName = CreatePolicy(cfg, appToScaleName, appToScaleGUID, policy)
		StartApp(appToScaleName, cfg.CfPushTimeoutDuration())
	})
	AfterEach(AppAfterEach)

	When("the label app-autoscaler.cloudfoundry.org/disable-autoscaling is set", func() {
		It("should not scale out", func() {
			By("Set the label app-autoscaler.cloudfoundry.org/disable-autoscaling to true")
			SetLabel(cfg, appToScaleGUID, "app-autoscaler.cloudfoundry.org/disable-autoscaling", "true")
			scaleOut := sendMetricToAutoscaler(cfg, appToScaleGUID, appToScaleName, 550, true)
			Consistently(scaleOut).
				WithTimeout(5 * time.Minute).
				WithPolling(15 * time.Second).
				Should(Equal(1))
		})
	})
})
