package app_test

import (
	. "acceptance/helpers"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Autoscaler lead times for scaling", func() {
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

	When("breach_duration_secs and cool_down_secs are set", func() {
		It("should do first scaling after breach_duration_secs have passed and second scaling after cool_down_secs have passed", func() {
			breachDuration := TestBreachDurationSeconds * time.Second
			coolDown := TestCoolDownSeconds * time.Second
			scalingTimewindow := 130 * time.Second // be friendly and allow some time for "internal autoscaler processes" (metric polling interval etc.) to take place before actual scaling happens

			sendMetricForScaleOutAndReturnNumInstancesFunc := sendMetricToAutoscaler(cfg, appToScaleGUID, appToScaleName, 510, true)
			sendMetricForScaleInAndReturnNumInstancesFunc := sendMetricToAutoscaler(cfg, appToScaleGUID, appToScaleName, 490, true)

			By("checking that no scaling out happens before breach_duration_secs have passed")
			Consistently(sendMetricForScaleOutAndReturnNumInstancesFunc).
				WithTimeout(breachDuration).
				WithPolling(time.Second).
				Should(Equal(1))

			By("checking that scale out happens in a certain time window after breach_duration_secs have passed")
			Eventually(sendMetricForScaleOutAndReturnNumInstancesFunc).
				WithTimeout(scalingTimewindow).
				WithPolling(time.Second).
				Should(Equal(2))

			By("checking that no scale in happens before cool_down_secs have passed")
			Consistently(sendMetricForScaleInAndReturnNumInstancesFunc).
				WithTimeout(coolDown).
				WithPolling(time.Second).
				Should(Equal(2))

			By("checking that scale in happens in a certain time window after cool_down_secs have passed")
			Eventually(sendMetricForScaleInAndReturnNumInstancesFunc).
				WithTimeout(scalingTimewindow).
				WithPolling(time.Second).
				Should(Equal(1))
		})
	})
})
