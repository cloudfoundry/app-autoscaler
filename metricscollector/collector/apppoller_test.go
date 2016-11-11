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
	"github.com/onsi/gomega/gstruct"

	"errors"
	"time"
)

var _ = Describe("Apppoller", func() {

	var (
		cfc       *fakes.FakeCfClient
		noaa      *fakes.FakeNoaaConsumer
		database  *fakes.FakeInstanceMetricsDB
		poller    AppPoller
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
		poller = NewAppPoller(logger, "test-app-id", TestPollInterval, cfc, noaa, database, fclock)
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

			fclock.Increment(TestPollInterval)
			Eventually(noaa.ContainerEnvelopesCallCount).Should(Equal(2))

			fclock.Increment(TestPollInterval)
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
						Expect(token).To(Equal("bearer test-access-token"))
						return []*events.Envelope{
							&events.Envelope{
								ContainerMetric: &events.ContainerMetric{
									ApplicationId: proto.String("test-app-id"),
									InstanceIndex: proto.Int32(0),
									MemoryBytes:   proto.Uint64(1234),
								},
								Timestamp: &timestamp,
							},
						}, nil
					}

					database.SaveMetricStub = func(metric *models.AppInstanceMetric) error {
						Expect(*metric).To(gstruct.MatchAllFields(gstruct.Fields{
							"AppId":         Equal("test-app-id"),
							"InstanceIndex": BeEquivalentTo(0),
							"CollectedAt":   Equal(fclock.Now().UnixNano()),
							"Name":          Equal(models.MetricNameMemory),
							"Unit":          Equal(models.UnitBytes),
							"Value":         Equal("1234"),
							"Timestamp":     BeEquivalentTo(111111),
						}))
						return nil
					}

				})

				It("saves the metrics to database", func() {
					Eventually(database.SaveMetricCallCount).Should(Equal(1))

					fclock.Increment(TestPollInterval)
					Eventually(database.SaveMetricCallCount).Should(Equal(2))

					fclock.Increment(TestPollInterval)
					Eventually(database.SaveMetricCallCount).Should(Equal(3))
				})
			})

			Context("when container envelopes are empty", func() {
				BeforeEach(func() {
					noaa.ContainerEnvelopesStub = func(appid string, token string) ([]*events.Envelope, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("bearer test-access-token"))

						return []*events.Envelope{}, nil
					}
				})

				It("saves nothing to database", func() {
					Consistently(database.SaveMetricCallCount).Should(BeZero())

					fclock.Increment(TestPollInterval)
					Consistently(database.SaveMetricCallCount).Should(BeZero())

					fclock.Increment(TestPollInterval)
					Consistently(database.SaveMetricCallCount).Should(BeZero())
				})

			})

			Context("when container envelopes are partially empty", func() {
				BeforeEach(func() {
					noaa.ContainerEnvelopesStub = func(appid string, token string) ([]*events.Envelope, error) {
						Expect(appid).To(Equal("test-app-id"))
						Expect(token).To(Equal("bearer test-access-token"))

						if noaa.ContainerEnvelopesCallCount()%2 == 0 {
							return []*events.Envelope{}, nil
						} else {
							return []*events.Envelope{
								&events.Envelope{
									ContainerMetric: &events.ContainerMetric{
										ApplicationId: proto.String("test-app-id"),
										InstanceIndex: proto.Int32(0),
										MemoryBytes:   proto.Uint64(1234),
									},
									Timestamp: &timestamp,
								},
							}, nil
						}
					}

					database.SaveMetricStub = func(metric *models.AppInstanceMetric) error {
						Expect(*metric).To(gstruct.MatchAllFields(gstruct.Fields{
							"AppId":         Equal("test-app-id"),
							"InstanceIndex": BeEquivalentTo(0),
							"CollectedAt":   Equal(fclock.Now().UnixNano()),
							"Name":          Equal(models.MetricNameMemory),
							"Unit":          Equal(models.UnitBytes),
							"Value":         Equal("1234"),
							"Timestamp":     BeEquivalentTo(111111),
						}))
						return nil
					}

				})

				It("saves metrics in non-empty container envelops to database", func() {
					Eventually(database.SaveMetricCallCount).Should(Equal(1))

					fclock.Increment(TestPollInterval)
					Consistently(database.SaveMetricCallCount).Should(Equal(1))

					fclock.Increment(TestPollInterval)
					Eventually(database.SaveMetricCallCount).Should(Equal(2))
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

				fclock.Increment(TestPollInterval)
				Eventually(buffer).Should(gbytes.Say("poll-metric-from-noaa"))
				Eventually(buffer).Should(gbytes.Say("test apppoller error"))
				Consistently(database.SaveMetricCallCount).Should(BeZero())
			})
		})

		Context("when retrieving container envelopes partially fails", func() {
			BeforeEach(func() {
				cfc.GetTokensReturns(cf.Tokens{AccessToken: "test-access-token"})

				noaa.ContainerEnvelopesStub = func(appid string, token string) ([]*events.Envelope, error) {
					Expect(appid).To(Equal("test-app-id"))
					Expect(token).To(Equal("bearer test-access-token"))

					if noaa.ContainerEnvelopesCallCount()%2 == 0 {
						return nil, errors.New("apppoller test error")
					} else {
						return []*events.Envelope{
							&events.Envelope{
								ContainerMetric: &events.ContainerMetric{
									ApplicationId: proto.String("test-app-id"),
									InstanceIndex: proto.Int32(0),
									MemoryBytes:   proto.Uint64(1234),
								},
								Timestamp: &timestamp,
							},
						}, nil
					}
				}

				database.SaveMetricStub = func(metric *models.AppInstanceMetric) error {
					Expect(*metric).To(gstruct.MatchAllFields(gstruct.Fields{
						"AppId":         Equal("test-app-id"),
						"InstanceIndex": BeEquivalentTo(0),
						"CollectedAt":   Equal(fclock.Now().UnixNano()),
						"Name":          Equal(models.MetricNameMemory),
						"Unit":          Equal(models.UnitBytes),
						"Value":         Equal("1234"),
						"Timestamp":     BeEquivalentTo(111111),
					}))
					return nil
				}

			})

			It("saves successful results to database and logs the errors", func() {
				Eventually(database.SaveMetricCallCount).Should(Equal(1))

				fclock.Increment(TestPollInterval)
				Eventually(buffer).Should(gbytes.Say("poll-metric-from-noaa"))
				Eventually(buffer).Should(gbytes.Say("apppoller test error"))
				Consistently(database.SaveMetricCallCount).Should(Equal(1))

				fclock.Increment(TestPollInterval)
				Eventually(database.SaveMetricCallCount).Should(Equal(2))

			})
		})

	})

	Describe("Stop", func() {
		BeforeEach(func() {
			poller.Start()
		})

		It("stops the polling", func() {
			Eventually(noaa.ContainerEnvelopesCallCount).Should(Equal(1))

			fclock.Increment(TestPollInterval)
			Eventually(noaa.ContainerEnvelopesCallCount).Should(Equal(2))

			poller.Stop()
			fclock.Increment(TestPollInterval)
			Consistently(noaa.ContainerEnvelopesCallCount).Should(Equal(2))
		})
	})

})
