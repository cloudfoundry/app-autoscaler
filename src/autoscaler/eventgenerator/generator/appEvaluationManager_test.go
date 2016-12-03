package generator_test

import (
	"autoscaler/eventgenerator/aggregator/fakes"
	. "autoscaler/eventgenerator/generator"
	. "autoscaler/eventgenerator/model"
	"net/http"
	"regexp"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("AppEvaluationManager", func() {

	var (
		getPolicies          GetPolicies
		logger               lager.Logger
		fclock               *fakeclock.FakeClock
		manager              *AppEvaluationManager
		testEvaluateInterval time.Duration
		testEvaluatorCount   int
		database             *fakes.FakeAppMetricDB
		triggerArrayChan     chan []*Trigger
		testAppId            string = "testAppId"
		testAppId2           string = "testAppId2"
		testMetricType       string = "MemoryUsage"
		fakeScalingEngine    *ghttp.Server
		regPath                                 = regexp.MustCompile(`^/v1/apps/.*/scale$`)
		policyMap            map[string]*Policy = map[string]*Policy{
			testAppId: &Policy{
				AppId: testAppId,
				TriggerRecord: &TriggerRecord{
					InstanceMaxCount: 5,
					InstanceMinCount: 1,
					ScalingRules: []*ScalingRule{
						&ScalingRule{
							MetricType:       "MemoryUsage",
							StatWindow:       300,
							BreachDuration:   300,
							CoolDownDuration: 300,
							Threshold:        30,
							Operator:         "<",
							Adjustment:       "-1",
						},
					},
				},
			},
			testAppId2: &Policy{
				AppId: testAppId2,
				TriggerRecord: &TriggerRecord{
					InstanceMaxCount: 5,
					InstanceMinCount: 1,
					ScalingRules: []*ScalingRule{
						&ScalingRule{
							MetricType:       "MemoryUsage",
							StatWindow:       300,
							BreachDuration:   300,
							CoolDownDuration: 300,
							Threshold:        30,
							Operator:         "<",
							Adjustment:       "-1",
						},
					},
				},
			},
		}
	)

	BeforeEach(func() {
		fclock = fakeclock.NewFakeClock(time.Now())
		testEvaluateInterval = 1 * time.Second
		logger = lagertest.NewTestLogger("ApplicationManager-test")
		triggerArrayChan = make(chan []*Trigger, 10)
		fakeScalingEngine = ghttp.NewServer()
		fakeScalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
		database = &fakes.FakeAppMetricDB{}
		testEvaluatorCount = 0
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			var err error
			manager, err = NewAppEvaluationManager(logger, testEvaluateInterval, fclock, triggerArrayChan, getPolicies)
			Expect(err).NotTo(HaveOccurred())
			manager.Start()
			Eventually(fclock.WatcherCount).Should(Equal(1))
		})

		AfterEach(func() {
			manager.Stop()
		})

		Context("when there are triggers for evaluation", func() {
			BeforeEach(func() {
				getPolicies = func() map[string]*Policy {
					return policyMap
				}
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
					Threshold:        30,
					Operator:         "<",
					Adjustment:       "-1",
				}}))

				Eventually(triggerArrayChan).Should(Receive(&arr))
				Expect(arr).To(Equal([]*Trigger{&Trigger{
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

		Context("when there is no trigger", func() {
			BeforeEach(func() {
				getPolicies = func() map[string]*Policy {
					return nil
				}
			})

			It("should add no trigger to evaluate", func() {
				fclock.Increment(10 * testEvaluateInterval)
				Consistently(triggerArrayChan).ShouldNot(Receive())
			})
		})
	})

	Describe("Stop", func() {
		BeforeEach(func() {
			getPolicies = func() map[string]*Policy {
				return policyMap
			}

			var err error
			manager, err = NewAppEvaluationManager(logger, testEvaluateInterval, fclock, triggerArrayChan, getPolicies)
			Expect(err).NotTo(HaveOccurred())
			manager.Start()
			Eventually(fclock.WatcherCount).Should(Equal(1))
			manager.Stop()
			Eventually(fclock.WatcherCount).Should(Equal(0))
		})

		It("stops adding triggers to evaluate ", func() {
			fclock.Increment(1 * testEvaluateInterval)
			Consistently(triggerArrayChan).ShouldNot(Receive())
		})
	})
})
