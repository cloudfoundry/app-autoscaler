package generator_test

import (
	"autoscaler/eventgenerator/aggregator/fakes"
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
		testAppId            string = "testAppId"
		testAppId2           string = "testAppId2"
		testMetricType       string = "MemoryUsage"
		fakeScalingEngine    *ghttp.Server
		regPath              *regexp.Regexp               = regexp.MustCompile(`^/v1/apps/.*/scale$`)
		policyMap            map[string]*models.AppPolicy = map[string]*models.AppPolicy{
			testAppId: &models.AppPolicy{
				AppId: testAppId,
				ScalingPolicy: &models.ScalingPolicy{
					InstanceMax: 5,
					InstanceMin: 1,
					ScalingRules: []*models.ScalingRule{
						&models.ScalingRule{
							MetricType:            "MemoryUsage",
							StatWindowSeconds:     200,
							BreachDurationSeconds: 200,
							CoolDownSeconds:       200,
							Threshold:             80,
							Operator:              ">=",
							Adjustment:            "1",
						},
					},
				},
			},
			testAppId2: &models.AppPolicy{
				AppId: testAppId2,
				ScalingPolicy: &models.ScalingPolicy{
					InstanceMax: 5,
					InstanceMin: 1,
					ScalingRules: []*models.ScalingRule{
						&models.ScalingRule{
							MetricType:            "MemoryUsage",
							StatWindowSeconds:     300,
							BreachDurationSeconds: 300,
							CoolDownSeconds:       300,
							Threshold:             20,
							Operator:              "<=",
							Adjustment:            "-1",
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
		triggerArrayChan = make(chan []*models.Trigger, 10)
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
				getPolicies = func() map[string]*models.AppPolicy {
					return policyMap
				}
			})

			It("should add triggers to evaluate", func() {
				fclock.Increment(10 * testEvaluateInterval)
				var arr []*models.Trigger
				Eventually(triggerArrayChan).Should(Receive(&arr))
				Expect(arr).To(Equal([]*models.Trigger{&models.Trigger{
					AppId:                 testAppId,
					MetricType:            testMetricType,
					BreachDurationSeconds: 200,
					CoolDownSeconds:       200,
					Threshold:             80,
					Operator:              ">=",
					Adjustment:            "1",
				}}))

				Eventually(triggerArrayChan).Should(Receive(&arr))
				Expect(arr).To(Equal([]*models.Trigger{&models.Trigger{
					AppId:                 testAppId2,
					MetricType:            testMetricType,
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
