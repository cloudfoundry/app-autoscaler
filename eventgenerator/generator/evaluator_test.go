package generator_test

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/generator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/cfhttp/v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"github.com/cenkalti/backoff/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	circuit "github.com/rubyist/circuitbreaker"
)

var _ = Describe("Evaluator", func() {
	var (
		logger             *lagertest.TestLogger
		httpClient         *http.Client
		triggerChan        chan []*models.Trigger
		scalingEngine      *ghttp.Server
		evaluator          *Evaluator
		testAppId          = "testAppId"
		testMetricType     = "testMetricType"
		testMetricUnit     = "testMetricUnit"
		urlPath            string
		breachDurationSecs = 30
		queryAppMetrics    aggregator.QueryAppMetricsFunc
		getBreaker         func(string) *circuit.Breaker
		setCoolDownExpired func(string, int64)
		cbEventChan        <-chan circuit.BreakerEvent
		cooldownExpired    map[string]int64
		fakeTime           = time.Now()
		lock               = &sync.Mutex{}
		scalingResult      *models.AppScalingResult
		triggerArrayGT     = []*models.Trigger{{
			AppId:           testAppId,
			MetricType:      testMetricType,
			CoolDownSeconds: 300,
			Threshold:       500,
			Operator:        ">",
			Adjustment:      "+1",
		}}
		triggerArrayGE = []*models.Trigger{{
			AppId:           testAppId,
			MetricType:      testMetricType,
			CoolDownSeconds: 300,
			Threshold:       500,
			Operator:        ">=",
			Adjustment:      "+1",
		}}
		triggerArrayLT = []*models.Trigger{{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: breachDurationSecs,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              "<",
			Adjustment:            "-1",
		}}
		triggerArrayLE = []*models.Trigger{{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			BreachDurationSeconds: breachDurationSecs,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              "<=",
			Adjustment:            "-1",
		}}

		firstTrigger = models.Trigger{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			MetricUnit:            testMetricUnit,
			BreachDurationSeconds: breachDurationSecs,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              ">=",
			Adjustment:            "+1",
		}

		secondTrigger = models.Trigger{
			AppId:                 testAppId,
			MetricType:            testMetricType,
			MetricUnit:            testMetricUnit,
			BreachDurationSeconds: breachDurationSecs,
			CoolDownSeconds:       300,
			Threshold:             500,
			Operator:              "<=",
			Adjustment:            "-1",
		}
		triggerArrayMultipleTriggers = []*models.Trigger{&firstTrigger, &secondTrigger}
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("Evaluator-test")
		httpClient = cfhttp.NewClient()
		triggerChan = make(chan []*models.Trigger, 1)

		r := routes.NewRouter()
		path, err := r.CreateScalingEngineRoutes().Get(routes.ScaleRouteName).URLPath("appid", testAppId)
		Expect(err).NotTo(HaveOccurred())
		urlPath = path.Path

		getBreaker = func(appID string) *circuit.Breaker {
			return nil
		}
		queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
			return nil, nil
		}

		scalingResult = &models.AppScalingResult{
			AppId:             testAppId,
			Adjustment:        1,
			Status:            models.ScalingStatusSucceeded,
			CooldownExpiredAt: fakeTime.Add(time.Duration(300) * time.Second).UnixNano(),
		}

		cooldownExpired = map[string]int64{}
		setCoolDownExpired = func(appId string, expiredAt int64) {
			lock.Lock()
			defer lock.Unlock()
			cooldownExpired[appId] = expiredAt
		}

	})
	AfterEach(func() {
		close(triggerChan)
	})

	Context("Start", func() {
		JustBeforeEach(func() {
			evaluator = NewEvaluator(logger, httpClient, scalingEngine.URL(), triggerChan, breachDurationSecs, queryAppMetrics, getBreaker, setCoolDownExpired)
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

			Context("when the appMetrics are not enough", func() {
				BeforeEach(func() {
					scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWith(http.StatusOK, "successful"))
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{600, 650, 620}, breachDurationSecs, false)
					queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
						return appMetrics, nil
					}
				})
				It("should not send trigger alarm to scaling engine", func() {
					Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
					Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("the appmetrics are not enough for evaluation")))
				})
			})
			Context("operators", func() {
				BeforeEach(func() {
					scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWithJSONEncoded(http.StatusOK, &scalingResult))
				})
				Context(">", func() {
					BeforeEach(func() {
						Expect(triggerChan).To(BeSent(triggerArrayGT))
					})
					Context("when the appMetrics breach the trigger", func() {
						BeforeEach(func() {
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{600, 650, 620}, breachDurationSecs, true)
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
							scalingEngine.RouteToHandler("POST", urlPath,
								ghttp.CombineHandlers(
									ghttp.VerifyJSONRepresenting(models.Trigger{
										AppId:                 testAppId,
										MetricType:            testMetricType,
										MetricUnit:            testMetricUnit,
										BreachDurationSeconds: breachDurationSecs,
										CoolDownSeconds:       300,
										Threshold:             500,
										Operator:              ">",
										Adjustment:            "+1",
									}),
									ghttp.RespondWithJSONEncoded(http.StatusOK, &scalingResult)),
							)
						})
						It("should send trigger alarm to scaling engine", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						})

					})
					Context("when the appMetrics do not breach the trigger", func() {
						BeforeEach(func() {
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{200, 150, 600}, breachDurationSecs, true)
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine")))
						})

					})
					Context("when appMetrics is empty", func() {
						BeforeEach(func() {
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return []*models.AppMetric{}, nil
							}
						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain empty value elements", func() {
						BeforeEach(func() {
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{600, 650, 620}, breachDurationSecs, true)
							appMetrics = append(appMetrics, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      "",
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
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
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{600, 500, 500}, breachDurationSecs, true)
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						})

					})
					Context("when the appMetrics do not breach the trigger", func() {
						BeforeEach(func() {
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{200, 150, 600}, breachDurationSecs, true)
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine")))
						})

					})
					Context("when appMetrics is empty", func() {
						BeforeEach(func() {
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return []*models.AppMetric{}, nil
							}

						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain  empty value elements", func() {
						BeforeEach(func() {
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{600, 500, 500}, breachDurationSecs, true)
							appMetrics = append(appMetrics, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      "",
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
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
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{200, 300, 400}, breachDurationSecs, true)
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
						})
						It("should send trigger alarm to scaling engine", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						})

					})
					Context("when the appMetrics do not breach the trigger", func() {
						BeforeEach(func() {
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{500, 550, 600}, breachDurationSecs, true)
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine")))
						})

					})
					Context("when appMetrics is empty", func() {
						BeforeEach(func() {
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return []*models.AppMetric{}, nil
							}

						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain  empty value elements", func() {
						BeforeEach(func() {
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{200, 300, 400}, breachDurationSecs, true)
							appMetrics = append(appMetrics, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      "",
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
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
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{200, 500, 500}, breachDurationSecs, true)
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}

						})
						It("should send trigger alarm to scaling engine", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						})

					})
					Context("when the appMetrics do not breach the trigger", func() {
						BeforeEach(func() {
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{500, 550, 600}, breachDurationSecs, true)
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("should not send trigger alarm to scaling engine")))
						})

					})
					Context("when appMetrics is empty", func() {
						BeforeEach(func() {
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return []*models.AppMetric{}, nil
							}

						})

						It("should not send trigger alarm", func() {
							Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(0))
						})
					})
					Context("when the appMetrics contain  empty value elements", func() {
						BeforeEach(func() {
							appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{200, 500, 500}, breachDurationSecs, true)
							appMetrics = append(appMetrics, &models.AppMetric{AppId: testAppId,
								MetricType: testMetricType,
								Value:      "",
								Unit:       "",
								Timestamp:  time.Now().UnixNano()})
							queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
								return appMetrics, nil
							}
						})
						It("should not send trigger alarm to scaling engine", func() {
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
								ghttp.RespondWithJSONEncoded(http.StatusOK, &scalingResult),
							),
						)
						appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{500, 550, 600}, breachDurationSecs, true)
						queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
							return appMetrics, nil
						}
					})
					It("should send alarm of first trigger to scaling engine", func() {
						Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("successfully-send-trigger-alarm with trigger")))

					})
				})

				Context("when only second tigger breaches", func() {
					BeforeEach(func() {
						scalingEngine.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("POST", urlPath),
								ghttp.VerifyJSONRepresenting(secondTrigger),
								ghttp.RespondWithJSONEncoded(http.StatusOK, &scalingResult),
							),
						)
						appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{300, 400, 500}, breachDurationSecs, true)
						queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
							return appMetrics, nil
						}
					})
					It("should send alarm  of second trigger to scaling engine", func() {
						Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("successfully-send-trigger-alarm with trigger")))

					})
				})

				Context("when both tiggers breach", func() {
					BeforeEach(func() {
						scalingEngine.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("POST", urlPath),
								ghttp.VerifyJSONRepresenting(firstTrigger),
								ghttp.RespondWithJSONEncoded(http.StatusOK, &scalingResult),
							),
						)
						appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{500, 500, 500}, breachDurationSecs, true)
						queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
							return appMetrics, nil
						}
					})
					It("should send alarm of first trigger to scaling engine", func() {
						Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("successfully-send-trigger-alarm with trigger")))
					})
				})

			})

			Context("sending trigger ", func() {
				BeforeEach(func() {
					appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{600, 650, 620}, breachDurationSecs, true)
					queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
						return appMetrics, nil
					}
					Expect(triggerChan).To(BeSent(triggerArrayGT))
				})

				Context("when the scaling engine returns 200 with different scalingResults", func() {
					Context("when cooldownExpiredAt is set to a valid timestamp in scalingResult ", func() {
						BeforeEach(func() {
							scalingEngine.AppendHandlers(
								ghttp.CombineHandlers(
									ghttp.VerifyRequest("POST", urlPath),
									ghttp.RespondWithJSONEncoded(http.StatusOK, &scalingResult),
								),
							)
						})

						It("should succeed to set cooldownExpiredAt map", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("successfully-send-trigger-alarm with trigger")))
							lock.Lock()
							Eventually(cooldownExpired).Should(HaveLen(1))
							Eventually(cooldownExpired[testAppId]).Should(Equal(fakeTime.Add(time.Duration(300) * time.Second).UnixNano()))
							lock.Unlock()
						})
					})

					Context("when cooldownExpiredAt is 0 in scalingResult", func() {
						BeforeEach(func() {
							scalingResult.CooldownExpiredAt = 0

							scalingEngine.AppendHandlers(
								ghttp.CombineHandlers(
									ghttp.VerifyRequest("POST", urlPath),
									ghttp.RespondWithJSONEncoded(http.StatusOK, &scalingResult),
								),
							)
						})

						It("should be no record in coolDownExpiredAt map", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("successfully-send-trigger-alarm with trigger")))
							lock.Lock()
							Eventually(cooldownExpired).Should(HaveLen(0))
							lock.Unlock()
						})
					})

					Context("when cooldownExpiredAt is not set in scalingResult", func() {
						BeforeEach(func() {
							scalingEngine.AppendHandlers(
								ghttp.CombineHandlers(
									ghttp.VerifyRequest("POST", urlPath),
									ghttp.RespondWith(http.StatusOK, "{\"app_id\":\"testAppId\"}"),
								),
							)
						})

						It("should be no record in coolDownExpiredAt map", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("successfully-send-trigger-alarm with trigger")))
							lock.Lock()
							Eventually(cooldownExpired).Should(HaveLen(0))
							lock.Unlock()
						})
					})

					Context("when response is not a valid type scalingResult", func() {
						BeforeEach(func() {
							scalingEngine.AppendHandlers(
								ghttp.CombineHandlers(
									ghttp.VerifyRequest("POST", urlPath),
									ghttp.RespondWith(http.StatusOK, "succeed"),
								),
							)
						})

						It("should be no record in coolDownExpiredAt map", func() {
							Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("send trigger alarm to scaling engine")))
							Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("successfully-send-trigger-alarm, but received wrong response")))
							lock.Lock()
							Eventually(cooldownExpired).Should(HaveLen(0))
							lock.Unlock()
						})
					})
				})

				Context("when the scaling engine returns error", func() {
					BeforeEach(func() {
						scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWithJSONEncoded(http.StatusBadRequest, "error"))
					})

					It("should log the error", func() {
						Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("failed-send-trigger-alarm")))
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
						Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("failed-read-response-body-from-scaling-engine")))
					})
				})

			})

			Context("circuit break for scaling failures", func() {
				BeforeEach(func() {
					appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{600, 650, 620}, breachDurationSecs, true)
					queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
						return appMetrics, nil
					}

					bf := backoff.NewExponentialBackOff()
					bf.InitialInterval = 500 * time.Millisecond
					bf.MaxInterval = 1 * time.Second
					bf.RandomizationFactor = 0
					bf.Multiplier = 2
					bf.Reset()
					breaker := circuit.NewBreakerWithOptions(&circuit.Options{
						BackOff:    bf,
						ShouldTrip: circuit.ConsecutiveTripFunc(1),
					})
					cbEventChan = breaker.Subscribe()
					getBreaker = func(appID string) *circuit.Breaker {
						return breaker
					}
				})
				It("open and close the circuit breaker", func() {
					By("scaling failure causes circuit breaker to be open")
					scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWithJSONEncoded(http.StatusBadRequest, "error"))
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(1))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerFail)))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerTripped)))

					By("return directly when circuit breaker is open")
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("circuit-tripped")))
					Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(1))

					By("circuit breaker becomes half open when timeout")
					time.Sleep(500 * time.Millisecond)
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerReady)))
					Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(2))

					By("circuit breaker goes to open again for retry failuer")
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerFail)))
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(2))

					By("circuit breaker doubles the timeout interval")
					time.Sleep(500 * time.Millisecond)
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Consistently(scalingEngine.ReceivedRequests).Should(HaveLen(2))

					By("circuit breaker becomes half open second time when time out")
					time.Sleep(500 * time.Millisecond)
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerReady)))
					Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(3))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerFail)))

					By("circuit breaker caps the next timeout interval")
					time.Sleep(1 * time.Second)
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerReady)))
					Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(4))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerFail)))

					By("circuit breaker becomes closed due to successful scaling")
					scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWithJSONEncoded(http.StatusOK, &scalingResult))
					time.Sleep(1 * time.Second)
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerReady)))
					Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(5))

					By("Circuit breaker sends request to scaling engine when it is closed")
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(6))

					By("Circuit breaker resets timeout interval when it is closed ")
					scalingEngine.RouteToHandler("POST", urlPath, ghttp.RespondWithJSONEncoded(http.StatusBadRequest, "error"))
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Eventually(scalingEngine.ReceivedRequests).Should(HaveLen(7))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerFail)))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerTripped)))
					time.Sleep(500 * time.Millisecond)
					Expect(triggerChan).To(BeSent(triggerArrayGT))
					Eventually(cbEventChan).Should(Receive(Equal(circuit.BreakerReady)))
				})
			})

			Context("when retrieving appMetrics  failed", func() {
				BeforeEach(func() {
					queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
						return nil, errors.New("an error")
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
						BreachDurationSeconds: breachDurationSecs,
						CoolDownSeconds:       300,
						Threshold:             500,
						Operator:              "invalid_operator",
						Adjustment:            "1",
					}}
					triggerChan = make(chan []*models.Trigger, 1)
					Eventually(triggerChan).Should(BeSent(invalidTriggerArray))
				})

				It("should log the error", func() {
					Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("operator-is-invalid")))
				})
			})

		})
	})

	Context("Stop", func() {
		BeforeEach(func() {
			scalingEngine = ghttp.NewServer()
			queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
				return nil, nil
			}
			evaluator = NewEvaluator(logger, httpClient, scalingEngine.URL(), triggerChan, breachDurationSecs, queryAppMetrics, getBreaker, setCoolDownExpired)
			evaluator.Start()
			Expect(triggerChan).To(BeSent(triggerArrayGT))

			evaluator.Stop()
		})

		It("should stop to send trigger alarm", func() {
			Eventually(triggerChan).ShouldNot(BeSent(triggerArrayGT))
		})
	})

	Context("Scaling Engine is not reachable", func() {
		BeforeEach(func() {
			scalingEngine = ghttp.NewUnstartedServer()
			appMetrics := generateTestAppMetrics(testAppId, testMetricType, testMetricUnit, []int64{600, 650, 620}, breachDurationSecs, true)
			queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
				return appMetrics, nil
			}
			evaluator = NewEvaluator(logger, httpClient, scalingEngine.URL(), triggerChan, breachDurationSecs, queryAppMetrics, getBreaker, setCoolDownExpired)
			evaluator.Start()
			Expect(triggerChan).To(BeSent(triggerArrayGT))
		})

		AfterEach(func() {
			evaluator.Stop()
			scalingEngine.Close()
		})

		It("should log the error", func() {
			Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("failed-send-trigger-alarm-request")))
		})
	})
})
