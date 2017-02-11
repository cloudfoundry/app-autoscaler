package aggregator_test

import (
	. "autoscaler/eventgenerator/aggregator"
	"autoscaler/models"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aggregator", func() {
	var (
		getPolicies      models.GetPolicies
		aggregator       *Aggregator
		clock            *fakeclock.FakeClock
		logger           lager.Logger
		appMonitorsChan  chan *models.AppMonitor
		testAppId        string = "testAppId"
		fakeWaitDuration time.Duration
		policyMap        = map[string]*models.AppPolicy{
			testAppId: &models.AppPolicy{
				AppId: testAppId,
				ScalingPolicy: &models.ScalingPolicy{
					InstanceMax: 5,
					InstanceMin: 1,
					ScalingRules: []*models.ScalingRule{
						&models.ScalingRule{
							MetricType:            models.MetricNameMemory,
							StatWindowSeconds:     300,
							BreachDurationSeconds: 300,
							CoolDownSeconds:       300,
							Threshold:             30,
							Operator:              "<",
							Adjustment:            "-1",
						},
					},
				},
			},
		}
	)

	BeforeEach(func() {
		getPolicies = func() map[string]*models.AppPolicy {
			return policyMap
		}

		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("Aggregator-test")

		appMonitorsChan = make(chan *models.AppMonitor, 10)
		if testEvaluateInteval > testAggregatorExecuteInterval {
			fakeWaitDuration = testEvaluateInteval
		} else {
			fakeWaitDuration = testAggregatorExecuteInterval
		}
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			var err error
			aggregator, err = NewAggregator(logger, clock, testAggregatorExecuteInterval, appMonitorsChan, getPolicies)
			Expect(err).NotTo(HaveOccurred())
			aggregator.Start()
			Eventually(clock.WatcherCount).Should(Equal(1))
		})

		AfterEach(func() {
			aggregator.Stop()
		})

		It("should send appMonitors", func() {
			clock.Increment(1 * fakeWaitDuration)
			Eventually(appMonitorsChan).Should(Receive())
		})
	})

	Describe("Stop", func() {
		JustBeforeEach(func() {
			var err error
			aggregator, err = NewAggregator(logger, clock, testAggregatorExecuteInterval, appMonitorsChan, getPolicies)
			Expect(err).NotTo(HaveOccurred())
			aggregator.Start()
			Eventually(clock.WatcherCount).Should(Equal(1))
			aggregator.Stop()
		})

		It("should not send any appMetrics", func() {
			Eventually(clock.WatcherCount).Should(BeZero())

			clock.Increment(1 * fakeWaitDuration)
			Consistently(appMonitorsChan).ShouldNot(Receive())
		})
	})
})
