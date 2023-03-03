package aggregator_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"sync"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aggregator", func() {
	const (
		fakeStatWindowSecs = 600
	)
	var (
		getPolicies          GetPoliciesFunc
		saveAppMetricToCache SaveAppMetricToCacheFunc
		aggregator           *Aggregator
		clock                *fakeclock.FakeClock
		logger               lager.Logger
		appMonitorsChan      chan *models.AppMonitor
		testAppId            = "testAppId"
		testMetricType       = "test-metric-name"
		testMetricUnit       = "a-metric-unit"
		fakeWaitDuration     = 0 * time.Millisecond
		policyMap            = map[string]*models.AppPolicy{
			testAppId: {
				AppId: testAppId,
				ScalingPolicy: &models.ScalingPolicy{
					InstanceMax: 5,
					InstanceMin: 1,
					ScalingRules: []*models.ScalingRule{
						{
							MetricType:            testMetricType,
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
		appMetricDatabase    *fakes.FakeAppMetricDB
		appMetricChan        chan *models.AppMetric
		cacheLock            sync.RWMutex
		saveToCacheCallCount int
	)

	BeforeEach(func() {
		getPolicies = func() map[string]*models.AppPolicy {
			return policyMap
		}

		saveToCacheCallCount = 0
		saveAppMetricToCache = func(metric *models.AppMetric) bool {
			cacheLock.Lock()
			saveToCacheCallCount++
			cacheLock.Unlock()
			return true
		}

		clock = fakeclock.NewFakeClock(time.Now())
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
			aggregator, err = NewAggregator(logger, clock, testAggregatorExecuteInterval, testSaveInterval, appMonitorsChan,
				getPolicies, saveAppMetricToCache, fakeStatWindowSecs, appMetricChan, appMetricDatabase)
			Expect(err).NotTo(HaveOccurred())
			aggregator.Start()
			Eventually(clock.WatcherCount).Should(Equal(2))
		})

		AfterEach(func() {
			aggregator.Stop()
		})
		Context("when there are incoming metrics", func() {
			BeforeEach(func() {
				Expect(appMetricChan).Should(BeSent(&models.AppMetric{
					AppId:      testAppId,
					MetricType: testMetricType,
					Value:      "250",
					Unit:       testMetricUnit,
					Timestamp:  time.Now().UnixNano(),
				}))
			})
			It("should send appMonitors and save appMetrics", func() {
				clock.Increment(1 * fakeWaitDuration)
				Eventually(appMonitorsChan).Should(Receive())
				Eventually(func() int {
					cacheLock.RLock()
					defer cacheLock.RUnlock()
					return saveToCacheCallCount
				}).Should(Equal(1))
				Eventually(appMetricDatabase.SaveAppMetricsInBulkCallCount).Should(Equal(1))
			})
		})
		Context("when there is no metrics", func() {
			It("does not save metrics to db", func() {
				clock.Increment(1 * fakeWaitDuration)
				Eventually(appMonitorsChan).Should(Receive())
				Eventually(func() int {
					cacheLock.RLock()
					defer cacheLock.RUnlock()
					return saveToCacheCallCount
				}).Should(BeZero())
				Eventually(appMetricDatabase.SaveAppMetricsInBulkCallCount).Should(BeZero())
			})
		})

	})

	Context("Stop", func() {
		JustBeforeEach(func() {
			var err error
			aggregator, err = NewAggregator(logger, clock, testAggregatorExecuteInterval, testSaveInterval, appMonitorsChan,
				getPolicies, saveAppMetricToCache, fakeStatWindowSecs, appMetricChan, appMetricDatabase)
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
