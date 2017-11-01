package collector_test

import (
	. "autoscaler/metricscollector/collector"
	"autoscaler/metricscollector/fakes"
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
		policyDb          *fakes.FakePolicyDB
		instanceMetricsDb *fakes.FakeInstanceMetricsDB
		coll              *Collector
		fclock            *fakeclock.FakeClock
		appCollector      *fakes.FakeAppCollector
		buffer            *gbytes.Buffer
	)

	BeforeEach(func() {
		policyDb = &fakes.FakePolicyDB{}
		instanceMetricsDb = &fakes.FakeInstanceMetricsDB{}

		logger := lagertest.NewTestLogger("collector-test")
		buffer = logger.Buffer()

		fclock = fakeclock.NewFakeClock(time.Now())
		appCollector = &fakes.FakeAppCollector{}
		createAppCollector := func(appId string, dataChan chan *models.AppInstanceMetric) AppCollector {
			return appCollector
		}
		coll = NewCollector(TestRefreshInterval, TestCollectInterval, TestSaveInterval, logger, policyDb, instanceMetricsDb, fclock, createAppCollector)
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			coll.Start()
		})

		AfterEach(func() {
			coll.Stop()
		})

		It("refreshes the apps with given interval", func() {
			Eventually(policyDb.GetAppIdsCallCount).Should(Equal(1))

			fclock.Increment(TestRefreshInterval)
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

					fclock.Increment(TestRefreshInterval)
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

					fclock.Increment(TestRefreshInterval)
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

					fclock.Increment(TestRefreshInterval)
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

					fclock.Increment(TestRefreshInterval)
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

				It("pools the new apps and stops polling removed apps", func() {
					Eventually(appCollector.StartCallCount).Should(Equal(2))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1", "app-id-3"))

					fclock.Increment(TestRefreshInterval)
					Eventually(appCollector.StartCallCount).Should(Equal(3))
					Eventually(appCollector.StopCallCount).Should(Equal(1))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-2", "app-id-3"))

					fclock.Increment(TestRefreshInterval)
					Eventually(appCollector.StartCallCount).Should(Equal(4))
					Eventually(appCollector.StopCallCount).Should(Equal(2))
					Eventually(coll.GetCollectorAppIds).Should(ConsistOf("app-id-1", "app-id-2"))
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

				fclock.Increment(TestRefreshInterval)
				Eventually(policyDb.GetAppIdsCallCount).Should(Equal(2))
				Eventually(buffer).Should(gbytes.Say("test collector error"))
				Consistently(coll.GetCollectorAppIds).Should(BeEmpty())
			})

		})

	})

	Describe("Stop", func() {
		BeforeEach(func() {
			policyDb.GetAppIdsReturns(map[string]bool{"app-id-1": true, "app-id-2": true, "app-id-3": true}, nil)
			coll.Start()
		})

		It("stops the collecting", func() {

			fclock.Increment(TestRefreshInterval)
			Eventually(policyDb.GetAppIdsCallCount).Should(Equal(2))

			coll.Stop()
			Eventually(appCollector.StopCallCount).Should(Equal(3))

			fclock.Increment(TestRefreshInterval)
			Consistently(policyDb.GetAppIdsCallCount).Should(Equal(2))
		})
	})

})
