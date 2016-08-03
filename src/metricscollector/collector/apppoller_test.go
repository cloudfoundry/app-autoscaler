package collector_test

import (
	"metricscollector/cf"
	. "metricscollector/collector"
	"metricscollector/collector/fakes"
	"metricscollector/metrics"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"errors"
	"time"
)

var _ = Describe("Apppoller", func() {

	var (
		cfc      *fakes.FakeCfClient
		noaa     *fakes.FakeNoaaConsumer
		database *fakes.FakeDB
		poller   *AppPoller
		fclock   *fakeclock.FakeClock
		buffer   *gbytes.Buffer
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		noaa = &fakes.FakeNoaaConsumer{}
		database = &fakes.FakeDB{}

		logger := lager.NewLogger("apppoller-test")
		buffer = gbytes.NewBuffer()
		logger.RegisterSink(lager.NewWriterSink(buffer, lager.ERROR))

		fclock = fakeclock.NewFakeClock(time.Now())
		poller = NewAppPoller("test-app-id", TestPollInterval, logger, cfc, noaa, database, fclock)
	})

	Describe("Start", func() {

		JustBeforeEach(func() {
			poller.Start()
		})

		AfterEach(func() {
			poller.Stop()
		})

		BeforeEach(func() {
			cfc.GetTokensReturns(cf.Tokens{AccessToken: "test-access-token"})

			database.SaveMetricStub = func(metric *metrics.Metric) error {
				Expect(metric.AppId).To(Equal("test-app-id"))
				Expect(metric.Name).To(Equal(metrics.MetricNameMemory))
				Expect(metric.Unit).To(Equal(metrics.UnitBytes))
				Expect(metric.Instances).To(ConsistOf(metrics.InstanceMetric{Index: 0, Value: "1234"}))
				return nil
			}

		})

		Context("when retrieving container metrics all succeeds", func() {

			Context("when all container metrics are not empty", func() {

				BeforeEach(func() {
					noaa.ContainerMetricsStub = func(appid string, token string) ([]*events.ContainerMetric, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("bearer test-access-token"))

						return []*events.ContainerMetric{
							&events.ContainerMetric{
								ApplicationId: proto.String("test-app-id"),
								InstanceIndex: proto.Int32(0),
								MemoryBytes:   proto.Uint64(1234),
							},
						}, nil
					}
				})

				It("saves all the metrics to database", func() {
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(database.SaveMetricCallCount).Should(Equal(1))

					fclock.Increment(TestPollInterval * time.Second)
					Eventually(database.SaveMetricCallCount).Should(Equal(2))

					fclock.Increment(TestPollInterval * time.Second)
					Eventually(database.SaveMetricCallCount).Should(Equal(3))
				})
			})

			Context("when all container metrics are empty", func() {
				BeforeEach(func() {
					noaa.ContainerMetricsStub = func(appid string, token string) ([]*events.ContainerMetric, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("bearer test-access-token"))

						return []*events.ContainerMetric{}, nil
					}
				})

				It("saves nothing to database", func() {
					fclock.Increment(TestPollInterval * time.Second)
					Consistently(database.SaveMetricCallCount).Should(BeZero())

					fclock.Increment(TestPollInterval * time.Second)
					Consistently(database.SaveMetricCallCount).Should(BeZero())

					fclock.Increment(TestPollInterval * time.Second)
					Consistently(database.SaveMetricCallCount).Should(BeZero())
				})

			})

			Context("when container metrics are partially empty", func() {
				BeforeEach(func() {
					noaa.ContainerMetricsStub = func(appid string, token string) ([]*events.ContainerMetric, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("bearer test-access-token"))

						if noaa.ContainerMetricsCallCount()%2 == 0 {
							return []*events.ContainerMetric{}, nil
						} else {
							return []*events.ContainerMetric{
								&events.ContainerMetric{
									ApplicationId: proto.String("test-app-id"),
									InstanceIndex: proto.Int32(0),
									MemoryBytes:   proto.Uint64(1234),
								},
							}, nil
						}

					}
				})

				It("saves non-empty metrics to database", func() {
					fclock.Increment(TestPollInterval * time.Second)
					Eventually(database.SaveMetricCallCount).Should(Equal(1))

					fclock.Increment(TestPollInterval * time.Second)
					Consistently(database.SaveMetricCallCount).Should(Equal(1))

					fclock.Increment(TestPollInterval * time.Second)
					Eventually(database.SaveMetricCallCount).Should(Equal(2))
				})
			})
		})

		Context("when retrieving container metrics all fails", func() {

			BeforeEach(func() {
				noaa.ContainerMetricsReturns(nil, errors.New("test apppoller error"))
			})

			It("saves nothing to database and logs the errors", func() {
				fclock.Increment(TestPollInterval * time.Second)
				Consistently(database.SaveMetricCallCount).Should(BeZero())
				Eventually(buffer).Should(gbytes.Say("test apppoller error"))

				fclock.Increment(TestPollInterval * time.Second)
				Consistently(database.SaveMetricCallCount).Should(BeZero())
				Eventually(buffer).Should(gbytes.Say("test apppoller error"))

			})
		})

		Context("when retrieving container metrics partially fails", func() {
			BeforeEach(func() {
				noaa.ContainerMetricsStub = func(appid string, token string) ([]*events.ContainerMetric, error) {
					Expect(appid).To(Equal("test-app-id"))
					Expect(token).To(Equal("bearer test-access-token"))

					if noaa.ContainerMetricsCallCount()%2 == 0 {
						return nil, errors.New("apppoller test error")
					} else {
						return []*events.ContainerMetric{
							&events.ContainerMetric{
								ApplicationId: proto.String("test-app-id"),
								InstanceIndex: proto.Int32(0),
								MemoryBytes:   proto.Uint64(1234),
							},
						}, nil
					}
				}
			})

			It("saves successful results to database and logs the errors", func() {
				fclock.Increment(TestPollInterval * time.Second)
				Eventually(database.SaveMetricCallCount).Should(Equal(1))

				fclock.Increment(TestPollInterval * time.Second)
				Consistently(database.SaveMetricCallCount).Should(Equal(1))
				Eventually(buffer).Should(gbytes.Say("apppoller test error"))

				fclock.Increment(TestPollInterval * time.Second)
				Eventually(database.SaveMetricCallCount).Should(Equal(2))

			})

		})

	})

	Describe("Stop", func() {
		BeforeEach(func() {
			poller.Start()
		})

		It("stops the polling", func() {
			fclock.Increment(TestPollInterval * time.Second)
			Eventually(noaa.ContainerMetricsCallCount).Should(Equal(1))

			fclock.Increment(TestPollInterval * time.Second)
			Eventually(noaa.ContainerMetricsCallCount).Should(Equal(2))

			poller.Stop()
			fclock.Increment(TestPollInterval * time.Second)
			Consistently(noaa.ContainerMetricsCallCount).Should(Equal(2))
		})
	})

})
