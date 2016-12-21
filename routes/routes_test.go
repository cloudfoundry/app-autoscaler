package routes_test

import (
	"autoscaler/routes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Routes", func() {

	var (
		testAppId      string = "testAppId"
		testScheduleId string = "testScheduleId"
	)
	Describe("MetricsCollectorRoutes", func() {
		Context("MemoryMetricRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.MetricsCollectorRoutes().Get(routes.MemoryMetricRoute).URLPath("appid", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/testAppId/metrics/memory"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.MetricsCollectorRoutes().Get(routes.MemoryMetricRoute).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.MetricsCollectorRoutes().Get(routes.MemoryMetricRoute).URLPath()
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("MemoryMetricHistoryRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.MetricsCollectorRoutes().Get(routes.MemoryMetricHistoryRoute).URLPath("appid", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/testAppId/metric_histories/memory"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.MetricsCollectorRoutes().Get(routes.MemoryMetricHistoryRoute).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.MetricsCollectorRoutes().Get(routes.MemoryMetricHistoryRoute).URLPath()
					Expect(err).To(HaveOccurred())

				})
			})
		})
	})

	Describe("ScalingEngineRoutes", func() {
		Context("ScaleRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ScalingEngineRoutes().Get(routes.ScaleRoute).URLPath("appid", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/testAppId/scale"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.ScaleRoute).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.ScaleRoute).URLPath()
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("HistoreisRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ScalingEngineRoutes().Get(routes.HistoreisRoute).URLPath("appid", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/testAppId/scaling_histories"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.HistoreisRoute).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.HistoreisRoute).URLPath()
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("UpdateActiveSchedulesRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ScalingEngineRoutes().Get(routes.UpdateActiveSchedulesRoute).URLPath("appid", testAppId, "scheduleid", testScheduleId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/testAppId/active_schedules/testScheduleId"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.UpdateActiveSchedulesRoute).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.UpdateActiveSchedulesRoute).URLPath("appid", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("DeleteActiveSchedulesRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ScalingEngineRoutes().Get(routes.DeleteActiveSchedulesRoute).URLPath("appid", testAppId, "scheduleid", testScheduleId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/testAppId/active_schedules/testScheduleId"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.DeleteActiveSchedulesRoute).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.DeleteActiveSchedulesRoute).URLPath("appid", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})
		})
	})
})
