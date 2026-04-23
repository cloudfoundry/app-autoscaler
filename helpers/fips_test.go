package helpers_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	dto "github.com/prometheus/client_model/go"
)

var _ = Describe("FIPS Mode", func() {
	Describe("AssertFIPSMode", func() {
		Context("when fipsEnabled is false", func() {
			It("sets the gauge to 0 and returns without exiting", func() {
				helpers.AssertFIPSMode(false)

				var metric dto.Metric
				Expect(helpers.FipsEnabledGauge.Write(&metric)).To(Succeed())
				Expect(metric.GetGauge().GetValue()).To(Equal(float64(0)))
			})
		})
	})

	Describe("FipsEnabledGauge", func() {
		It("has the correct description", func() {
			Expect(helpers.FipsEnabledGauge.Desc().String()).To(ContainSubstring("autoscaler_fips_enabled"))
		})
	})
})
