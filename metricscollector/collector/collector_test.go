package collector_test

import (
	"autoscaler/collection"
	"autoscaler/db"
	"autoscaler/fakes"
	. "autoscaler/metricscollector/collector"
	"autoscaler/models"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"errors"
	"time"
)

var _ = Describe("Collector", func() {

	var (
		policyDb                  *fakes.FakePolicyDB
		instanceMetricsDb         *fakes.FakeInstanceMetricsDB
		coll                      *Collector
		fclock                    *fakeclock.FakeClock
		appCollector              *fakes.FakeAppCollector
		buffer                    *gbytes.Buffer
		logger                    *lagertest.TestLogger
		nodeNum                   int
		nodeIndex                 int
		createAppCollector        func(string, chan *models.AppInstanceMetric) AppCollector
		metric1, metric2, metric3 *models.AppInstanceMetric
		persistMetrics            bool
	)

	BeforeEach(func() {
		nodeNum = 1
		nodeIndex = 0
		policyDb = &fakes.FakePolicyDB{}
		instanceMetricsDb = &fakes.FakeInstanceMetricsDB{}
		logger = lagertest.NewTestLogger("collector-test")
		buffer = logger.Buffer()
		fclock = fakeclock.NewFakeClock(time.Now())
		appCollector = &fakes.FakeAppCollector{}
		persistMetrics = false
		createAppCollector = func(appId string, dataChan chan *models.AppInstanceMetric) AppCollector {
			return appCollector
		}

	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			coll = NewCollector(TestRefreshInterval, TestCollectInterval, persistMetrics, TestSaveInterval, nodeIndex, nodeNum, logger, policyDb, instanceMetricsDb, fclock, createAppCollector)
			coll.Start()
		})

		AfterEach(func() {
			coll.Stop()
		})

		It("refreshes the apps with given interval", func() {
			Eventually(policyDb.GetAppIdsCallCount).Should(Equal(1))

			fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
			Eventually(policyDb.GetAppIdsCallCount).Should(Equal(2))

			fclock.Increment(TestRefreshInterval)
			Eventually(policyDb.GetAppIdsCallCount).Should(Equal(3))

		})

		Context("when getting apps from policy policyDb succeeds", func() {

			Context("when no apps in policy policyDb", func() {
				BeforeEach(func() {
					policyDb.GetAppIdsReturns(make(map[string]bool), nil)
				})

				It("does nothing", func() {
					Consistently(coll.GetCollectorAppIds).Should(BeEmpty())

					fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
					Consistently(coll.GetCollectorAppIds).Should(BeEmpty())
				})
			})

			Context("when refresh does not have app changes", func() {
				BeforeEach(func() {
					policyDb.GetAppIdsReturns(map[string]bool{"app-id-1": true, "app-id-2": true, "app-id-3": true}, nil)
				})

				It("should always poll the same set of apps", func() {
					Eventually(appCollector.StartCallCount).Should(Equal(3))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

					fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
					Consistently(appCollector.StartCallCount).Should(Equal(3))
					Consistently(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))
				})
			})

			Context("when refresh has new apps", func() {
				BeforeEach(func() {
					policyDb.GetAppIdsStub = func() (map[string]bool, error) {
						switch policyDb.GetAppIdsCallCount() {
						case 1:
							return map[string]bool{"app-id-1": true}, nil
						case 2:
							return map[string]bool{"app-id-1": true, "app-id-2": true}, nil
						default:
							return map[string]bool{"app-id-1": true, "app-id-2": true, "app-id-3": true}, nil
						}
					}
				})

				It("polls newly added ones", func() {
					Eventually(appCollector.StartCallCount).Should(Equal(1))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1"))

					fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
					Eventually(appCollector.StartCallCount).Should(Equal(2))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1", "app-id-2"))

					fclock.Increment(TestRefreshInterval)
					Eventually(appCollector.StartCallCount).Should(Equal(3))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

				})

			})

			Context("when refresh has app removals", func() {
				BeforeEach(func() {
					policyDb.GetAppIdsStub = func() (map[string]bool, error) {
						switch policyDb.GetAppIdsCallCount() {
						case 1:
							return map[string]bool{"app-id-1": true, "app-id-2": true, "app-id-3": true}, nil
						case 2:
							return map[string]bool{"app-id-2": true, "app-id-3": true}, nil
						default:
							return map[string]bool{"app-id-3": true}, nil
						}
					}
				})

				It("stops polling removed apps", func() {
					Eventually(appCollector.StartCallCount).Should(Equal(3))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

					fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
					Consistently(appCollector.StartCallCount).Should(Equal(3))
					Eventually(appCollector.StopCallCount).Should(Equal(1))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-2", "app-id-3"))

					fclock.Increment(TestRefreshInterval)
					Consistently(appCollector.StartCallCount).Should(Equal(3))
					Eventually(appCollector.StopCallCount).Should(Equal(2))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-3"))
				})
			})

			Context("when refresh has both new apps and app removals", func() {
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

				It("polls the new apps and stops polling removed apps", func() {
					Eventually(appCollector.StartCallCount).Should(Equal(2))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1", "app-id-3"))

					fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
					Eventually(appCollector.StartCallCount).Should(Equal(3))
					Eventually(appCollector.StopCallCount).Should(Equal(1))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-2", "app-id-3"))

					fclock.Increment(TestRefreshInterval)
					Eventually(appCollector.StartCallCount).Should(Equal(4))
					Eventually(appCollector.StopCallCount).Should(Equal(2))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1", "app-id-2"))
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
					It("polls the app shard 0", func() {
						Eventually(appCollector.StartCallCount).Should(Equal(2))
						Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-3", "app-id-4"))

						fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
						Consistently(appCollector.StartCallCount).Should(Equal(2))
						Consistently(appCollector.StopCallCount).Should(Equal(0))
						Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-3", "app-id-4"))

						fclock.Increment(TestRefreshInterval)
						Eventually(appCollector.StartCallCount).Should(Equal(3))
						Eventually(appCollector.StopCallCount).Should(Equal(2))
						Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-8"))
					})
				})
				Context("when current index is 1", func() {
					BeforeEach(func() {
						nodeIndex = 1
					})
					It("polls app shard 1", func() {
						Consistently(appCollector.StartCallCount).Should(Equal(0))

						fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
						Eventually(appCollector.StartCallCount).Should(Equal(2))
						Consistently(appCollector.StopCallCount).Should(Equal(0))
						Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-5", "app-id-6"))

						fclock.Increment(TestRefreshInterval)
						Consistently(appCollector.StartCallCount).Should(Equal(2))
						Consistently(appCollector.StopCallCount).Should(Equal(0))
						Consistently(coll.GetCollectorAppIds).Should(ConsistOf("app-id-5", "app-id-6"))

					})
				})
				Context("when current index is 2", func() {
					BeforeEach(func() {
						nodeIndex = 2
					})
					It("polls app shard 2", func() {
						Eventually(appCollector.StartCallCount).Should(Equal(2))
						Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1", "app-id-2"))

						fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
						Consistently(appCollector.StartCallCount).Should(Equal(2))
						Eventually(appCollector.StopCallCount).Should(Equal(2))
						Eventually(coll.GetCollectorAppIds).Should(BeEmpty())

						fclock.Increment(TestRefreshInterval)
						Eventually(appCollector.StartCallCount).Should(Equal(3))
						Consistently(appCollector.StopCallCount).Should(Equal(2))
						Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-7"))

					})
				})

			})
		})

		Context("when getting apps from policy policyDb fails", func() {
			BeforeEach(func() {
				policyDb.GetAppIdsReturns(nil, errors.New("test collector error"))
			})

			It("does not poll and logs the error", func() {
				Eventually(buffer).Should(gbytes.Say("test collector error"))
				Consistently(coll.GetCollectorAppIds).Should(BeEmpty())

				fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
				Eventually(policyDb.GetAppIdsCallCount).Should(Equal(2))
				Eventually(buffer).Should(gbytes.Say("test collector error"))
				Consistently(coll.GetCollectorAppIds).Should(BeEmpty())
			})

		})

	})

	Describe("Stop", func() {
		BeforeEach(func() {
			policyDb.GetAppIdsReturns(map[string]bool{"app-id-1": true, "app-id-2": true, "app-id-3": true}, nil)
			coll = NewCollector(TestRefreshInterval, TestCollectInterval, persistMetrics, TestSaveInterval, nodeIndex, nodeNum, logger, policyDb, instanceMetricsDb, fclock, createAppCollector)
			coll.Start()
		})

		It("stops the collecting", func() {
			Eventually(policyDb.GetAppIdsCallCount).Should(Equal(1))
			fclock.WaitForWatcherAndIncrement(TestRefreshInterval)
			Eventually(policyDb.GetAppIdsCallCount).Should(Equal(2))

			coll.Stop()
			Eventually(appCollector.StopCallCount).Should(Equal(3))

			fclock.Increment(TestRefreshInterval)
			Consistently(policyDb.GetAppIdsCallCount).Should(Equal(2))
		})
	})

	Describe("QueryMetricsFromCache", func() {
		JustBeforeEach(func() {
			coll = NewCollector(TestRefreshInterval, TestCollectInterval, persistMetrics, TestSaveInterval, nodeIndex, nodeNum, logger, policyDb, instanceMetricsDb, fclock, createAppCollector)
			coll.Start()
		})
		BeforeEach(func() {
			policyDb.GetAppIdsReturns(map[string]bool{"an-app-id": true}, nil)
			metric1 = &models.AppInstanceMetric{
				AppId:         "an-app-id",
				InstanceIndex: 1,
				CollectedAt:   2222,
				Name:          models.MetricNameThroughput,
				Unit:          models.UnitRPS,
				Value:         "3",
				Timestamp:     2222,
			}
			metric2 = &models.AppInstanceMetric{
				AppId:         "an-app-id",
				InstanceIndex: 1,
				CollectedAt:   3333,
				Name:          models.MetricNameResponseTime,
				Unit:          models.UnitMilliseconds,
				Value:         "300",
				Timestamp:     3333,
			}
			metric3 = &models.AppInstanceMetric{
				AppId:         "another-app-id",
				InstanceIndex: 1,
				CollectedAt:   1111,
				Name:          models.MetricNameThroughput,
				Unit:          models.UnitRPS,
				Value:         "3",
				Timestamp:     1111,
			}

			appCollector.QueryReturns([]collection.TSD{metric1, metric2}, true)
		})

		Context("when cache hits", func() {
			It("returns the data in cache", func() {
				Eventually(policyDb.GetAppIdsCallCount).Should(Equal(1))

				data, err := coll.QueryMetrics("an-app-id", -1, "test-metric-type", 0, 5555, db.ASC)
				Expect(err).To(BeNil())
				Expect(data).To(Equal([]*models.AppInstanceMetric{metric1, metric2}))

				data, err = coll.QueryMetrics("an-app-id", -1, "test-metric-type", 0, 5555, db.DESC)
				Expect(err).To(BeNil())
				Expect(data).To(Equal([]*models.AppInstanceMetric{metric2, metric1}))
			})
		})
		Context("when cache potenailly misses", func() {
			BeforeEach(func() {
				appCollector.QueryReturns([]collection.TSD{metric1}, false)
			})

			Context("when metrics data are not persisted", func() {
				It("returns metrics cached", func() {
					Eventually(policyDb.GetAppIdsCallCount).Should(Equal(1))
					data, err := coll.QueryMetrics("an-app-id", -1, "test-metric-type", 0, 5555, db.ASC)
					Expect(err).NotTo(HaveOccurred())
					Expect(data).To(Equal([]*models.AppInstanceMetric{metric1}))
				})
			})
			Context("when metrics data are persisted", func() {
				BeforeEach(func() {
					persistMetrics = true
				})
				Context("when retrieving metrics from database succeeds", func() {
					BeforeEach(func() {
						instanceMetricsDb.RetrieveInstanceMetricsReturns([]*models.AppInstanceMetric{metric1, metric2}, nil)
					})
					It("returns metrics from database", func() {
						Eventually(policyDb.GetAppIdsCallCount).Should(Equal(1))
						data, err := coll.QueryMetrics("an-app-id", -1, "test-metric-type", 0, 5555, db.ASC)
						Expect(err).NotTo(HaveOccurred())
						Expect(data).To(Equal([]*models.AppInstanceMetric{metric1, metric2}))
					})
				})

				Context("when retrieving metrics from database has error", func() {
					BeforeEach(func() {
						instanceMetricsDb.RetrieveInstanceMetricsReturns(nil, errors.New("an error"))
					})
					It("returns error", func() {
						Eventually(policyDb.GetAppIdsCallCount).Should(Equal(1))
						_, err := coll.QueryMetrics("an-app-id", -1, "test-metric-type", 0, 5555, db.ASC)
						Expect(err).To(MatchError("an error"))
					})

				})
			})

		})
		Context("when app instance metrics are not being collected now", func() {
			Context("when metric data are not persisted", func() {
				It("return empty result", func() {
					data, err := coll.QueryMetrics("another-app-id", -1, "test-metric-type", 0, 5555, db.ASC)
					Expect(err).NotTo(HaveOccurred())
					Expect(data).To(BeEmpty())
				})
			})

			Context("when metric data are persisted", func() {
				BeforeEach(func() {
					persistMetrics = true
					instanceMetricsDb.RetrieveInstanceMetricsReturns([]*models.AppInstanceMetric{metric3}, nil)
				})
				It("retrieves from database", func() {
					data, err := coll.QueryMetrics("non-exist-app-id", -1, "test-metric-type", 0, 5555, db.ASC)
					Expect(err).NotTo(HaveOccurred())
					Expect(data).To(Equal([]*models.AppInstanceMetric{metric3}))
				})
			})
		})

		AfterEach(func() {
			coll.Stop()
		})
	})

})
