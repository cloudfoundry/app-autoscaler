package collector_test

import (
	"autoscaler/cf"
	. "autoscaler/metricscollector/collector"
	"autoscaler/metricscollector/fakes"
	"autoscaler/models"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"errors"
	"time"
)

var _ = Describe("AppStreamer", func() {

	var (
		cfc      *fakes.FakeCfClient
		noaa     *fakes.FakeNoaaConsumer
		database *fakes.FakeInstanceMetricsDB
		streamer AppStreamer
		buffer   *gbytes.Buffer
		msgChan  chan *events.Envelope
		errChan  chan error
		fclock   *fakeclock.FakeClock
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		noaa = &fakes.FakeNoaaConsumer{}
		database = &fakes.FakeInstanceMetricsDB{}

		logger := lagertest.NewTestLogger("AppStreamer-test")
		buffer = logger.Buffer()
		fclock = fakeclock.NewFakeClock(time.Now())

		streamer = NewAppStreamer(logger, "an-app-id", cfc, noaa, database, fclock)

		msgChan = make(chan *events.Envelope)
		errChan = make(chan error, 1)
	})

	Describe("Start", func() {

		JustBeforeEach(func() {
			streamer.Start()
		})

		AfterEach(func() {
			streamer.Stop()
		})

		BeforeEach(func() {
			cfc.GetTokensReturns(cf.Tokens{AccessToken: "test-access-token"})
			noaa.StreamStub = func(appId string, authToken string) (outputChan <-chan *events.Envelope, errorChan <-chan error) {
				Expect(appId).To(Equal("an-app-id"))
				Expect(authToken).To(Equal("test-access-token"))
				return msgChan, errChan
			}
		})

		Context("when there are containermetric events", func() {
			BeforeEach(func() {
				go func() {
					msgChan <- models.NewContainerEnvelope(111111, "an-app-id", 0, 12.8, 12345678, 987654321)
					msgChan <- models.NewContainerEnvelope(222222, "an-app-id", 1, 12.8, 23563212, 987654321)
				}()
			})
			It("Saves metrics to database", func() {
				Eventually(database.SaveMetricCallCount).Should(Equal(2))
				Expect(database.SaveMetricArgsForCall(0)).To(Equal(&models.AppInstanceMetric{
					AppId:         "an-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameMemory,
					Unit:          models.UnitMegaBytes,
					Value:         "12",
					Timestamp:     111111,
				}))
				Expect(database.SaveMetricArgsForCall(1)).To(Equal(&models.AppInstanceMetric{
					AppId:         "an-app-id",
					InstanceIndex: 1,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameMemory,
					Unit:          models.UnitMegaBytes,
					Value:         "22",
					Timestamp:     222222,
				}))

			})
			Context("when save metrics to database fails", func() {
				BeforeEach(func() {
					database.SaveMetricReturns(errors.New("an error"))
				})
				It("logs the errors", func() {
					Eventually(buffer).Should(gbytes.Say("process-event-save-metric"))
					Eventually(buffer).Should(gbytes.Say("an error"))
				})
			})

		})

		Context("when there is no containermetric event", func() {
			BeforeEach(func() {
				go func() {
					eventType := events.Envelope_CounterEvent
					msgChan <- &events.Envelope{EventType: &eventType}
				}()
			})
			It("Saves nothing to database", func() {
				Consistently(database.SaveMetricCallCount).Should(BeZero())
			})
		})
		Context("when there is error  streaming events", func() {
			BeforeEach(func() {
				errChan <- errors.New("an error")
			})
			It("logs the error", func() {
				Eventually(buffer).Should(gbytes.Say("stream-metrics"))
				Eventually(buffer).Should(gbytes.Say("an-app-id"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})
	})

	Describe("Stop", func() {
		BeforeEach(func() {
			streamer.Start()
		})
		JustBeforeEach(func() {
			streamer.Stop()
		})
		It("stops the streaming", func() {
			Eventually(buffer).Should(gbytes.Say("app-streamer-stopped"))
			Eventually(buffer).Should(gbytes.Say("an-app-id"))
		})
		Context("when error occurs closing the connection", func() {
			BeforeEach(func() {
				noaa.CloseReturns(errors.New("an error"))
			})
			It("logs the error", func() {
				Eventually(buffer).Should(gbytes.Say("close-noaa-connections"))
				Eventually(buffer).Should(gbytes.Say("an-app-id"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})
		Context("when closing the connection succeeds", func() {
			It("logs the message", func() {
				Eventually(buffer).Should(gbytes.Say("noaa-connections-closed"))
				Eventually(buffer).Should(gbytes.Say("an-app-id"))
			})
		})

	})

})
