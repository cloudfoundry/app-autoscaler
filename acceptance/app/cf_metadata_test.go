package app_test

import (
	. "acceptance/helpers"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AutoScaler CF metadata support", func() {
	BeforeEach(setupCustomMetricTestApp)
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
