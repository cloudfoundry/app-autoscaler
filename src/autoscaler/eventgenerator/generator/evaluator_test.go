package generator_test

import (
	"autoscaler/eventgenerator/aggregator/fakes"
	. "autoscaler/eventgenerator/generator"
	. "autoscaler/eventgenerator/model"
	"errors"
	"net/http"
	"regexp"
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
		triggerChan    chan []*Trigger
		database       *fakes.FakeAppMetricDB
		scalingEngine  *ghttp.Server
		evaluator      *Evaluator
		testAppId      string     = "testAppId"
		testMetricType string     = "MemoryUsage"
		regPath                   = regexp.MustCompile(`^/v1/apps/.*/scale$`)
		triggerArrayGT []*Trigger = []*Trigger{&Trigger{
			AppId:            testAppId,
			MetricType:       testMetricType,
			BreachDuration:   300,
			CoolDownDuration: 300,
			Threshold:        500,
			Operator:         ">",
			Adjustment:       "1",
		}}
		triggerArrayGE []*Trigger = []*Trigger{&Trigger{
			AppId:            testAppId,
			MetricType:       testMetricType,
			BreachDuration:   300,
			CoolDownDuration: 300,
			Threshold:        500,
			Operator:         ">=",
			Adjustment:       "1",
		}}
		triggerArrayLT []*Trigger = []*Trigger{&Trigger{
			AppId:            testAppId,
			MetricType:       testMetricType,
			BreachDuration:   300,
			CoolDownDuration: 300,
			Threshold:        500,
			Operator:         "<",
			Adjustment:       "1",
		}}
		triggerArrayLE []*Trigger = []*Trigger{&Trigger{
			AppId:            testAppId,
			MetricType:       testMetricType,
			BreachDuration:   300,
			CoolDownDuration: 300,
			Threshold:        500,
			Operator:         "<=",
			Adjustment:       "1",
		}}
		//test appmetric for >
		appMetricGTUpper []*AppMetric = []*AppMetric{
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(650),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(620),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricGTLower []*AppMetric = []*AppMetric{
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(200),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(150),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(120),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}

		//test appmetric for >=
		appMetricGEUpper []*AppMetric = []*AppMetric{
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricGELower []*AppMetric = []*AppMetric{
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(200),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(150),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(120),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}

		//test appmetric for <
		appMetricLTUpper []*AppMetric = []*AppMetric{
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricLTLower []*AppMetric = []*AppMetric{
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(200),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(150),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(120),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}

		//test appmetric for <=
		appMetricLEUpper []*AppMetric = []*AppMetric{
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(600),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricLELower []*AppMetric = []*AppMetric{
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      GetInt64Pointer(500),
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("Evaluator-test")
		httpClient = cfhttp.NewClient()
		triggerChan = make(chan []*Trigger, 1)
		database = &fakes.FakeAppMetricDB{}
		scalingEngine = ghttp.NewServer()

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
					scalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
					database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
					scalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
				})
				Context(">", func() {
					BeforeEach(func() {
						Expect(triggerChan).To(BeSent(triggerArrayGT))
					})
					Context("when the appMetrics breach the trigger", func() {
						BeforeEach(func() {
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
								return []*AppMetric{}, nil
							}
						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain nil-value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricGTUpper, &AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      nil,
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
								return []*AppMetric{}, nil
							}
						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain nil-value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricGEUpper, &AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      nil,
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
								return []*AppMetric{}, nil
							}
						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain nil-value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricLTLower, &AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      nil,
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
								return []*AppMetric{}, nil
							}
						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain nil-value elements", func() {
						BeforeEach(func() {
							appMetricNilValue := append(appMetricLELower, &AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      nil,
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
					database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
						scalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWithJSONEncoded(http.StatusBadRequest, "error"))
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
						scalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWithJSONEncoded(http.StatusBadRequest, errorStr))
					})

					It("should log the error", func() {
						Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("failed to read body from scaling engine's response")))
					})
				})
			})

			Context("when retrieve appMetrics from database failed", func() {
				BeforeEach(func() {
					database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
						return nil, errors.New("error when retrieve appMetrics from database")
					}
				})

				It("should not send trigger alarm", func() {
					Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
				})
			})

			Context("when there are invalid operators in trigger", func() {
				BeforeEach(func() {
					invalidTriggerArray := []*Trigger{&Trigger{
						AppId:            testAppId,
						MetricType:       testMetricType,
						BreachDuration:   300,
						CoolDownDuration: 300,
						Threshold:        500,
						Operator:         "invalid_operator",
						Adjustment:       "1",
					}}
					triggerChan = make(chan []*Trigger, 1)
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
			database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
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
