package collector_test

import (
	. "metricscollector/collector"
	"metricscollector/collector/fakes"
	"metricscollector/config"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"errors"
	"time"
)

var _ = Describe("Collector", func() {

	var (
		cfc      *fakes.FakeCfClient
		noaa     *fakes.FakeNoaaConsumer
		database *fakes.FakeDB
		coll     *Collector
		fclock   *fakeclock.FakeClock
		buffer   *gbytes.Buffer
	)

	BeforeEach(func() {

		cfc = &fakes.FakeCfClient{}
		noaa = &fakes.FakeNoaaConsumer{}
		database = &fakes.FakeDB{}

		conf := &config.CollectorConfig{
			PollInterval:    TestPollInterval,
			RefreshInterval: TestRefreshInterval,
		}

		logger := lager.NewLogger("collector-test")
		buffer = gbytes.NewBuffer()
		logger.RegisterSink(lager.NewWriterSink(buffer, lager.ERROR))

		fclock = fakeclock.NewFakeClock(time.Now())
		coll = NewCollector(conf, logger, cfc, noaa, database, fclock)
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			coll.Start()
		})

		AfterEach(func() {
			coll.Stop()
		})

		Context("when getting apps from policy database succeeds", func() {
			BeforeEach(func() {
				noaa.ContainerMetricsStub = func(appId string, token string) ([]*events.ContainerMetric, error) {
					switch appId {
					case "app-id-1":
						pollings[0] = true
					case "app-id-2":
						pollings[1] = true
					case "app-id-3":
						pollings[2] = true
					}
					return []*events.ContainerMetric{}, nil
				}
			})

			Context("when no apps in policy database", func() {
				BeforeEach(func() {
					database.GetAppIdsReturns(make(map[string]bool), nil)
				})

				It("does not poll anything", func() {
					fclock.Increment(TestRefreshInterval * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(1))

					fclock.Increment(TestPollInterval * time.Second)
					Consistently(noaa.ContainerMetricsCallCount).Should(BeZero())

					fclock.Increment((TestRefreshInterval - TestPollInterval) * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(2))

					fclock.Increment(TestPollInterval * time.Second)
					Consistently(noaa.ContainerMetricsCallCount).Should(BeZero())

				})
			})

			Context("when refresh does not have app changes", func() {
				BeforeEach(func() {
					database.GetAppIdsReturns(map[string]bool{"app-id-1": true, "app-id-2": true, "app-id-3": true}, nil)
				})

				It("should always poll the same set of apps", func() {
					fclock.Increment(TestRefreshInterval * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(1))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{true, true, true}))

					fclock.Increment((TestRefreshInterval - TestPollInterval) * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(2))
					Consistently(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{true, true, true}))
				})
			})

			Context("when refresh has new apps", func() {
				BeforeEach(func() {
					database.GetAppIdsStub = func() (map[string]bool, error) {
						switch database.GetAppIdsCallCount() {
						case 1:
							return map[string]bool{"app-id-1": true}, nil
						case 2:
							return map[string]bool{"app-id-1": true, "app-id-2": true}, nil
						default:
							return map[string]bool{"app-id-1": true, "app-id-2": true, "app-id-3": true}, nil
						}
					}
				})

				It("polls newly added ones too", func() {
					fclock.Increment(TestRefreshInterval * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(1))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{true, false, false}))

					fclock.Increment((TestRefreshInterval - TestPollInterval) * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(2))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{true, true, false}))

					fclock.Increment((TestRefreshInterval - TestPollInterval) * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(3))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{true, true, true}))
				})

			})

			Context("when refresh has app removals", func() {
				BeforeEach(func() {
					database.GetAppIdsStub = func() (map[string]bool, error) {
						switch database.GetAppIdsCallCount() {
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
					fclock.Increment(TestRefreshInterval * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(1))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{true, true, true}))

					fclock.Increment((TestRefreshInterval - TestPollInterval) * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(2))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-2", "app-id-3"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{false, true, true}))
					Consistently(pollings).Should(Equal([]bool{false, true, true}))

					fclock.Increment((TestRefreshInterval - TestPollInterval) * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(3))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-3"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{false, false, true}))
					Consistently(pollings).Should(Equal([]bool{false, false, true}))

				})
			})

			Context("when refresh has both new apps and app removals", func() {
				BeforeEach(func() {
					database.GetAppIdsStub = func() (map[string]bool, error) {
						switch database.GetAppIdsCallCount() {
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
					fclock.Increment(TestRefreshInterval * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(1))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-3"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{true, false, true}))

					fclock.Increment((TestRefreshInterval - TestPollInterval) * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(2))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-2", "app-id-3"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{false, true, true}))
					Consistently(pollings).Should(Equal([]bool{false, true, true}))

					fclock.Increment((TestRefreshInterval - TestPollInterval) * time.Second)
					Eventually(database.GetAppIdsCallCount).Should(Equal(3))
					Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2"))

					resetPollings()
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(pollings).Should(Equal([]bool{true, true, false}))
					Consistently(pollings).Should(Equal([]bool{true, true, false}))

				})

			})
		})

		Context("when getting apps from policy database fails", func() {
			BeforeEach(func() {
				database.GetAppIdsReturns(nil, errors.New("test collector error"))
			})

			It("does not poll anything and logs the error", func() {
				fclock.Increment(TestRefreshInterval * time.Second)
				Eventually(database.GetAppIdsCallCount).Should(Equal(1))
				Eventually(buffer).Should(gbytes.Say("test collector error"))

				fclock.Increment(TestPollInterval * time.Second)
				Consistently(noaa.ContainerMetricsCallCount).Should(BeZero())

				fclock.Increment((TestRefreshInterval - TestPollInterval) * time.Second)
				Eventually(database.GetAppIdsCallCount).Should(Equal(2))
				Eventually(buffer).Should(gbytes.Say("test collector error"))

				fclock.Increment(TestPollInterval * time.Second)
				Consistently(noaa.ContainerMetricsCallCount).Should(BeZero())
			})
		})

	})

	Describe("Stop", func() {
		BeforeEach(func() {
			database.GetAppIdsReturns(map[string]bool{"app-id-1": true, "app-id-2": true, "app-id-3": true}, nil)
			coll.Start()
		})

		It("stops the collecting", func() {

			fclock.Increment(TestRefreshInterval * time.Second)
			Eventually(database.GetAppIdsCallCount).Should(Equal(1))
			Eventually(coll.GetPollerAppIds).Should(ConsistOf("app-id-1", "app-id-2", "app-id-3"))

			fclock.Increment(TestPollInterval * time.Second)
			Eventually(noaa.ContainerMetricsCallCount).Should(Equal(3))

			coll.Stop()

			fclock.Increment((TestRefreshInterval - TestPollInterval) * time.Second)
			Consistently(database.GetAppIdsCallCount).Should(Equal(1))
			Consistently(noaa.ContainerMetricsCallCount).Should(Equal(3))

		})
	})

})

var pollings []bool

func resetPollings() {
	pollings = []bool{false, false, false}

}
