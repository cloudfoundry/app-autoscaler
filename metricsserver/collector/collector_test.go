package collector_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsserver/collector"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Collector", func() {

	var (
		policyDb                      *fakes.FakePolicyDB
		instanceMetricsDb             *fakes.FakeInstanceMetricsDB
		fclock                        *fakeclock.FakeClock
		buffer                        *gbytes.Buffer
		logger                        *lagertest.TestLogger
		nodeNum                       int
		nodeIndex                     int
		metric1, metric2, metric3     *models.AppInstanceMetric
		isMetricPersistencySuppported bool
		metricCacheSizePerApp         int
		mc                            MetricCollector
		metricsChan                   chan *models.AppInstanceMetric
	)

	BeforeEach(func() {
		nodeNum = 1
		nodeIndex = 0
		policyDb = &fakes.FakePolicyDB{}
		instanceMetricsDb = &fakes.FakeInstanceMetricsDB{}
		logger = lagertest.NewTestLogger("collector-test")
		buffer = logger.Buffer()
		fclock = fakeclock.NewFakeClock(time.Now())
		isMetricPersistencySuppported = false
		metricCacheSizePerApp = 3
		metricsChan = make(chan *models.AppInstanceMetric, 10)
		metric1 = &models.AppInstanceMetric{
			AppId:         "an-app-id",
			InstanceIndex: 0,
			CollectedAt:   2222,
			Name:          models.MetricNameThroughput,
			Unit:          models.UnitRPS,
			Value:         "3",
			Timestamp:     2222,
		}
		metric2 = &models.AppInstanceMetric{
			AppId:         "an-app-id",
			InstanceIndex: 0,
			CollectedAt:   3333,
			Name:          models.MetricNameThroughput,
			Unit:          models.UnitRPS,
			Value:         "5",
			Timestamp:     3333,
		}

		metric3 = &models.AppInstanceMetric{
			AppId:         "an-app-id",
			InstanceIndex: 0,
			CollectedAt:   6666,
			Name:          models.MetricNameThroughput,
			Unit:          models.UnitRPS,
			Value:         "5",
			Timestamp:     6666,
		}

	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			mc = NewCollector(logger, TestRefreshInterval, TestCollectInterval, isMetricPersistencySuppported, TestSaveInterval,
				nodeIndex, nodeNum, metricCacheSizePerApp, policyDb, instanceMetricsDb, fclock, metricsChan)
			mc.Start()
		})

		AfterEach(func() {
			mc.Stop()
		})

		It("refreshes the apps with given interval", func() {
			Eventually(policyDb.GetAppIdsCallCount).Should(Equal(1), "policyDb.GetAppIds called in poll loop")

			fclock.Increment(TestRefreshInterval)
			Eventually(policyDb.GetAppIdsCallCount).Should(Equal(2), "policyDb.GetAppIds called in poll loop")

			fclock.Increment(TestRefreshInterval)
			Eventually(policyDb.GetAppIdsCallCount).Should(Equal(3), "policyDb.GetAppIds called in poll loop")

		})

		Context("when getting apps from policy db succeeds", func() {
			Context("when no apps in policy db", func() {
				BeforeEach(func() {
					policyDb.GetAppIdsReturns(make(map[string]bool), nil)
				})

				It("does nothing", func() {
					Consistently(mc.GetAppIDs).Should(BeEmpty())

					fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
					Consistently(mc.GetAppIDs).Should(BeEmpty())
				})
			})

			Context("when there are apps in policy db", func() {
				BeforeEach(func() {
					policyDb.GetAppIdsStub = func() (map[string]bool, error) {
						switch policyDb.GetAppIdsCallCount() {
						case 1:
							return map[string]bool{"app-id-1": true, "app-id-3": true}, nil
						case 2:
							return map[string]bool{"app-id-2": true, "app-id-3": true}, nil
						default:
							return map[string]bool{"app-id-1": true, "app-id-2": true}, nil
						}
					}
				})

				It("retrieves the app IDs", func() {
					Eventually(mc.GetAppIDs).Should(Equal(map[string]bool{"app-id-1": true, "app-id-3": true}), "mc.GetAppIds match after poll loop %d", policyDb.GetAppIdsCallCount())

					fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
					Eventually(mc.GetAppIDs).Should(Equal(map[string]bool{"app-id-2": true, "app-id-3": true}), "mc.GetAppIds match after poll loop %d", policyDb.GetAppIdsCallCount())

					fclock.Increment(TestRefreshInterval)
					Eventually(mc.GetAppIDs).Should(Equal(map[string]bool{"app-id-1": true, "app-id-2": true}), "mc.GetAppIds match after poll loop %d", policyDb.GetAppIdsCallCount())
				})

			})

			Context("when running with 3 nodes", func() {
				BeforeEach(func() {
					nodeNum = 3
					policyDb.GetAppIdsStub = func() (map[string]bool, error) {
						switch policyDb.GetAppIdsCallCount() {
						case 1:
							return map[string]bool{"app-id-1": true, "app-id-2": true, "app-id-3": true, "app-id-4": true}, nil
						case 2:
							return map[string]bool{"app-id-3": true, "app-id-4": true, "app-id-5": true, "app-id-6": true}, nil
						default:
							return map[string]bool{"app-id-5": true, "app-id-6": true, "app-id-7": true, "app-id-8": true}, nil
						}
					}
				})
				Context("when current index is 0", func() {
					BeforeEach(func() {
						nodeIndex = 0
					})
					It("retrieves app shard 0", func() {
						Eventually(mc.GetAppIDs).Should(Equal(map[string]bool{"app-id-3": true, "app-id-4": true}))

						fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
						Eventually(mc.GetAppIDs).Should(Equal(map[string]bool{"app-id-3": true, "app-id-4": true}))

						fclock.Increment(TestRefreshInterval)
						Eventually(mc.GetAppIDs).Should(Equal(map[string]bool{"app-id-8": true}))
					})
				})
				Context("when current index is 1", func() {
					BeforeEach(func() {
						nodeIndex = 1
					})
					It("retrieves app shard 1", func() {
						Consistently(mc.GetAppIDs).Should(BeEmpty())

						fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
						Eventually(mc.GetAppIDs).Should(Equal(map[string]bool{"app-id-5": true, "app-id-6": true}))

						fclock.Increment(TestRefreshInterval)
						Consistently(mc.GetAppIDs).Should(Equal(map[string]bool{"app-id-5": true, "app-id-6": true}))

					})
				})
				Context("when current index is 2", func() {
					BeforeEach(func() {
						nodeIndex = 2
					})
					It("retrieves app shard 2", func() {
						Eventually(mc.GetAppIDs).Should(Equal(map[string]bool{"app-id-1": true, "app-id-2": true}))

						fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
						Eventually(mc.GetAppIDs).Should(BeEmpty())

						fclock.Increment(TestRefreshInterval)
						Eventually(mc.GetAppIDs).Should(Equal(map[string]bool{"app-id-7": true}))

					})
				})

			})

		})

		Context("when getting apps from policy db fails", func() {
			BeforeEach(func() {
				policyDb.GetAppIdsReturns(nil, errors.New("test collector error"))
			})

			It("logs the error", func() {
				Eventually(buffer).Should(gbytes.Say("test collector error"))
				Consistently(mc.GetAppIDs).Should(BeEmpty())

				fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
				Eventually(buffer).Should(gbytes.Say("test collector error"))
				Consistently(mc.GetAppIDs).Should(BeEmpty())
			})
		})

		Context("when there are incoming metrics", func() {
			BeforeEach(func() {
				policyDb.GetAppIdsReturns(map[string]bool{"an-app-id": true}, nil)
				Expect(metricsChan).Should(BeSent(metric1))
				Expect(metricsChan).Should(BeSent(metric2))
				Expect(metricsChan).Should(BeSent(metric3))

			})
			Context("when metric persistency is not supported", func() {
				It("does not save metrics to db", func() {
					Consistently(instanceMetricsDb.SaveMetricCallCount).Should(BeZero())
					fclock.WaitForWatcherAndIncrement(TestSaveInterval)
					Consistently(instanceMetricsDb.SaveMetricCallCount).Should(BeZero())
				})
			})
			Context("when metric persistency is supported", func() {
				BeforeEach(func() {
					isMetricPersistencySuppported = true
				})
				It("save metrics to db", func() {
					Consistently(instanceMetricsDb.SaveMetricCallCount).Should(BeZero())

					fclock.WaitForWatcherAndIncrement(TestSaveInterval)
					Eventually(instanceMetricsDb.SaveMetricsInBulkCallCount).Should(Equal(1))
				})
			})
		})
		Context("when there is no metrics", func() {
			BeforeEach(func() {
				policyDb.GetAppIdsReturns(map[string]bool{"an-app-id": true}, nil)

			})
			Context("when metric persistency is supported", func() {
				BeforeEach(func() {
					isMetricPersistencySuppported = true
				})
				It("does not save metrics to db", func() {
					Consistently(instanceMetricsDb.SaveMetricCallCount).Should(BeZero())

					fclock.WaitForWatcherAndIncrement(TestSaveInterval)
					Consistently(instanceMetricsDb.SaveMetricsInBulkCallCount).Should(BeZero())
				})
			})
		})
	})

	Describe("QueryMetrics", func() {
		JustBeforeEach(func() {
			policyDb.GetAppIdsReturns(map[string]bool{"an-app-id": true}, nil)
			mc = NewCollector(logger, TestRefreshInterval, TestCollectInterval, isMetricPersistencySuppported, TestSaveInterval,
				nodeIndex, nodeNum, metricCacheSizePerApp, policyDb, instanceMetricsDb, fclock, metricsChan)
			mc.Start()
			Eventually(mc.GetAppIDs).Should(Equal(map[string]bool{"an-app-id": true}))

			Expect(metricsChan).To(BeSent(metric1))
			Expect(metricsChan).To(BeSent(metric2))
			Expect(metricsChan).To(BeSent(metric3))

		})

		AfterEach(func() {
			mc.Stop()
		})

		Context("when cache hits", func() {
			It("returns the metrics in cache", func() {
				time.Sleep(100 * time.Millisecond)
				result, err := mc.QueryMetrics("an-app-id", 0, models.MetricNameThroughput, 2222, 5555, db.ASC)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal([]*models.AppInstanceMetric{metric1, metric2}))
				Expect(instanceMetricsDb.RetrieveInstanceMetricsCallCount()).To(BeZero())
			})
		})

		Context("when cache misses", func() {
			BeforeEach(func() {
				metricCacheSizePerApp = 2
			})
			Context("when metrics persistency is not supported", func() {
				It("return empty result", func() {
					time.Sleep(100 * time.Millisecond)
					result, err := mc.QueryMetrics("an-app-id", 0, models.MetricNameThroughput, 0, 3331, db.ASC)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeEmpty())
				})
			})
			Context("when metrics persistency is supported", func() {
				BeforeEach(func() {
					isMetricPersistencySuppported = true
				})
				It("retrieves metrics from database", func() {
					time.Sleep(100 * time.Millisecond)
					_, err := mc.QueryMetrics("an-app-id", 0, models.MetricNameThroughput, 0, 3331, db.ASC)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(instanceMetricsDb.RetrieveInstanceMetricsCallCount()).Should(Equal(1))
				})

				Context("when retrieving metrics from db fails", func() {
					BeforeEach(func() {
						instanceMetricsDb.RetrieveInstanceMetricsReturns(nil, errors.New("an error"))
					})
					It("returns the error", func() {
						time.Sleep(100 * time.Millisecond)
						result, err := mc.QueryMetrics("an-app-id", 0, models.MetricNameThroughput, 2222, 5555, db.ASC)
						Expect(instanceMetricsDb.RetrieveInstanceMetricsCallCount()).Should(Equal(1))
						Expect(err).Should(HaveOccurred())
						Eventually(result).Should(BeNil())
					})
				})
			})
		})
	})

})
