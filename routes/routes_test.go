package routes_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Routes", func() {
	var (
		autoscalerRouter *routes.Router
		router           *mux.Router
		testAppId        = "testAppId"
		testScheduleId   = "testScheduleId"
		testMetricType   = "testMetricType"
	)

	BeforeEach(func() {
		autoscalerRouter = routes.NewRouter()
	})

	JustBeforeEach(func() {
		router = autoscalerRouter.GetRouter()
	})

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
		JustBeforeEach(func() {
			autoscalerRouter.CreateApiPublicSubrouter()
		})

		Context("PublicApiInfoRouteName", func() {
			It("should return the correct path", func() {
				path, err := router.Get(routes.PublicApiInfoRouteName).URLPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path.Path).To(Equal("/v1/info"))
			})
		})

		XContext("PublicApiHealthRouteName", func() {
			It("should return the correct path", func() {
				path, err := router.Get(routes.PublicApiHealthRouteName).URLPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path.Path).To(Equal("/health"))
			})
		})
	})

	Describe("ApiRoutes", func() {
		JustBeforeEach(func() {
			autoscalerRouter.CreateApiSubrouter()
		})
		Context("PublicApiScalingHistoryRouteName", func() {

			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {

					path, err := router.Get(routes.PublicApiScalingHistoryRouteName).URLPath("appId", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/scaling_histories"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.PublicApiScalingHistoryRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.PublicApiScalingHistoryRouteName).URLPath()
					Expect(err).To(HaveOccurred())
				})
			})
		})
		Context("PublicApiAggregatedMetricsHistoryRouteName", func() {

			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := router.Get(routes.PublicApiAggregatedMetricsHistoryRouteName).URLPath("appId", testAppId, "metricType", testMetricType)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/aggregated_metric_histories/" + testMetricType))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.PublicApiAggregatedMetricsHistoryRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.PublicApiAggregatedMetricsHistoryRouteName).URLPath()
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

	Describe("CreateEventGeneratorRoutes", func() {
		JustBeforeEach(func() {
			autoscalerRouter.CreateEventGeneratorSubrouter()
		})

		Context("GetAggregatedMetricHistoriesRouteName", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := router.Get(routes.GetAggregatedMetricHistoriesRouteName).URLPath("appid", testAppId, "metrictype", testMetricType)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/aggregated_metric_histories/" + testMetricType))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.GetAggregatedMetricHistoriesRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.GetAggregatedMetricHistoriesRouteName).URLPath("appid", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})
		})

	})

	Describe("CreateScalingEngineRoutes", func() {
		JustBeforeEach(func() {
			autoscalerRouter.CreateScalingEngineRoutes()
		})
		Context("ScaleRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := router.Get(routes.ScaleRouteName).URLPath("appid", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/scale"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.ScaleRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.ScaleRouteName).URLPath()
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("GetScalingHistoriesRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := router.Get(routes.GetScalingHistoriesRouteName).URLPath("guid", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/scaling_histories"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.GetScalingHistoriesRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.GetScalingHistoriesRouteName).URLPath()
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("SetActiveScheduleRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := router.Get(routes.SetActiveScheduleRouteName).URLPath("appid", testAppId, "scheduleid", testScheduleId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/active_schedules/" + testScheduleId))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.SetActiveScheduleRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.SetActiveScheduleRouteName).URLPath("appid", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("DeleteActiveScheduleRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := router.Get(routes.DeleteActiveScheduleRouteName).URLPath("appid", testAppId, "scheduleid", testScheduleId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/active_schedules/" + testScheduleId))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.DeleteActiveScheduleRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.DeleteActiveScheduleRouteName).URLPath("appid", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})
		})

		Context("GetActiveSchedulesRoute", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := router.Get(routes.GetActiveSchedulesRouteName).URLPath("appid", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/active_schedules"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.GetActiveSchedulesRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := router.Get(routes.GetActiveSchedulesRouteName).URLPath()
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
