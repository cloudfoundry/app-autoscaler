package aggregator_test

import (
	. "autoscaler/eventgenerator/aggregator"
	"autoscaler/eventgenerator/aggregator/fakes"
	"autoscaler/models"

	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aggregator", func() {
	const (
		fakeStatWindowSecs = 600
	)
	var (
		getPolicies      models.GetPolicies
		aggregator       *Aggregator
		clock            *fakeclock.FakeClock
		saveClock        *fakeclock.FakeClock
		logger           lager.Logger
		appMonitorsChan  chan *models.AppMonitor
		testAppId        string        = "testAppId"
		testMetricType   string        = "test-metric-name"
		testMetricUnit   string        = "a-metric-unit"
		fakeWaitDuration time.Duration = 0 * time.Millisecond
		policyMap                      = map[string]*models.AppPolicy{
			testAppId: {
				AppId: testAppId,
				ScalingPolicy: &models.ScalingPolicy{
					InstanceMax: 5,
					InstanceMin: 1,
					ScalingRules: []*models.ScalingRule{
						{
							MetricType:            testMetricType,
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
		appMetricDatabase *fakes.FakeAppMetricDB
		appMetricChan     chan *models.AppMetric
	)

	BeforeEach(func() {
		getPolicies = func() map[string]*models.AppPolicy {
			return policyMap
		}

		clock = fakeclock.NewFakeClock(time.Now())
		saveClock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("Aggregator-test")

		appMonitorsChan = make(chan *models.AppMonitor, 10)
		if testAggregatorExecuteInterval > fakeWaitDuration {
			fakeWaitDuration = testAggregatorExecuteInterval
		}
		if testSaveInterval > fakeWaitDuration {
			fakeWaitDuration = testSaveInterval
		}
		appMetricDatabase = &fakes.FakeAppMetricDB{}
		appMetricChan = make(chan *models.AppMetric, 1)
	})

	Context("Start", func() {
		JustBeforeEach(func() {
			var err error
			aggregator, err = NewAggregator(logger, clock, testAggregatorExecuteInterval, testSaveInterval, appMonitorsChan, getPolicies, fakeStatWindowSecs, appMetricChan, appMetricDatabase)
			Expect(err).NotTo(HaveOccurred())
			aggregator.Start()
			Expect(appMetricChan).Should(BeSent(&models.AppMetric{
				AppId:      testAppId,
				MetricType: testMetricType,
				Value:      "250",
				Unit:       testMetricUnit,
				Timestamp:  time.Now().UnixNano(),
			}))
			Eventually(clock.WatcherCount).Should(Equal(2))
		})

		AfterEach(func() {
			aggregator.Stop()
		})

		It("should send appMonitors and save appMetrics", func() {
			clock.Increment(1 * fakeWaitDuration)
			Eventually(appMonitorsChan).Should(Receive())
			Eventually(appMetricDatabase.SaveAppMetricsInBulkCallCount).Should(Equal(1))
		})
	})

	Context("Stop", func() {
		JustBeforeEach(func() {
			var err error
			aggregator, err = NewAggregator(logger, clock, testAggregatorExecuteInterval, testSaveInterval, appMonitorsChan, getPolicies, fakeStatWindowSecs, appMetricChan, appMetricDatabase)
			Expect(err).NotTo(HaveOccurred())
			aggregator.Start()
			Eventually(clock.WatcherCount).Should(Equal(2))
			aggregator.Stop()
			Expect(appMetricChan).Should(BeSent(&models.AppMetric{
				AppId:      testAppId,
				MetricType: testMetricType,
				Value:      "250",
				Unit:       testMetricUnit,
				Timestamp:  time.Now().UnixNano(),
			}))
		})

		It("should not send any appMetrics", func() {
			Eventually(clock.WatcherCount).Should(BeZero())

			clock.Increment(1 * fakeWaitDuration)
			Consistently(appMonitorsChan).ShouldNot(Receive())
			Consistently(appMetricDatabase.SaveAppMetricsInBulkCallCount).Should(Equal(0))
		})
	})
})
