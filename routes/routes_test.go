package routes_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Routes", func() {

	var (
		testAppId      = "testAppId"
		testScheduleId = "testScheduleId"
		testMetricType = "testMetricType"
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

	Describe("PublicApiRoutes", func() {
		Context("PublicApiInfoRouteName", func() {
			It("should return the correct path", func() {
				path, err := routes.ApiOpenRoutes().Get(routes.PublicApiInfoRouteName).URLPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path.Path).To(Equal("/v1/info"))
			})
		})

		Context("PublicApiHealthRouteName", func() {
			It("should return the correct path", func() {
				path, err := routes.ApiOpenRoutes().Get(routes.PublicApiHealthRouteName).URLPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path.Path).To(Equal("/health"))
			})
		})
	})

	Describe("ApiRoutes", func() {
		Context("PublicApiScalingHistoryRouteName", func() {

			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ApiRoutes().Get(routes.PublicApiScalingHistoryRouteName).URLPath("appId", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/scaling_histories"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ApiRoutes().Get(routes.PublicApiScalingHistoryRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ApiRoutes().Get(routes.PublicApiScalingHistoryRouteName).URLPath()
					Expect(err).To(HaveOccurred())
				})
			})
		})
		Context("PublicApiAggregatedMetricsHistoryRouteName", func() {

			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ApiRoutes().Get(routes.PublicApiAggregatedMetricsHistoryRouteName).URLPath("appId", testAppId, "metricType", testMetricType)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/aggregated_metric_histories/" + testMetricType))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ApiRoutes().Get(routes.PublicApiAggregatedMetricsHistoryRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ApiRoutes().Get(routes.PublicApiAggregatedMetricsHistoryRouteName).URLPath()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("PublicApiGetPolicyRouteName", func() {

			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ApiPolicyRoutes().Get(routes.PublicApiGetPolicyRouteName).URLPath("appId", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/policy"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ApiPolicyRoutes().Get(routes.PublicApiGetPolicyRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ApiPolicyRoutes().Get(routes.PublicApiGetPolicyRouteName).URLPath()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("PublicApiAttachPolicyRouteName", func() {

			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ApiPolicyRoutes().Get(routes.PublicApiAttachPolicyRouteName).URLPath("appId", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/policy"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ApiPolicyRoutes().Get(routes.PublicApiAttachPolicyRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ApiPolicyRoutes().Get(routes.PublicApiAttachPolicyRouteName).URLPath()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("PublicApiDetachPolicyRouteName", func() {

			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.ApiPolicyRoutes().Get(routes.PublicApiDetachPolicyRouteName).URLPath("appId", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/policy"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.ApiPolicyRoutes().Get(routes.PublicApiDetachPolicyRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.ApiPolicyRoutes().Get(routes.PublicApiDetachPolicyRouteName).URLPath()
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
					path, err := routes.ScalingEngineRoutes().Get(routes.GetScalingHistoriesRouteName).URLPath("guid", testAppId)
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

	Context("PostCustomMetricsRoute", func() {
		Context("when provide correct route variable", func() {
			It("should return the correct path", func() {
				path, err := routes.MetricsForwarderRoutes().Get(routes.PostCustomMetricsRouteName).URLPath("appid", testAppId)
				Expect(err).NotTo(HaveOccurred())
				Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/metrics"))
			})
		})
	})
})
