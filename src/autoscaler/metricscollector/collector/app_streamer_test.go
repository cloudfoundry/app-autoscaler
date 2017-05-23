package collector_test

import (
	"autoscaler/cf"
	. "autoscaler/metricscollector/collector"
	"autoscaler/metricscollector/fakes"
	"autoscaler/metricscollector/noaa"
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
		cfc          *fakes.FakeCfClient
		noaaConsumer *fakes.FakeNoaaConsumer
		database     *fakes.FakeInstanceMetricsDB
		streamer     AppStreamer
		buffer       *gbytes.Buffer
		msgChan      chan *events.Envelope
		errChan      chan error
		fclock       *fakeclock.FakeClock
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		noaaConsumer = &fakes.FakeNoaaConsumer{}
		database = &fakes.FakeInstanceMetricsDB{}

		logger := lagertest.NewTestLogger("AppStreamer-test")
		buffer = logger.Buffer()
		fclock = fakeclock.NewFakeClock(time.Now())

		streamer = NewAppStreamer(logger, "an-app-id", TestCollectInterval, cfc, noaaConsumer, database, fclock)

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
			noaaConsumer.StreamStub = func(appId string, authToken string) (outputChan <-chan *events.Envelope, errorChan <-chan error) {
				Expect(appId).To(Equal("an-app-id"))
				Expect(authToken).To(Equal("test-access-token"))
				return msgChan, errChan
			}
		})

		Context("when there are containermetric events", func() {
			BeforeEach(func() {
				go func() {
					msgChan <- noaa.NewContainerEnvelope(111111, "an-app-id", 0, 12.8, 1000, 1000, 3000, 2000)
					msgChan <- noaa.NewContainerEnvelope(222222, "an-app-id", 1, 12.8, 2000, 1000, 3000, 2000)
				}()
			})
			It("Saves metrics to database", func() {
				Eventually(database.SaveMetricCallCount).Should(Equal(2))
				Expect(database.SaveMetricArgsForCall(0)).To(Equal(&models.AppInstanceMetric{
					AppId:         "an-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameMemory,
					Unit:          models.UnitPercentage,
					Value:         "33",
					Timestamp:     111111,
				}))
				Expect(database.SaveMetricArgsForCall(1)).To(Equal(&models.AppInstanceMetric{
					AppId:         "an-app-id",
					InstanceIndex: 1,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameMemory,
					Unit:          models.UnitPercentage,
					Value:         "67",
					Timestamp:     222222,
				}))

			})
			Context("when saving to database fails", func() {
				BeforeEach(func() {
					database.SaveMetricReturns(errors.New("an error"))
				})
				It("logs the errors", func() {
					Eventually(buffer).Should(gbytes.Say("process-event-save-metric"))
					Eventually(buffer).Should(gbytes.Say("an error"))
				})
			})
		})

		Context("when there are httpstartstop events", func() {
			It("Saves metrics to database with the given time interval", func() {
				go func() {
					msgChan <- noaa.NewHttpStartStopEnvelope(111111, 100000000, 200000000, 0)
					msgChan <- noaa.NewHttpStartStopEnvelope(222222, 300000000, 600000000, 0)
				}()

				By("collecting and computing throughput and responsetime for first interval")
				Consistently(database.SaveMetricCallCount).Should(Equal(0))

				By("save throughput and responsetime metric after the first collect interval")
				fclock.WaitForWatcherAndIncrement(TestCollectInterval)
				Eventually(database.SaveMetricCallCount).Should(Equal(2))

				Expect(database.SaveMetricArgsForCall(0)).To(Equal(&models.AppInstanceMetric{
					AppId:         "an-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameThroughput,
					Unit:          models.UnitRPS,
					Value:         "2",
					Timestamp:     fclock.Now().UnixNano(),
				}))
				Expect(database.SaveMetricArgsForCall(1)).To(Equal(&models.AppInstanceMetric{
					AppId:         "an-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameResponseTime,
					Unit:          models.UnitMilliseconds,
					Value:         "200",
					Timestamp:     fclock.Now().UnixNano(),
				}))

				go func() {
					msgChan <- noaa.NewHttpStartStopEnvelope(333333, 100000000, 300000000, 1)
					msgChan <- noaa.NewHttpStartStopEnvelope(555555, 300000000, 600000000, 1)
					msgChan <- noaa.NewHttpStartStopEnvelope(666666, 300000000, 700000000, 1)
				}()

				By("collecting and computing throughput and responsetime for second interval")
				Consistently(database.SaveMetricCallCount).Should(Equal(2))

				By("save throughput and responsetime metric after the second collect interval")
				fclock.Increment(TestCollectInterval)
				Eventually(database.SaveMetricCallCount).Should(Equal(4))

				Expect(database.SaveMetricArgsForCall(2)).To(Equal(&models.AppInstanceMetric{
					AppId:         "an-app-id",
					InstanceIndex: 1,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameThroughput,
					Unit:          models.UnitRPS,
					Value:         "3",
					Timestamp:     fclock.Now().UnixNano(),
				}))
				Expect(database.SaveMetricArgsForCall(3)).To(Equal(&models.AppInstanceMetric{
					AppId:         "an-app-id",
					InstanceIndex: 1,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameResponseTime,
					Unit:          models.UnitMilliseconds,
					Value:         "300",
					Timestamp:     fclock.Now().UnixNano(),
				}))
			})

			Context("when the app has multiple instances", func() {
				BeforeEach(func() {
					go func() {
						msgChan <- noaa.NewHttpStartStopEnvelope(111111, 100000000, 200000000, 0)
						msgChan <- noaa.NewHttpStartStopEnvelope(222222, 300000000, 500000000, 1)
						msgChan <- noaa.NewHttpStartStopEnvelope(333333, 200000000, 600000000, 2)
						msgChan <- noaa.NewHttpStartStopEnvelope(555555, 300000000, 500000000, 2)
					}()

				})
				It("saves throughput and responsetime metrics of multiple instances", func() {
					Consistently(database.SaveMetricCallCount).Should(Equal(0))
					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					Eventually(database.SaveMetricCallCount).Should(Equal(6))
				})
			})

			Context("when database fails", func() {
				BeforeEach(func() {
					database.SaveMetricReturns(errors.New("an error"))
					go func() {
						msgChan <- noaa.NewHttpStartStopEnvelope(111111, 100000000, 200000000, 0)
					}()
				})
				It("logs the errors", func() {
					Consistently(database.SaveMetricCallCount).Should(Equal(0))

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					Eventually(buffer).Should(gbytes.Say("save-metric-to-database"))
					Eventually(buffer).Should(gbytes.Say("an error"))
					Eventually(buffer).Should(gbytes.Say("throughput"))

					Eventually(buffer).Should(gbytes.Say("save-metric-to-database"))
					Eventually(buffer).Should(gbytes.Say("an error"))
					Eventually(buffer).Should(gbytes.Say("responsetime"))
				})
			})

		})

		Context("when there is no containermetrics or httpstartstop event", func() {
			BeforeEach(func() {
				go func() {
					eventType := events.Envelope_CounterEvent
					msgChan <- &events.Envelope{EventType: &eventType}
				}()
			})
			It("Saves nothing to database", func() {
				fclock.WaitForWatcherAndIncrement(TestCollectInterval)
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
				noaaConsumer.CloseReturns(errors.New("an error"))
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
