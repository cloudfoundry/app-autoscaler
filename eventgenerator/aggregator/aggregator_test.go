package aggregator_test

import (
	. "autoscaler/eventgenerator/aggregator"
	. "autoscaler/eventgenerator/model"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aggregator", func() {
	var (
		getPolicies      GetPolicies
		aggregator       *Aggregator
		clock            *fakeclock.FakeClock
		logger           lager.Logger
		appMonitorsChan  chan *AppMonitor
		testAppId        string = "testAppId"
		fakeWaitDuration time.Duration
		policyMap        = map[string]*Policy{
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
		}
	)

	BeforeEach(func() {
		getPolicies = func() map[string]*Policy {
			return policyMap
		}

		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("Aggregator-test")

		appMonitorsChan = make(chan *AppMonitor, 10)
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
