package collector_test

import (
	. "autoscaler/metricscollector/collector"
	"autoscaler/metricscollector/fakes"

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
		database *fakes.FakePolicyDB
		coll     *Collector
		fclock   *fakeclock.FakeClock
		poller   *fakes.FakeAppPoller
		buffer   *gbytes.Buffer
	)

	BeforeEach(func() {
		database = &fakes.FakePolicyDB{}

		logger := lagertest.NewTestLogger("collector-test")
		buffer = logger.Buffer()

		fclock = fakeclock.NewFakeClock(time.Now())
		poller = &fakes.FakeAppPoller{}
		createPoller := func(appId string) AppPoller {
			return poller
		}
		coll = NewCollector(TestRefreshInterval, logger, database, fclock, createPoller)
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			coll.Start()
		})

		AfterEach(func() {
			coll.Stop()
		})

		It("refreshes the apps with given interval", func() {
			Eventually(database.GetAppIdsCallCount).Should(Equal(1))

			fclock.Increment(TestRefreshInterval)
			Eventually(database.GetAppIdsCallCount).Should(Equal(2))

			fclock.Increment(TestRefreshInterval)
			Eventually(database.GetAppIdsCallCount).Should(Equal(3))

		})

		Context("when getting apps from policy database succeeds", func() {

			Context("when no apps in policy database", func() {
				BeforeEach(func() {
					database.GetAppIdsReturns(make(map[string]struct{}), nil)
				})

				It("does nothing", func() {
					Consistently(coll.GetPollerAppIds).Should(BeEmpty())

					fclock.Increment(TestRefreshInterval)
					Consistently(coll.GetPollerAppIds).Should(BeEmpty())
				})
			})

			Context("when refresh does not have app changes", func() {
				BeforeEach(func() {
					database.GetAppIdsReturns(map[string]struct{}{"app-id-1": struct{}{}, "app-id-2": struct{}{}, "app-id-3": struct{}{}}, nil)
				})

				It("should always poll the same set of apps", func() {
					Eventually(poller.StartCallCount).Should(Equal(3))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

					fclock.Increment(TestRefreshInterval)
					Consistently(poller.StartCallCount).Should(Equal(3))
					Consistently(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))
				})
			})

			Context("when refresh has new apps", func() {
				BeforeEach(func() {
					database.GetAppIdsStub = func() (map[string]struct{}, error) {
						switch database.GetAppIdsCallCount() {
						case 1:
							return map[string]struct{}{"app-id-1": struct{}{}}, nil
						case 2:
							return map[string]struct{}{"app-id-1": struct{}{}, "app-id-2": struct{}{}}, nil
						default:
							return map[string]struct{}{"app-id-1": struct{}{}, "app-id-2": struct{}{}, "app-id-3": struct{}{}}, nil
						}
					}
				})

				It("polls newly added ones", func() {
					Eventually(poller.StartCallCount).Should(Equal(1))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1"))

					fclock.Increment(TestRefreshInterval)
					Eventually(poller.StartCallCount).Should(Equal(2))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2"))

					fclock.Increment(TestRefreshInterval)
					Eventually(poller.StartCallCount).Should(Equal(3))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

				})

			})

			Context("when refresh has app removals", func() {
				BeforeEach(func() {
					database.GetAppIdsStub = func() (map[string]struct{}, error) {
						switch database.GetAppIdsCallCount() {
						case 1:
							return map[string]struct{}{"app-id-1": struct{}{}, "app-id-2": struct{}{}, "app-id-3": struct{}{}}, nil
						case 2:
							return map[string]struct{}{"app-id-2": struct{}{}, "app-id-3": struct{}{}}, nil
						default:
							return map[string]struct{}{"app-id-3": struct{}{}}, nil
						}
					}
				})

				It("stops polling removed apps", func() {
					Eventually(poller.StartCallCount).Should(Equal(3))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

					fclock.Increment(TestRefreshInterval)
					Consistently(poller.StartCallCount).Should(Equal(3))
					Eventually(poller.StopCallCount).Should(Equal(1))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-2", "app-id-3"))

					fclock.Increment(TestRefreshInterval)
					Consistently(poller.StartCallCount).Should(Equal(3))
					Eventually(poller.StopCallCount).Should(Equal(2))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-3"))
				})
			})

			Context("when refresh has both new apps and app removals", func() {
				BeforeEach(func() {
					database.GetAppIdsStub = func() (map[string]struct{}, error) {
						switch database.GetAppIdsCallCount() {
						case 1:
							return map[string]struct{}{"app-id-1": struct{}{}, "app-id-3": struct{}{}}, nil
						case 2:
							return map[string]struct{}{"app-id-2": struct{}{}, "app-id-3": struct{}{}}, nil
						default:
							return map[string]struct{}{"app-id-1": struct{}{}, "app-id-2": struct{}{}}, nil
						}
					}
				})

				It("pools the new apps and stops polling removed apps", func() {
					Eventually(poller.StartCallCount).Should(Equal(2))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-3"))

					fclock.Increment(TestRefreshInterval)
					Eventually(poller.StartCallCount).Should(Equal(3))
					Eventually(poller.StopCallCount).Should(Equal(1))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-2", "app-id-3"))

					fclock.Increment(TestRefreshInterval)
					Eventually(poller.StartCallCount).Should(Equal(4))
					Eventually(poller.StopCallCount).Should(Equal(2))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2"))
				})

			})
		})

		Context("when getting apps from policy database fails", func() {
			BeforeEach(func() {
				database.GetAppIdsReturns(nil, errors.New("test collector error"))
			})

			It("does not poll and logs the error", func() {
				Eventually(buffer).Should(gbytes.Say("test collector error"))
				Consistently(coll.GetPollerAppIds).Should(BeEmpty())

				fclock.Increment(TestRefreshInterval)
				Eventually(database.GetAppIdsCallCount).Should(Equal(2))
				Eventually(buffer).Should(gbytes.Say("test collector error"))
				Consistently(coll.GetPollerAppIds).Should(BeEmpty())
			})

		})

	})

	Describe("Stop", func() {
		BeforeEach(func() {
			database.GetAppIdsReturns(map[string]struct{}{"app-id-1": struct{}{}, "app-id-2": struct{}{}, "app-id-3": struct{}{}}, nil)
			coll.Start()
		})

		It("stops the collecting", func() {

			fclock.Increment(TestRefreshInterval)
			Eventually(database.GetAppIdsCallCount).Should(Equal(2))

			coll.Stop()
			Eventually(poller.StopCallCount).Should(Equal(3))

			fclock.Increment(TestRefreshInterval)
			Consistently(database.GetAppIdsCallCount).Should(Equal(2))
		})
	})

})
