package generator_test

import (
	"autoscaler/eventgenerator/aggregator/fakes"
	. "autoscaler/eventgenerator/generator"
	"autoscaler/models"
	"autoscaler/routes"
	"errors"
	"net/http"
	"time"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Evaluator", func() {
	var (
		logger         *lagertest.TestLogger
		httpClient     *http.Client
		triggerChan    chan []*models.Trigger
		database       *fakes.FakeAppMetricDB
		scalingEngine  *ghttp.Server
		evaluator      *Evaluator
		testAppId      string = "testAppId"
		testMetricType string = models.MetricNameMemory
		urlPath        string
		triggerArrayGT []*models.Trigger = []*models.Trigger{{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: 300,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              ">",
			Adjustment:            "+1",
		}}
		triggerArrayGE []*models.Trigger = []*models.Trigger{{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: 300,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              ">=",
			Adjustment:            "+1",
		}}
		triggerArrayLT []*models.Trigger = []*models.Trigger{{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: 300,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              "<",
			Adjustment:            "-1",
		}}
		triggerArrayLE []*models.Trigger = []*models.Trigger{{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: 300,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              "<=",
			Adjustment:            "-1",
		}}

		firstTrigger = models.Trigger{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: 300,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              ">=",
			Adjustment:            "+1",
		}

		secondTrigger = models.Trigger{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: 300,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              "<=",
			Adjustment:            "-1",
		}
		triggerArrayMultipleTriggers []*models.Trigger = []*models.Trigger{&firstTrigger, &secondTrigger}

		//test appmetric for >
		appMetricGTBreach []*models.AppMetric = []*models.AppMetric{
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "600",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "650",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "620",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricGTNotBreach []*models.AppMetric = []*models.AppMetric{
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "200",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "150",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "600",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
		}

		//test appmetric for >=
		appMetricGEBreach []*models.AppMetric = []*models.AppMetric{
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "600",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "500",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "700",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricGENotBreach []*models.AppMetric = []*models.AppMetric{
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "200",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "500",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "120",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
		}

		//test appmetric for <
		appMetricLTNotBreach []*models.AppMetric = []*models.AppMetric{
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "600",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "300",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "200",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricLTBreach []*models.AppMetric = []*models.AppMetric{
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "200",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "150",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "320",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
		}

		//test appmetric for <=
		appMetricLENotBreach []*models.AppMetric = []*models.AppMetric{
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "600",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "500",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "300",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricLEBreach []*models.AppMetric = []*models.AppMetric{
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "500",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "300",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "200",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
		}

		appMetricMultipleTriggerAllBreach []*models.AppMetric = []*models.AppMetric{
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "500",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "500",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
			{AppId: testAppId,
				MetricType: testMetricType,
				Value:      "500",
				Unit:       models.UnitMegaBytes,
				Timestamp:  time.Now().UnixNano()},
		}
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("Evaluator-test")
		httpClient = cfhttp.NewClient()
		triggerChan = make(chan []*models.Trigger, 1)
		database = &fakes.FakeAppMetricDB{}
		scalingEngine = nil

		path, err := routes.ScalingEngineRoutes().Get(routes.ScaleRoute).URLPath("appid", testAppId)
		Expect(err).NotTo(HaveOccurred())
		urlPath = path.Path

	})

	Context("Start", func() {
		JustBeforeEach(func() {
			evaluator = NewEvaluator(logger, httpClient, scalingEngine.URL(), triggerChan, database)
			evaluator.Start()
		})

		BeforeEach(func() {
			scalingEngine = ghttp.NewServer()
		})

		AfterEach(func() {
			evaluator.Stop()
			scalingEngine.Close()
		})

		Context("when evaluator is started", func() {

			Context("retrieve appMatrics", func() {
				BeforeEach(func() {
					scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWith(http.StatusOK, "successful"))
					database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
						return appMetricGTBreach, nil
					}
					Expect(triggerChan).To(BeSent(triggerArrayGT))
				})

				It("should retrieve appMetrics from database for each trigger", func() {
					Eventually(database.RetrieveAppMetricsCallCount).Should(Equal(1))
				})
			})

			Context("operators", func() {
				BeforeEach(func() {
					scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWith(http.StatusOK, "successful"))
				})
				Context(">", func() {
					BeforeEach(func() {
						Expect(triggerChan).To(BeSent(triggerArrayGT))
					})
					Context("when the appMetrics breach the trigger", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricGTBreach, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						})
					})
					Context("when the appMetrics do not breach the trigger", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricGTNotBreach, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine")))
						})
					})
					Context("when appMetrics is empty", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return []*models.AppMetric{}, nil
							}
						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain  empty value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricGTBreach, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      "",
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricNilValue, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine because there is empty value metric")))
						})
					})
				})
				Context(">=", func() {
					BeforeEach(func() {
						Expect(triggerChan).To(BeSent(triggerArrayGE))
					})
					Context("when the appMetrics breach the trigger", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricGEBreach, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						})
					})
					Context("when the appMetrics do not breach the trigger", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricGENotBreach, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine")))
						})
					})
					Context("when appMetrics is empty", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return []*models.AppMetric{}, nil
							}
						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain empty value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricGEBreach, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      "",
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricNilValue, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine because there is empty value metric")))
						})
					})
				})
				Context("<", func() {
					BeforeEach(func() {
						Expect(triggerChan).To(BeSent(triggerArrayLT))
					})
					Context("when the appMetrics breach the trigger", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricLTBreach, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						})
					})
					Context("when the appMetrics do not breach the trigger", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricLTNotBreach, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine")))
						})
					})
					Context("when appMetrics is empty", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return []*models.AppMetric{}, nil
							}
						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain empty value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricLTBreach, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      "",
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricNilValue, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine because there is empty value metric")))
						})
					})
				})
				Context("<=", func() {
					BeforeEach(func() {
						Expect(triggerChan).To(BeSent(triggerArrayLE))
					})
					Context("when the appMetrics breach the trigger", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricLEBreach, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						})
					})
					Context("when the appMetrics do not breach the trigger", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricLENotBreach, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine")))
						})
					})
					Context("when appMetrics is empty", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return []*models.AppMetric{}, nil
							}
						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain empty value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricLEBreach, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      "",
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricNilValue, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine because there is empty value metric")))
						})
					})
				})
			})

			Context("multiple triggers", func() {
				BeforeEach(func() {
					Expect(triggerChan).To(BeSent(triggerArrayMultipleTriggers))
				})
				Context("when only the first trigger breaches", func() {
					BeforeEach(func() {
						scalingEngine.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("POST", urlPath),
								ghttp.VerifyJSONRepresenting(firstTrigger),
								ghttp.RespondWith(http.StatusOK, ""),
							),
						)

						database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
							return appMetricGEBreach, nil
						}
					})
					It("should send alarm of first trigger to scaling engine", func() {
						Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
					})
				})

				Context("when only second tigger breaches", func() {
					BeforeEach(func() {
						scalingEngine.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("POST", urlPath),
								ghttp.VerifyJSONRepresenting(secondTrigger),
								ghttp.RespondWith(http.StatusOK, ""),
							),
						)

						database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
							return appMetricLEBreach, nil
						}
					})
					It("should send alarm  of second trigger to scaling engine", func() {
						Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
					})
				})

				Context("when both tiggers breach", func() {
					BeforeEach(func() {
						scalingEngine.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("POST", urlPath),
								ghttp.VerifyJSONRepresenting(firstTrigger),
								ghttp.RespondWith(http.StatusOK, ""),
							),
						)
						database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
							return appMetricMultipleTriggerAllBreach, nil
						}
					})
					It("should send alarm of first trigger to scaling engine", func() {
						Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
					})
				})

			})

			Context("send trigger failed", func() {
				BeforeEach(func() {
					database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
						return appMetricGTBreach, nil
					}
				})
				Context("when the scaling engine returns error", func() {
					BeforeEach(func() {
						scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWithJSONEncoded(http.StatusBadRequest, "error"))
						Expect(triggerChan).To(BeSent(triggerArrayGT))
					})

					It("should log the error", func() {
						Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("scaling engine error,failed to send trigger alarm")))
					})
				})

				PContext("when the scaling engine's response is too long", func() {
					BeforeEach(func() {
						tmp := ""
						errorStr := ""
						for i := 0; i < 9999; i++ {
							tmp = tmp + "error-error-error-error-error-error-error-error-error-error-error-error-error-error-error-error-error-error-error-error"
						}
						for i := 0; i < 999; i++ {
							errorStr = errorStr + tmp
						}
						scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWithJSONEncoded(http.StatusBadRequest, errorStr))
						Expect(triggerChan).To(BeSent(triggerArrayGT))
					})

					It("should log the error", func() {
						Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("failed to read body from scaling engine's response")))
					})
				})
			})

			Context("when retrieve appMetrics from database failed", func() {
				BeforeEach(func() {
					database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
						return nil, errors.New("error when retrieve appMetrics from database")
					}
				})

				It("should not send trigger alarm", func() {
					Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
				})
			})

			Context("when there are invalid operators in trigger", func() {
				BeforeEach(func() {
					invalidTriggerArray := []*models.Trigger{{
						AppId:                 testAppId,
						MetricType:            testMetricType,
						BreachDurationSeconds: 300,
						CoolDownSeconds:       300,
						Threshold:             500,
						Operator:              "invalid_operator",
						Adjustment:            "1",
					}}
					triggerChan = make(chan []*models.Trigger, 1)
					Eventually(triggerChan).Should(BeSent(invalidTriggerArray))
				})

				It("should log the error", func() {
					Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("operator is invalid")))
				})
			})

		})
	})

	Context("Stop", func() {
		BeforeEach(func() {
			scalingEngine = ghttp.NewServer()
			database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
				return nil, errors.New("no alarm")
			}
			evaluator = NewEvaluator(logger, httpClient, scalingEngine.URL(), triggerChan, database)
			evaluator.Start()
			Expect(triggerChan).To(BeSent(triggerArrayGT))
			Eventually(database.RetrieveAppMetricsCallCount).Should(Equal(1))

			evaluator.Stop()
		})

		It("should stop to send trigger alarm", func() {
			Eventually(triggerChan).ShouldNot(BeSent(triggerArrayGT))
		})
	})

	Context("Scaling Engine is not reachable", func() {
		BeforeEach(func() {
			scalingEngine = ghttp.NewUnstartedServer()
			database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
				return appMetricGTBreach, nil
			}
			evaluator = NewEvaluator(logger, httpClient, scalingEngine.URL(), triggerChan, database)
			evaluator.Start()
			Expect(triggerChan).To(BeSent(triggerArrayGT))
		})

		AfterEach(func() {
			evaluator.Stop()
			scalingEngine.Close()
		})

		It("should log the error", func() {
			Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("http reqeust error,failed to send trigger alarm")))
		})
	})
})
