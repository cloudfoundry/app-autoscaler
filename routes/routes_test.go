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
		testInstanceId string = "testInstanceId"
		testBindingId  string = "testBindingId"
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
				path, err := routes.PublicApiRoutes().Get(routes.PublicApiInfoRouteName).URLPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path.Path).To(Equal("/v1/info"))
			})
		})

		Context("PublicApiHealthRouteName", func() {
			It("should return the correct path", func() {
				path, err := routes.PublicApiRoutes().Get(routes.PublicApiHealthRouteName).URLPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path.Path).To(Equal("/health"))
			})
		})
	})

	Describe("PublicApiProtectedRoutes", func() {
		Context("PublicApiScalingHistoryRouteName", func() {

			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.PublicApiProtectedRoutes().Get(routes.PublicApiScalingHistoryRouteName).URLPath("appId", testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/scaling_histories"))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.PublicApiProtectedRoutes().Get(routes.PublicApiScalingHistoryRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.PublicApiProtectedRoutes().Get(routes.PublicApiScalingHistoryRouteName).URLPath()
					Expect(err).To(HaveOccurred())
				})
			})
		})
		Context("PublicApiMetricsHistoryRouteName", func() {

			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.PublicApiProtectedRoutes().Get(routes.PublicApiMetricsHistoryRouteName).URLPath("appId", testAppId, "metricType", testMetricType)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/metric_histories/" + testMetricType))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.PublicApiProtectedRoutes().Get(routes.PublicApiMetricsHistoryRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.PublicApiProtectedRoutes().Get(routes.PublicApiMetricsHistoryRouteName).URLPath()
					Expect(err).To(HaveOccurred())
				})
			})
		})
		Context("PublicApiAggregatedMetricsHistoryRouteName", func() {

			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.PublicApiProtectedRoutes().Get(routes.PublicApiAggregatedMetricsHistoryRouteName).URLPath("appId", testAppId, "metricType", testMetricType)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v1/apps/" + testAppId + "/aggregated_metric_histories/" + testMetricType))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.PublicApiProtectedRoutes().Get(routes.PublicApiAggregatedMetricsHistoryRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.PublicApiProtectedRoutes().Get(routes.PublicApiAggregatedMetricsHistoryRouteName).URLPath()
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})

	Describe("BrokerRoutes", func() {
		Context("BrokerCatalogRouteName", func() {
			It("should return the correct path", func() {
				path, err := routes.BrokerRoutes().Get(routes.BrokerCatalogRouteName).URLPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path.Path).To(Equal("/v2/catalog"))
			})
		})
		Context("BrokerCreateInstanceRouteName", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.BrokerRoutes().Get(routes.BrokerCreateInstanceRouteName).URLPath("instanceId", testInstanceId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v2/service_instances/" + testInstanceId))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.BrokerRoutes().Get(routes.BrokerCreateInstanceRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.BrokerRoutes().Get(routes.BrokerCreateInstanceRouteName).URLPath()
					Expect(err).To(HaveOccurred())
				})
			})
		})
		Context("BrokerDeleteInstanceRouteName", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.BrokerRoutes().Get(routes.BrokerDeleteInstanceRouteName).URLPath("instanceId", testInstanceId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v2/service_instances/" + testInstanceId))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.BrokerRoutes().Get(routes.BrokerDeleteInstanceRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.BrokerRoutes().Get(routes.BrokerDeleteInstanceRouteName).URLPath()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("BrokerCreateBindingRouteName", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.BrokerRoutes().Get(routes.BrokerCreateBindingRouteName).URLPath("instanceId", testInstanceId, "bindingId", testBindingId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v2/service_instances/" + testInstanceId + "/service_bindings/" + testBindingId))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.BrokerRoutes().Get(routes.BrokerCreateBindingRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.BrokerRoutes().Get(routes.BrokerCreateBindingRouteName).URLPath("instanceId", testInstanceId)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("BrokerDeleteBindingRouteName", func() {
			Context("when provide correct route variable", func() {
				It("should return the correct path", func() {
					path, err := routes.BrokerRoutes().Get(routes.BrokerDeleteBindingRouteName).URLPath("instanceId", testInstanceId, "bindingId", testBindingId)
					Expect(err).NotTo(HaveOccurred())
					Expect(path.Path).To(Equal("/v2/service_instances/" + testInstanceId + "/service_bindings/" + testBindingId))
				})
			})

			Context("when provide wrong route variable", func() {
				It("should return error", func() {
					_, err := routes.BrokerRoutes().Get(routes.BrokerDeleteBindingRouteName).URLPath("wrongVariable", testAppId)
					Expect(err).To(HaveOccurred())

				})
			})

			Context("when provide not enough route variable", func() {
				It("should return error", func() {
					_, err := routes.BrokerRoutes().Get(routes.BrokerDeleteBindingRouteName).URLPath("instanceId", testInstanceId)
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

	Describe("MetricServerRoutes", func() {
		Context("EnvelopeReportRouteName", func() {
			It("should return the correct path", func() {
				path, err := routes.MetricServerRoutes().Get(routes.EnvelopeReportRouteName).URLPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path.Path).To(Equal("/v1/envelopes"))
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
