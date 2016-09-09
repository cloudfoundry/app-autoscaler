package generator_test

import (
	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager/lagertest"
	"errors"
	"eventgenerator/aggregator/fakes"
	. "eventgenerator/generator"
	. "eventgenerator/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"net/http"
	"regexp"
	"time"
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
		triggerArray   []*Trigger = []*Trigger{&Trigger{
			AppId:            testAppId,
			MetricType:       testMetricType,
			BreachDuration:   300,
			CoolDownDuration: 300,
			Threshold:        500,
			Operator:         ">",
			Adjustment:       "1",
		}}
		appMetricUpper []*AppMetric = []*AppMetric{
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      600,
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      650,
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      620,
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
		appMetricLower []*AppMetric = []*AppMetric{
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      200,
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      150,
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
			&AppMetric{AppId: testAppId,
				MetricType: testMetricType,
				Value:      120,
				Unit:       "mb",
				Timestamp:  time.Now().UnixNano()},
		}
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("Evaluator-test")
		httpClient = cfhttp.NewClient()
		triggerChan = make(chan []*Trigger, 2)
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

			BeforeEach(func() {
				Expect(triggerChan).To(BeSent(triggerArray))
			})
			Context("retrieve appMatrics", func() {
				BeforeEach(func() {
					scalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
					database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
						return appMetricUpper, nil
					}
				})
				It("should retrieve appMetrics from database for each trigger", func() {
					Eventually(database.RetrieveAppMetricsCallCount).Should(Equal(1))
				})
			})

			Context("when the appMetrics breach the trigger", func() {
				BeforeEach(func() {
					database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
						return appMetricUpper, nil
					}
				})
				Context("should send trigger alarm to scaling engine", func() {
					Context("when send trigger alarm successfully", func() {
						BeforeEach(func() {
							scalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
						})
						It("when send trigger successfully", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						})
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

			})

			Context("when the appMetrics do not breach the trigger", func() {
				BeforeEach(func() {
					database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
						return appMetricLower, nil
					}
				})

				It("should not send trigger alarm", func() {
					Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
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

					Expect(triggerChan).To(BeSent(invalidTriggerArray))
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
			scalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			evaluator = NewEvaluator(logger, httpClient, scalingEngine.URL(), triggerChan, database)
			database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
				return appMetricUpper, nil
			}
			evaluator.Start()
			evaluator.Stop()
			time.Sleep(1 * time.Second)
		})

		AfterEach(func() {
			scalingEngine.Close()
		})

		It("should stop to send trigger alarm", func() {
			Expect(triggerChan).To(BeSent(triggerArray))
			Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
		})
	})
})
