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
		testMetricType string = "testMetricType"
	)
	Describe("MetricsCollectorRoutes", func() {
		Context("GetMetricHistoriesRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.MetricsCollectorRoutes().Get(routes.GetMetricHistoriesRouteName).URLPath("appid", testAppId, "metrictype", testMetricType)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/metric_histories/" + testMetricType))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.MetricsCollectorRoutes().Get(routes.GetMetricHistoriesRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.MetricsCollectorRoutes().Get(routes.GetMetricHistoriesRouteName).URLPath("appid", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})
		})

	})

	Describe("EventGeneratorRoutes", func() {
		Context("GetAggregatedMetricHistoriesRouteName", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.EventGeneratorRoutes().Get(routes.GetAggregatedMetricHistoriesRouteName).URLPath("appid", testAppId, "metrictype", testMetricType)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/aggregated_metric_histories/" + testMetricType))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.EventGeneratorRoutes().Get(routes.GetAggregatedMetricHistoriesRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.EventGeneratorRoutes().Get(routes.GetAggregatedMetricHistoriesRouteName).URLPath("appid", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})
		})

	})

	Describe("ScalingEngineRoutes", func() {
		Context("ScaleRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ScalingEngineRoutes().Get(routes.ScaleRouteName).URLPath("appid", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/scale"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.ScaleRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.ScaleRouteName).URLPath()
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("GetScalingHistoriesRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ScalingEngineRoutes().Get(routes.GetScalingHistoriesRouteName).URLPath("appid", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/scaling_histories"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.GetScalingHistoriesRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.GetScalingHistoriesRouteName).URLPath()
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("SetActiveScheduleRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ScalingEngineRoutes().Get(routes.SetActiveScheduleRouteName).URLPath("appid", testAppId, "scheduleid", testScheduleId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/active_schedules/" + testScheduleId))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.SetActiveScheduleRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.SetActiveScheduleRouteName).URLPath("appid", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("DeleteActiveScheduleRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ScalingEngineRoutes().Get(routes.DeleteActiveScheduleRouteName).URLPath("appid", testAppId, "scheduleid", testScheduleId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/active_schedules/" + testScheduleId))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.DeleteActiveScheduleRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.DeleteActiveScheduleRouteName).URLPath("appid", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("GetActiveSchedulesRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ScalingEngineRoutes().Get(routes.GetActiveSchedulesRouteName).URLPath("appid", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/active_schedules"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.GetActiveSchedulesRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ScalingEngineRoutes().Get(routes.GetActiveSchedulesRouteName).URLPath()
					Expect(err).To(HaveOccurred())

				})
			})
		})
	})
})
