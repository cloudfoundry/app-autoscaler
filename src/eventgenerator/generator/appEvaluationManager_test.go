package generator_test

import (
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
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

var _ = Describe("AppEvaluationManager", func() {

	var (
		logger               lager.Logger
		rclock               clock.Clock
		fclock               *fakeclock.FakeClock
		manager              *AppEvaluationManager
		testEvaluateInterval time.Duration
		testEvaluatorCount   int
		database             *fakes.FakeAppMetricDB
		triggerArrayChan     chan []*Trigger
		testAppId            string = "testAppId"
		testAppId2           string = "testAppId2"
		testMetricType       string = "MemoeryUsage"
		fakeScalingEngine    *ghttp.Server
		regPath                                    = regexp.MustCompile(`^/v1/apps/.*/scale$`)
		testMap              map[string][]*Trigger = map[string][]*Trigger{
			testAppId + "#" + testMetricType: []*Trigger{&Trigger{
				AppId:            testAppId,
				MetricType:       testMetricType,
				BreachDuration:   300,
				CoolDownDuration: 300,
				Threshold:        80,
				Operator:         ">",
				Adjustment:       "1",
			}, &Trigger{
				AppId:            testAppId,
				MetricType:       testMetricType,
				BreachDuration:   300,
				CoolDownDuration: 300,
				Threshold:        30,
				Operator:         "<",
				Adjustment:       "-1",
			}},
		}

		testMap2 map[string][]*Trigger = map[string][]*Trigger{
			testAppId2 + "#" + testMetricType: []*Trigger{&Trigger{
				AppId:            testAppId2,
				MetricType:       testMetricType,
				BreachDuration:   300,
				CoolDownDuration: 300,
				Threshold:        80,
				Operator:         ">",
				Adjustment:       "1",
			}, &Trigger{
				AppId:            testAppId2,
				MetricType:       testMetricType,
				BreachDuration:   300,
				CoolDownDuration: 300,
				Threshold:        30,
				Operator:         "<",
				Adjustment:       "-1",
			}},
		}
	)
	BeforeEach(func() {
		fclock = fakeclock.NewFakeClock(time.Now())
		rclock = clock.NewClock()
		testEvaluateInterval = 1 * time.Second
		logger = lagertest.NewTestLogger("ApplicationManager-test")
		triggerArrayChan = make(chan []*Trigger, 10)
		fakeScalingEngine = ghttp.NewServer()
		fakeScalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
		database = &fakes.FakeAppMetricDB{}
		testEvaluatorCount = 0
		manager = NewAppEvaluationManager(testEvaluateInterval, logger, fclock, triggerArrayChan, testEvaluatorCount, database, fakeScalingEngine.URL())
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			manager.Start()
			Eventually(fclock.WatcherCount).Should(Equal(1))
		})

		AfterEach(func() {
			manager.Stop()
		})

		Context("EvaluatorArray", func() {
			var unBlockChan chan bool
			var calledChan chan string
			BeforeEach(func() {
				testEvaluatorCount = 4
				manager = NewAppEvaluationManager(testEvaluateInterval, logger, fclock, triggerArrayChan, testEvaluatorCount, database, fakeScalingEngine.URL())
				unBlockChan = make(chan bool)
				calledChan = make(chan string)
				database.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
					defer GinkgoRecover()
					calledChan <- appId
					<-unBlockChan
					return nil, nil
				}
				testTriggerMap := map[string][]*Trigger{
					"id1" + "#" + testMetricType: []*Trigger{&Trigger{
						AppId:            "id1",
						MetricType:       testMetricType,
						BreachDuration:   300,
						CoolDownDuration: 300,
						Threshold:        80,
						Operator:         ">",
						Adjustment:       "1",
					}},
					"id2" + "#" + testMetricType: []*Trigger{&Trigger{
						AppId:            "id2",
						MetricType:       testMetricType,
						BreachDuration:   300,
						CoolDownDuration: 300,
						Threshold:        80,
						Operator:         ">",
						Adjustment:       "1",
					}},
					"id3" + "#" + testMetricType: []*Trigger{&Trigger{
						AppId:            "id3",
						MetricType:       testMetricType,
						BreachDuration:   300,
						CoolDownDuration: 300,
						Threshold:        80,
						Operator:         ">",
						Adjustment:       "1",
					}},
					"id4" + "#" + testMetricType: []*Trigger{&Trigger{
						AppId:            "id4",
						MetricType:       testMetricType,
						BreachDuration:   300,
						CoolDownDuration: 300,
						Threshold:        80,
						Operator:         ">",
						Adjustment:       "1",
					}},
					"id5" + "#" + testMetricType: []*Trigger{&Trigger{
						AppId:            "id5",
						MetricType:       testMetricType,
						BreachDuration:   300,
						CoolDownDuration: 300,
						Threshold:        80,
						Operator:         ">",
						Adjustment:       "1",
					}},
				}
				manager.SetTriggers(testTriggerMap)
			})
			It("should create evaluatorCount evaluators", func() {
				fclock.Increment(1 * testEvaluateInterval)
				for i := 0; i < testEvaluatorCount; i++ {
					Eventually(calledChan).Should(Receive())

				}
				Consistently(calledChan).ShouldNot(Receive())
				fclock.Increment(1 * testEvaluateInterval)
				Eventually(fclock.WatcherCount).Should(Equal(1))
				Eventually(database.RetrieveAppMetricsCallCount).Should(Equal(int(testEvaluatorCount)))

				close(unBlockChan)
			})
		})

		Context("when there are triggers for evaluation", func() {
			BeforeEach(func() {
				manager.SetTriggers(testMap)

			})

			It("should add triggers to evaluate", func() {
				fclock.Increment(10 * testEvaluateInterval)
				var arr []*Trigger
				Eventually(triggerArrayChan).Should(Receive(&arr))
				Expect(arr).To(Equal([]*Trigger{&Trigger{
					AppId:            testAppId,
					MetricType:       testMetricType,
					BreachDuration:   300,
					CoolDownDuration: 300,
					Threshold:        80,
					Operator:         ">",
					Adjustment:       "1",
				}, &Trigger{
					AppId:            testAppId,
					MetricType:       testMetricType,
					BreachDuration:   300,
					CoolDownDuration: 300,
					Threshold:        30,
					Operator:         "<",
					Adjustment:       "-1",
				}}))
			})
		})

		Context("when there is trigger", func() {
			BeforeEach(func() {
				manager.SetTriggers(map[string][]*Trigger{})
			})

			It("should add no trigger to evaluate", func() {
				Consistently(triggerArrayChan).ShouldNot(Receive())
			})
		})
		Context("when triggers are changed", func() {
			BeforeEach(func() {
				manager.SetTriggers(testMap)
			})

			It("should add new triggers to evaluate", func() {
				fclock.Increment(1 * testEvaluateInterval)
				var arr []*Trigger
				Eventually(triggerArrayChan).Should(Receive(&arr))

				Expect(arr).To(Equal([]*Trigger{&Trigger{
					AppId:            testAppId,
					MetricType:       testMetricType,
					BreachDuration:   300,
					CoolDownDuration: 300,
					Threshold:        80,
					Operator:         ">",
					Adjustment:       "1",
				}, &Trigger{
					AppId:            testAppId,
					MetricType:       testMetricType,
					BreachDuration:   300,
					CoolDownDuration: 300,
					Threshold:        30,
					Operator:         "<",
					Adjustment:       "-1",
				}}))

				manager.SetTriggers(testMap2)
				fclock.Increment(1 * testEvaluateInterval)
				Eventually(triggerArrayChan).Should(Receive(&arr))

				Expect(arr).To(Equal([]*Trigger{&Trigger{
					AppId:            testAppId2,
					MetricType:       testMetricType,
					BreachDuration:   300,
					CoolDownDuration: 300,
					Threshold:        80,
					Operator:         ">",
					Adjustment:       "1",
				}, &Trigger{
					AppId:            testAppId2,
					MetricType:       testMetricType,
					BreachDuration:   300,
					CoolDownDuration: 300,
					Threshold:        30,
					Operator:         "<",
					Adjustment:       "-1",
				}}))
			})
		})

	})

	Describe("Stop", func() {
		BeforeEach(func() {
			manager.Start()
			Eventually(fclock.WatcherCount).Should(Equal(1))
			manager.Stop()
			Eventually(fclock.WatcherCount).Should(Equal(0))
			manager.SetTriggers(testMap)
		})

		It("stops adding triggers to evaluate ", func() {
			fclock.Increment(1 * testEvaluateInterval)
			Consistently(triggerArrayChan).ShouldNot(Receive())
		})
	})
})
