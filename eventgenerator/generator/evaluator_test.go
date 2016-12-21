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
		testMetricType string = "MemoryUsage"
		urlPath        string
		triggerArrayGT []*models.Trigger = []*models.Trigger{&models.Trigger{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: 300,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              ">",
			Adjustment:            "1",
		}}
		triggerArrayGE []*models.Trigger = []*models.Trigger{&models.Trigger{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: 300,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              ">=",
			Adjustment:            "1",
		}}
		triggerArrayLT []*models.Trigger = []*models.Trigger{&models.Trigger{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: 300,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              "<",
			Adjustment:            "1",
		}}
		triggerArrayLE []*models.Trigger = []*models.Trigger{&models.Trigger{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: 300,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              "<=",
			Adjustment:            "1",
		}}
		//test appmetric for >
		appMetricGTUpper []*models.AppMetric = []*models.AppMetric{
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(650),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(620),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricGTLower []*models.AppMetric = []*models.AppMetric{
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(200),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(150),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(120),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}

		//test appmetric for >=
		appMetricGEUpper []*models.AppMetric = []*models.AppMetric{
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricGELower []*models.AppMetric = []*models.AppMetric{
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(200),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(150),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(120),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}

		//test appmetric for <
		appMetricLTUpper []*models.AppMetric = []*models.AppMetric{
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricLTLower []*models.AppMetric = []*models.AppMetric{
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(200),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(150),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(120),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}

		//test appmetric for <=
		appMetricLEUpper []*models.AppMetric = []*models.AppMetric{
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricLELower []*models.AppMetric = []*models.AppMetric{
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&models.AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("Evaluator-test")
		httpClient = cfhttp.NewClient()
		triggerChan = make(chan []*models.Trigger, 1)
		database = &fakes.FakeAppMetricDB{}
		scalingEngine = ghttp.NewServer()

		path, err := routes.ScalingEngineRoutes().Get(routes.ScaleRoute).URLPath("appid", testAppId)
		Expect(err).NotTo(HaveOccurred())
		urlPath = path.Path

	})

	Context("Start", func() {
		JustBeforeEach(func() {
			evaluator = NewEvaluator(logger, httpClient, scalingEngine.URL(), triggerChan, database)
			evaluator.Start()
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
						return appMetricGTUpper, nil
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
								return appMetricGTUpper, nil
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
								return appMetricGTLower, nil
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
					Context("when the appMetrics contain nil-value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricGTUpper, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      nil,
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricNilValue, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine because there is nil-value metric")))
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
								return appMetricGEUpper, nil
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
								return appMetricGELower, nil
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
					Context("when the appMetrics contain nil-value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricGEUpper, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      nil,
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricNilValue, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine because there is nil-value metric")))
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
								return appMetricLTLower, nil
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
								return appMetricLTUpper, nil
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
					Context("when the appMetrics contain nil-value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricLTLower, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      nil,
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricNilValue, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine because there is nil-value metric")))
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
								return appMetricLELower, nil
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
								return appMetricLEUpper, nil
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
					Context("when the appMetrics contain nil-value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricLELower, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      nil,
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
								return appMetricNilValue, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine because there is nil-value metric")))
						})
					})
				})
			})
			Context("send trigger failed", func() {
				BeforeEach(func() {
					database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*models.AppMetric, error) {
						return appMetricGTUpper, nil
					}
					Expect(triggerChan).To(BeSent(triggerArrayGT))
				})
				Context("when the send request encounters error", func() {
					JustBeforeEach(func() {
						scalingEngine.Close()
					})

					It("should log the error", func() {
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("http reqeust error,failed to send trigger alarm")))
					})
				})

				Context("when the scaling engine returns error", func() {
					BeforeEach(func() {
						scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWithJSONEncoded(http.StatusBadRequest, "error"))
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
					invalidTriggerArray := []*models.Trigger{&models.Trigger{
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
})
