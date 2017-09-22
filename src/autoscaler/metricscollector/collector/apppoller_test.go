package collector_test

import (
	"autoscaler/cf"
	. "autoscaler/metricscollector/collector"
	"autoscaler/metricscollector/fakes"
	"autoscaler/models"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"
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
		cfc       *fakes.FakeCfClient
		noaa      *fakes.FakeNoaaConsumer
		database  *fakes.FakeInstanceMetricsDB
		poller    AppCollector
		fclock    *fakeclock.FakeClock
		buffer    *gbytes.Buffer
		timestamp int64
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		noaa = &fakes.FakeNoaaConsumer{}
		database = &fakes.FakeInstanceMetricsDB{}

		logger := lagertest.NewTestLogger("apppoller-test")
		buffer = logger.Buffer()

		fclock = fakeclock.NewFakeClock(time.Now())
		poller = NewAppPoller(logger, "test-app-id", TestCollectInterval, cfc, noaa, database, fclock)
		timestamp = 111111
	})

	Describe("Start", func() {

		JustBeforeEach(func() {
			poller.Start()
		})

		AfterEach(func() {
			poller.Stop()
		})

		It("polls metrics with the given interval", func() {
			Eventually(noaa.ContainerEnvelopesCallCount).Should(Equal(1))

			fclock.WaitForWatcherAndIncrement(TestCollectInterval)
			Eventually(noaa.ContainerEnvelopesCallCount).Should(Equal(2))

			fclock.WaitForWatcherAndIncrement(TestCollectInterval)
			Eventually(noaa.ContainerEnvelopesCallCount).Should(Equal(3))
		})

		Context("when retrieving container metrics all succeeds", func() {

			BeforeEach(func() {
				cfc.GetTokensReturns(cf.Tokens{AccessToken: "test-access-token"})
			})

			Context("when container envelopes are not empty", func() {

				BeforeEach(func() {
					noaa.ContainerEnvelopesStub = func(appid string, token string) ([]*events.Envelope, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("Bearer test-access-token"))
						return []*events.Envelope{
							&events.Envelope{
								ContainerMetric: &events.ContainerMetric{
									ApplicationId:    proto.String("test-app-id"),
									InstanceIndex:    proto.Int32(0),
									MemoryBytes:      proto.Uint64(100000000),
									MemoryBytesQuota: proto.Uint64(300000000),
									CpuPercentage:    proto.Float64(50.0),
								},
								Timestamp: &timestamp,
							},
						}, nil
					}
				})

				It("saves the metrics to database", func() {
					Eventually(database.SaveMetricCallCount).Should(Equal(3))
					Expect(database.SaveMetricArgsForCall(0)).To(Equal(&models.AppInstanceMetric{
						AppId:         "test-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameMemoryUsed,
						Unit:          models.UnitMegaBytes,
						Value:         "95",
						Timestamp:     111111,
					}))
					Expect(database.SaveMetricArgsForCall(1)).To(Equal(&models.AppInstanceMetric{
						AppId:         "test-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameMemoryUtil,
						Unit:          models.UnitPercentage,
						Value:         "33",
						Timestamp:     111111,
					}))
					Expect(database.SaveMetricArgsForCall(2)).To(Equal(&models.AppInstanceMetric{
						AppId:         "test-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameCpuPercentage,
						Unit:          models.UnitPercentage,
						Value:         "50",
						Timestamp:     111111,
					}))

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					Eventually(database.SaveMetricCallCount).Should(Equal(6))

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					Eventually(database.SaveMetricCallCount).Should(Equal(9))
				})
			})

			Context("when container envelopes are empty", func() {
				BeforeEach(func() {
					noaa.ContainerEnvelopesStub = func(appid string, token string) ([]*events.Envelope, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("Bearer test-access-token"))

						return []*events.Envelope{}, nil
					}
				})

				It("saves nothing to database", func() {
					Consistently(database.SaveMetricCallCount).Should(BeZero())

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					Consistently(database.SaveMetricCallCount).Should(BeZero())

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					Consistently(database.SaveMetricCallCount).Should(BeZero())
				})

			})

			Context("when container envelopes are partially empty", func() {
				BeforeEach(func() {
					noaa.ContainerEnvelopesStub = func(appid string, token string) ([]*events.Envelope, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("Bearer test-access-token"))

						if noaa.ContainerEnvelopesCallCount()%2 == 0 {
							return []*events.Envelope{}, nil
						} else {
							return []*events.Envelope{
								&events.Envelope{
									ContainerMetric: &events.ContainerMetric{
										ApplicationId:    proto.String("test-app-id"),
										InstanceIndex:    proto.Int32(0),
										MemoryBytes:      proto.Uint64(100000000),
										MemoryBytesQuota: proto.Uint64(300000000),
									},
									Timestamp: &timestamp,
								},
							}, nil
						}
					}
				})

				It("saves metrics in non-empty container envelops to database", func() {
					Eventually(database.SaveMetricCallCount).Should(Equal(2))
					Expect(database.SaveMetricArgsForCall(0)).To(Equal(&models.AppInstanceMetric{
						AppId:         "test-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameMemoryUsed,
						Unit:          models.UnitMegaBytes,
						Value:         "95",
						Timestamp:     111111,
					}))
					Expect(database.SaveMetricArgsForCall(1)).To(Equal(&models.AppInstanceMetric{
						AppId:         "test-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameMemoryUtil,
						Unit:          models.UnitPercentage,
						Value:         "33",
						Timestamp:     111111,
					}))

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					Consistently(database.SaveMetricCallCount).Should(Equal(2))

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					Eventually(database.SaveMetricCallCount).Should(Equal(4))
				})
			})
		})

		Context("when retrieving container envelopes all fails", func() {

			BeforeEach(func() {
				noaa.ContainerEnvelopesReturns(nil, errors.New("test apppoller error"))
			})

			It("saves nothing to database and logs the errors", func() {
				Eventually(buffer).Should(gbytes.Say("poll-metric-from-noaa"))
				Eventually(buffer).Should(gbytes.Say("test apppoller error"))
				Consistently(database.SaveMetricCallCount).Should(BeZero())

				fclock.WaitForWatcherAndIncrement(TestCollectInterval)
				Eventually(buffer).Should(gbytes.Say("poll-metric-from-noaa"))
				Eventually(buffer).Should(gbytes.Say("test apppoller error"))
				Consistently(database.SaveMetricCallCount).Should(BeZero())
			})
		})

		Context("when retrieving container envelopes fail, the metrics collector retries", func() {
			BeforeEach(func() {
				cfc.GetTokensReturns(cf.Tokens{AccessToken: "test-access-token"})

				noaa.ContainerEnvelopesStub = func(appid string, token string) ([]*events.Envelope, error) {
					Expect(appid).To(Equal("test-app-id"))
					Expect(token).To(Equal("Bearer test-access-token"))

					if noaa.ContainerEnvelopesCallCount() < 3 {
						return nil, errors.New("apppoller test error")
					} else {
						return []*events.Envelope{
							&events.Envelope{
								ContainerMetric: &events.ContainerMetric{
									ApplicationId:    proto.String("test-app-id"),
									InstanceIndex:    proto.Int32(0),
									MemoryBytes:      proto.Uint64(100000000),
									MemoryBytesQuota: proto.Uint64(300000000),
								},
							},
						}, nil
					}
				}

			})

			It("polls container envelopes successfully; logs the retries and errors", func() {
				Eventually(buffer).Should(gbytes.Say("poll-metric-from-noaa-retry"))

				Eventually(buffer).Should(gbytes.Say("poll-metric-from-noaa-retry"))

				Eventually(buffer).Should(gbytes.Say("poll-metric-get-memory-metric"))
				Eventually(noaa.ContainerEnvelopesCallCount).Should(Equal(3))

			})
		})

	})

	Describe("Stop", func() {
		BeforeEach(func() {
			poller.Start()
		})

		It("stops the polling", func() {
			Eventually(noaa.ContainerEnvelopesCallCount).Should(Equal(1))

			fclock.WaitForWatcherAndIncrement(TestCollectInterval)
			Eventually(noaa.ContainerEnvelopesCallCount).Should(Equal(2))

			poller.Stop()
			fclock.Increment(TestCollectInterval)
			Consistently(noaa.ContainerEnvelopesCallCount).Should(Equal(2))
		})
	})

})
