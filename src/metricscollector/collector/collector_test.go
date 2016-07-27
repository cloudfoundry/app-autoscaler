package collector_test

import (
	. "metricscollector/collector"
	"metricscollector/collector/fakes"
	"metricscollector/config"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"
	"time"
)

var _ = Describe("Collector", func() {

	var (
		cfc       *fakes.FakeCfClient
		noaa      *fakes.FakeNoaaConsumer
		database  *fakes.FakeDB
		coll      *Collector
		duration  time.Duration
		startTime time.Time
		pollings  *[3][3]bool
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		noaa = &fakes.FakeNoaaConsumer{}
		database = &fakes.FakeDB{}
		logger := lager.NewLogger("collector-test")
		conf := &config.CollectorConfig{RefreshInterval: TestRefreshInterval, PollInterval: TestPollInterval}
		coll = NewCollector(logger, conf, cfc, noaa, database)
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			coll.Start()
			time.Sleep(duration)
			coll.Stop()
		})

		Context("when getting apps from policy database succeeds", func() {
			BeforeEach(func() {
				pollings = &[3][3]bool{}
				startTime = time.Now()

				noaa.ContainerMetricsStub = func(appid string, token string) ([]*events.ContainerMetric, error) {
					computePoolings(appid, startTime, pollings)
					return []*events.ContainerMetric{}, nil
				}
			})

			Context("when no apps in policy database", func() {
				BeforeEach(func() {
					duration = 2500 * time.Millisecond
					database.GetAppIdsReturns(make(map[string]bool), nil)
				})

				It("does not poll anything", func() {
					Expect(noaa.ContainerMetricsCallCount()).To(BeZero())
				})
			})

			Context("when refresh does not have apps change", func() {
				BeforeEach(func() {
					duration = 5500 * time.Millisecond
					database.GetAppIdsReturns(map[string]bool{"app-id-1": true, "app-id-2": true, "app-id-3": true}, nil)
				})

				It("should always poll the same set of apps", func() {
					Expect(*pollings).To(ConsistOf([3]bool{true, true, true}, [3]bool{true, true, true}, [3]bool{true, true, true}))
				})
			})

			Context("when refresh has new app", func() {
				BeforeEach(func() {
					duration = 5500 * time.Millisecond
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

				It("polls newly added one too", func() {
					Expect(*pollings).To(ConsistOf([3]bool{true, true, true}, [3]bool{false, true, true}, [3]bool{false, false, true}))
				})

			})

			Context("when refresh has app removal", func() {
				BeforeEach(func() {
					duration = 5500 * time.Millisecond
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

				It("stops polling removed app", func() {
					Expect(*pollings).To(ConsistOf([3]bool{true, false, false}, [3]bool{true, true, false}, [3]bool{true, true, true}))
				})

			})

			Context("when refresh has both new app and app removal", func() {
				BeforeEach(func() {
					duration = 5500 * time.Millisecond
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

				It("pools the new app and stops polling removed app", func() {
					Expect(*pollings).To(ConsistOf([3]bool{true, false, true}, [3]bool{false, true, true}, [3]bool{true, true, false}))
				})

			})
		})

		Context("when getting apps from policy database fails", func() {
			BeforeEach(func() {
				duration = 1 * time.Second
				database.GetAppIdsReturns(nil, errors.New("an error"))
			})

			It("does not poll anything", func() {
				Expect(noaa.ContainerMetricsCallCount()).To(BeZero())
			})
		})

	})

	Describe("Stop", func() {
		JustBeforeEach(func() {
			coll.Stop()
		})
		Context("when collector is started", func() {
			BeforeEach(func() {
				coll.Start()
				time.Sleep(1 * time.Second)
			})

			It("stops the collecting", func() {
				numNoaa := noaa.ContainerMetricsCallCount()
				numGetAppIds := database.GetAppIdsCallCount()
				time.Sleep(2500 * time.Millisecond)
				Expect(noaa.ContainerMetricsCallCount()).To(Equal(numNoaa))
				Expect(database.GetAppIdsCallCount()).To(Equal(numGetAppIds))
			})
		})

		Context("when collector is not started", func() {
			It("does nothing", func() {
				Expect(noaa.ContainerMetricsCallCount()).To(BeZero())
				Expect(database.GetAppIdsCallCount()).To(BeZero())
			})
		})

	})

})

func computePoolings(appId string, start time.Time, pollings *[3][3]bool) {
	var appIndex int
	switch appId {
	case "app-id-1":
		appIndex = 0
	case "app-id-2":
		appIndex = 1
	case "app-id-3":
		appIndex = 2
	}

	elapsed := time.Since(start)

	if (elapsed > 0) && (elapsed < TestRefreshInterval*time.Second-500*time.Millisecond) {
		(*pollings)[appIndex][0] = true
		return
	}

	if (elapsed > TestRefreshInterval*time.Second+500*time.Millisecond) && (elapsed < 2*TestRefreshInterval*time.Second-500*time.Millisecond) {
		(*pollings)[appIndex][1] = true
		return
	}

	if elapsed > 2*TestRefreshInterval*time.Second+500*time.Millisecond {
		(*pollings)[appIndex][2] = true
	}
}
