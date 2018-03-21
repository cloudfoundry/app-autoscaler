package generator_test

import (
	"autoscaler/eventgenerator/aggregator/fakes"
	"autoscaler/eventgenerator/config"
	. "autoscaler/eventgenerator/generator"
	"autoscaler/models"
	"net/http"
	"regexp"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/rubyist/circuitbreaker"
)

var _ = Describe("AppEvaluationManager", func() {

	var (
		getPolicies          models.GetPolicies
		logger               lager.Logger
		fclock               *fakeclock.FakeClock
		manager              *AppEvaluationManager
		testEvaluateInterval time.Duration
		testEvaluatorCount   int
		database             *fakes.FakeAppMetricDB
		triggerArrayChan     chan []*models.Trigger
		testAppId1           string                      = "testAppId1"
		testAppId2           string                      = "testAppId2"
		testMetricName       string                      = "Test-Metric-Name"
		testBreakerConfig    config.CircuitBreakerConfig = config.CircuitBreakerConfig{}
		fakeScalingEngine    *ghttp.Server
		regPath              *regexp.Regexp = regexp.MustCompile(`^/v1/apps/.*/scale$`)

		appPolicy1 = &models.AppPolicy{
			AppId: testAppId1,
			ScalingPolicy: &models.ScalingPolicy{
				InstanceMax: 5,
				InstanceMin: 1,
				ScalingRules: []*models.ScalingRule{
					{
						MetricType:            testMetricName,
						StatWindowSeconds:     200,
						BreachDurationSeconds: 200,
						CoolDownSeconds:       200,
						Threshold:             80,
						Operator:              ">=",
						Adjustment:            "1",
					},
				},
			},
		}

		appPolicy2 = &models.AppPolicy{
			AppId: testAppId2,
			ScalingPolicy: &models.ScalingPolicy{
				InstanceMax: 5,
				InstanceMin: 1,
				ScalingRules: []*models.ScalingRule{
					{
						MetricType:            testMetricName,
						StatWindowSeconds:     300,
						BreachDurationSeconds: 300,
						CoolDownSeconds:       300,
						Threshold:             20,
						Operator:              "<=",
						Adjustment:            "-1",
					},
				},
			},
		}
	)

	BeforeEach(func() {
		fclock = fakeclock.NewFakeClock(time.Now())
		testEvaluateInterval = 1 * time.Second
		logger = lagertest.NewTestLogger("ApplicationManager-test")
		triggerArrayChan = make(chan []*models.Trigger, 10)
		fakeScalingEngine = ghttp.NewServer()
		fakeScalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
		database = &fakes.FakeAppMetricDB{}
		testEvaluatorCount = 0
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			var err error
			manager, err = NewAppEvaluationManager(logger, testEvaluateInterval, fclock, triggerArrayChan, getPolicies, testBreakerConfig)
			Expect(err).NotTo(HaveOccurred())
			manager.Start()
			Eventually(fclock.WatcherCount).Should(Equal(1))
		})

		AfterEach(func() {
			manager.Stop()
		})

		Context("when there are triggers for evaluation", func() {
			BeforeEach(func() {
				getPolicies = func() map[string]*models.AppPolicy {
					return map[string]*models.AppPolicy{
						testAppId1: appPolicy1,
						testAppId2: appPolicy2,
					}
				}
			})

			It("should add triggers to evaluate", func() {
				fclock.Increment(10 * testEvaluateInterval)
				var arr []*models.Trigger
				var triggerArray [][]*models.Trigger = [][]*models.Trigger{}
				Eventually(triggerArrayChan).Should(Receive(&arr))
				triggerArray = append(triggerArray, arr)
				Eventually(triggerArrayChan).Should(Receive(&arr))
				triggerArray = append(triggerArray, arr)
				Expect(triggerArray).Should(ContainElement(
					[]*models.Trigger{{
						AppId:                 testAppId1,
						MetricType:            testMetricName,
						BreachDurationSeconds: 200,
						CoolDownSeconds:       200,
						Threshold:             80,
						Operator:              ">=",
						Adjustment:            "1",
					}}))
				Expect(triggerArray).Should(ContainElement(
					[]*models.Trigger{{
						AppId:                 testAppId2,
						MetricType:            testMetricName,
						BreachDurationSeconds: 300,
						CoolDownSeconds:       300,
						Threshold:             20,
						Operator:              "<=",
						Adjustment:            "-1",
					}}))
			})
		})

		Context("when there is no trigger", func() {
			BeforeEach(func() {
				getPolicies = func() map[string]*models.AppPolicy {
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
			getPolicies = func() map[string]*models.AppPolicy {
				return map[string]*models.AppPolicy{testAppId1: appPolicy1}
			}

			var err error
			manager, err = NewAppEvaluationManager(logger, testEvaluateInterval, fclock, triggerArrayChan, getPolicies, testBreakerConfig)
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

	Describe("GetBreaker", func() {
		BeforeEach(func() {
			index := 0
			getPolicies = func() map[string]*models.AppPolicy {
				index++
				switch index {
				case 1:
					return map[string]*models.AppPolicy{testAppId1: appPolicy1}
				case 2:
					return map[string]*models.AppPolicy{testAppId1: appPolicy1, testAppId2: appPolicy2}
				case 3:
					return map[string]*models.AppPolicy{testAppId2: appPolicy2}
				}
				return map[string]*models.AppPolicy{testAppId1: appPolicy1}
			}
			var err error
			manager, err = NewAppEvaluationManager(logger, testEvaluateInterval, fclock, triggerArrayChan, getPolicies, testBreakerConfig)
			Expect(err).NotTo(HaveOccurred())
			manager.Start()
			Eventually(fclock.WatcherCount).Should(Equal(1))
		})
		It("retrieves the right breaker", func() {
			Expect(manager.GetBreaker(testAppId1)).To(BeNil())

			fclock.Increment(1 * testEvaluateInterval)
			Eventually(func() *circuit.Breaker { return manager.GetBreaker(testAppId1) }).ShouldNot(BeNil())
			breaker1 := manager.GetBreaker(testAppId1)

			fclock.Increment(1 * testEvaluateInterval)
			Eventually(func() *circuit.Breaker { return manager.GetBreaker(testAppId2) }).ShouldNot(BeNil())
			breaker2 := manager.GetBreaker(testAppId2)
			Expect(manager.GetBreaker(testAppId1)).To(BeIdenticalTo(breaker1))

			fclock.Increment(1 * testEvaluateInterval)
			Eventually(func() *circuit.Breaker { return manager.GetBreaker(testAppId1) }).Should(BeNil())
			Expect(manager.GetBreaker(testAppId2)).To(BeIdenticalTo(breaker2))

		})
		AfterEach(func() {
			manager.Stop()
		})
	})
})
