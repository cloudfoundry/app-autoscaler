package collector_test

import (
	"autoscaler/cf"
	"autoscaler/collection"
	"autoscaler/fakes"
	. "autoscaler/metricscollector/collector"
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
		cfc                           *fakes.FakeCFClient
		noaaConsumer                  *fakes.FakeNoaaConsumer
		streamer                      AppCollector
		buffer                        *gbytes.Buffer
		msgChan                       chan *events.Envelope
		errChan                       chan error
		fclock                        *fakeclock.FakeClock
		dataChan                      chan *models.AppInstanceMetric
		cacheSize                     int
		logger                        *lagertest.TestLogger
		isMetricsPersistencySupported bool
		metrics                       []*models.AppInstanceMetric
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCFClient{}
		noaaConsumer = &fakes.FakeNoaaConsumer{}
		logger = lagertest.NewTestLogger("AppStreamer-test")
		buffer = logger.Buffer()
		fclock = fakeclock.NewFakeClock(time.Now())
		dataChan = make(chan *models.AppInstanceMetric)
		cacheSize = 100
		msgChan = make(chan *events.Envelope)
		errChan = make(chan error, 1)
		isMetricsPersistencySupported = true
	})

	Describe("Start", func() {

		JustBeforeEach(func() {
			streamer = NewAppStreamer(logger, "an-app-id", TestCollectInterval, cacheSize, isMetricsPersistencySupported, cfc, noaaConsumer, fclock, dataChan)
			streamer.Start()
		})

		AfterEach(func() {
			streamer.Stop()
		})

		BeforeEach(func() {
			cfc.GetTokensReturns(cf.Tokens{AccessToken: "test-access-token"})
			noaaConsumer.StreamStub = func(appId string, authToken string) (outputChan <-chan *events.Envelope, errorChan <-chan error) {
				Expect(appId).To(Equal("an-app-id"))
				Expect(authToken).To(Equal("Bearer test-access-token"))
				return msgChan, errChan
			}
		})

		Context("when there are containermetric events", func() {
			BeforeEach(func() {
				metrics = []*models.AppInstanceMetric{
					&models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameMemoryUsed,
						Unit:          models.UnitMegaBytes,
						Value:         "95",
						Timestamp:     111111,
					},
					&models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameMemoryUtil,
						Unit:          models.UnitPercentage,
						Value:         "33",
						Timestamp:     111111,
					},
					&models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameCPUUtil,
						Unit:          models.UnitPercentage,
						Value:         "13",
						Timestamp:     111111,
					},
					&models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 1,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameMemoryUsed,
						Unit:          models.UnitMegaBytes,
						Value:         "191",
						Timestamp:     222222,
					},
					&models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 1,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameMemoryUtil,
						Unit:          models.UnitPercentage,
						Value:         "67",
						Timestamp:     222222,
					},
					&models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 1,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameCPUUtil,
						Unit:          models.UnitPercentage,
						Value:         "31",
						Timestamp:     222222,
					},
				}

				go func() {
					msgChan <- noaa.NewContainerEnvelope(111111, "an-app-id", 0, 12.8, 100000000, 1000000000, 300000000, 2000000000)
					msgChan <- noaa.NewContainerEnvelope(222222, "an-app-id", 1, 30.6, 200000000, 1000000000, 300000000, 2000000000)
				}()
			})

			Context("when metrics are not persisted to database", func() {
				BeforeEach(func() {
					isMetricsPersistencySupported = false
				})
				It("sends container metrics to cache only", func() {
					Consistently(dataChan).ShouldNot(Receive())

					data, ok := streamer.Query(111111, 333333, map[string]string{models.MetricLabelName: models.MetricNameMemoryUsed})
					Expect(ok).To(BeTrue())
					Expect(data).To(Equal([]collection.TSD{metrics[0], metrics[3]}))

					data, ok = streamer.Query(111111, 333333, map[string]string{models.MetricLabelName: models.MetricNameMemoryUtil})
					Expect(ok).To(BeTrue())
					Expect(data).To(Equal([]collection.TSD{metrics[1], metrics[4]}))

					data, ok = streamer.Query(111111, 333333, map[string]string{models.MetricLabelName: models.MetricNameCPUUtil})
					Expect(ok).To(BeTrue())
					Expect(data).To(Equal([]collection.TSD{metrics[2], metrics[5]}))

					data, ok = streamer.Query(111111, 333333, map[string]string{models.MetricLabelName: models.MetricNameThroughput})
					Expect(ok).To(BeTrue())
					Expect(data).To(BeEmpty())

					By("collecting and computing throughput and response time")
					Consistently(dataChan).ShouldNot(Receive())

					By("sending throughput and response time after the collect interval")
					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					metric1 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameThroughput,
						Unit:          models.UnitRPS,
						Value:         "0",
						Timestamp:     fclock.Now().UnixNano(),
					}

					metric2 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameResponseTime,
						Unit:          models.UnitMilliseconds,
						Value:         "0",
						Timestamp:     fclock.Now().UnixNano(),
					}

					Consistently(dataChan).ShouldNot(Receive())

					data, ok = streamer.Query(0, fclock.Now().UnixNano(), map[string]string{models.MetricLabelName: models.MetricNameThroughput})
					Expect(ok).To(BeFalse())
					Expect(data).To(Equal([]collection.TSD{metric1}))

					data, ok = streamer.Query(0, fclock.Now().UnixNano(), map[string]string{models.MetricLabelName: models.MetricNameResponseTime})
					Expect(ok).To(BeFalse())
					Expect(data).To(Equal([]collection.TSD{metric2}))

				})
			})

			Context("when metrics are persisted to database", func() {
				It("sends container metrics to channel and cache", func() {
					Expect(<-dataChan).To(Equal(metrics[0]))
					Expect(<-dataChan).To(Equal(metrics[1]))
					Expect(<-dataChan).To(Equal(metrics[2]))
					Expect(<-dataChan).To(Equal(metrics[3]))
					Expect(<-dataChan).To(Equal(metrics[4]))
					Expect(<-dataChan).To(Equal(metrics[5]))

					data, ok := streamer.Query(111111, 333333, map[string]string{models.MetricLabelName: models.MetricNameMemoryUsed})
					Expect(ok).To(BeTrue())
					Expect(data).To(Equal([]collection.TSD{metrics[0], metrics[3]}))

					data, ok = streamer.Query(111111, 333333, map[string]string{models.MetricLabelName: models.MetricNameMemoryUtil})
					Expect(ok).To(BeTrue())
					Expect(data).To(Equal([]collection.TSD{metrics[1], metrics[4]}))

					data, ok = streamer.Query(111111, 333333, map[string]string{models.MetricLabelName: models.MetricNameCPUUtil})
					Expect(ok).To(BeTrue())
					Expect(data).To(Equal([]collection.TSD{metrics[2], metrics[5]}))

					data, ok = streamer.Query(111111, 333333, map[string]string{models.MetricLabelName: models.MetricNameThroughput})
					Expect(ok).To(BeTrue())
					Expect(data).To(BeEmpty())

					By("collecting and computing throughput")
					Consistently(dataChan).ShouldNot(Receive())

					By("sending throughput after the collect interval")
					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					metric1 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameThroughput,
						Unit:          models.UnitRPS,
						Value:         "0",
						Timestamp:     fclock.Now().UnixNano(),
					}

					metric2 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameResponseTime,
						Unit:          models.UnitMilliseconds,
						Value:         "0",
						Timestamp:     fclock.Now().UnixNano(),
					}

					Expect(<-dataChan).To(Equal(metric1))
					Expect(<-dataChan).To(Equal(metric2))

					data, ok = streamer.Query(0, fclock.Now().UnixNano(), map[string]string{models.MetricLabelName: models.MetricNameThroughput})
					Expect(ok).To(BeFalse())
					Expect(data).To(Equal([]collection.TSD{metric1}))

					data, ok = streamer.Query(0, fclock.Now().UnixNano(), map[string]string{models.MetricLabelName: models.MetricNameResponseTime})
					Expect(ok).To(BeFalse())
					Expect(data).To(Equal([]collection.TSD{metric2}))

				})

			})
		})

		Context("when there are httpstartstop events", func() {
			Context("when metrics are persisted to database", func() {
				It("sends responsetime and throughput metrics to cache and channel", func() {
					msgChan <- noaa.NewHttpStartStopEnvelope(111111, 100000000, 200000000, 0)
					msgChan <- noaa.NewHttpStartStopEnvelope(222222, 300000000, 600000000, 0)

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)

					metric1 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameThroughput,
						Unit:          models.UnitRPS,
						Value:         "2",
						Timestamp:     fclock.Now().UnixNano(),
					}
					metric2 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameResponseTime,
						Unit:          models.UnitMilliseconds,
						Value:         "200",
						Timestamp:     fclock.Now().UnixNano(),
					}

					Expect(<-dataChan).To(Equal(metric1))
					Expect(<-dataChan).To(Equal(metric2))

					data, ok := streamer.Query(0, fclock.Now().UnixNano(), map[string]string{})
					Expect(ok).To(BeFalse())
					Expect(data).To(Equal([]collection.TSD{metric1, metric2}))

					msgChan <- noaa.NewHttpStartStopEnvelope(333333, 100000000, 300000000, 1)
					msgChan <- noaa.NewHttpStartStopEnvelope(555555, 300000000, 600000000, 1)
					msgChan <- noaa.NewHttpStartStopEnvelope(666666, 300000000, 700000000, 1)
					Consistently(dataChan).ShouldNot(Receive())

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)

					metric3 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 1,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameThroughput,
						Unit:          models.UnitRPS,
						Value:         "3",
						Timestamp:     fclock.Now().UnixNano(),
					}
					metric4 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 1,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameResponseTime,
						Unit:          models.UnitMilliseconds,
						Value:         "300",
						Timestamp:     fclock.Now().UnixNano(),
					}

					Expect(<-dataChan).To(Equal(metric3))
					Expect(<-dataChan).To(Equal(metric4))

					data, ok = streamer.Query(0, fclock.Now().UnixNano(), map[string]string{})
					Expect(ok).To(BeFalse())
					Expect(data).To(Equal([]collection.TSD{metric1, metric2, metric3, metric4}))
				})
			})

			Context("when metrics are not persisted to database", func() {
				BeforeEach(func() {
					isMetricsPersistencySupported = false
				})
				It("sends responsetime and throughput metrics to cache only", func() {
					msgChan <- noaa.NewHttpStartStopEnvelope(111111, 100000000, 200000000, 0)
					msgChan <- noaa.NewHttpStartStopEnvelope(222222, 300000000, 600000000, 0)

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)

					metric1 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameThroughput,
						Unit:          models.UnitRPS,
						Value:         "2",
						Timestamp:     fclock.Now().UnixNano(),
					}
					metric2 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameResponseTime,
						Unit:          models.UnitMilliseconds,
						Value:         "200",
						Timestamp:     fclock.Now().UnixNano(),
					}

					Consistently(dataChan).ShouldNot(Receive())

					data, ok := streamer.Query(0, fclock.Now().UnixNano(), map[string]string{})
					Expect(ok).To(BeFalse())
					Expect(data).To(Equal([]collection.TSD{metric1, metric2}))

					msgChan <- noaa.NewHttpStartStopEnvelope(333333, 100000000, 300000000, 1)
					msgChan <- noaa.NewHttpStartStopEnvelope(555555, 300000000, 600000000, 1)
					msgChan <- noaa.NewHttpStartStopEnvelope(666666, 300000000, 700000000, 1)
					Consistently(dataChan).ShouldNot(Receive())

					fclock.WaitForWatcherAndIncrement(TestCollectInterval)

					metric3 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 1,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameThroughput,
						Unit:          models.UnitRPS,
						Value:         "3",
						Timestamp:     fclock.Now().UnixNano(),
					}
					metric4 := &models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 1,
						CollectedAt:   fclock.Now().UnixNano(),
						Name:          models.MetricNameResponseTime,
						Unit:          models.UnitMilliseconds,
						Value:         "300",
						Timestamp:     fclock.Now().UnixNano(),
					}

					Consistently(dataChan).ShouldNot(Receive())
					data, ok = streamer.Query(0, fclock.Now().UnixNano(), map[string]string{})
					Expect(ok).To(BeFalse())
					Expect(data).To(Equal([]collection.TSD{metric1, metric2, metric3, metric4}))
				})
			})

			Context("when the app has multiple instances", func() {
				JustBeforeEach(func() {
					msgChan <- noaa.NewHttpStartStopEnvelope(111111, 100000000, 200000000, 0)
					msgChan <- noaa.NewHttpStartStopEnvelope(222222, 300000000, 500000000, 1)
					msgChan <- noaa.NewHttpStartStopEnvelope(333333, 200000000, 600000000, 2)
					msgChan <- noaa.NewHttpStartStopEnvelope(555555, 300000000, 500000000, 2)
				})
				It("sends throughput and responsetime metrics of multiple instances to channel and cache", func() {
					fclock.WaitForWatcherAndIncrement(TestCollectInterval)
					Eventually(dataChan).Should(Receive())
					Eventually(dataChan).Should(Receive())
					Eventually(dataChan).Should(Receive())
					Eventually(dataChan).Should(Receive())
					Eventually(dataChan).Should(Receive())
					Eventually(dataChan).Should(Receive())
					Consistently(dataChan).ShouldNot(Receive())

					data, ok := streamer.Query(0, fclock.Now().UnixNano(), map[string]string{})
					Expect(ok).To(BeFalse())
					Expect(data).To(HaveLen(6))

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
			It("Sends zero throughput metric to cache and chaneel", func() {

				By("sending throughput and response time after the collect interval")
				fclock.WaitForWatcherAndIncrement(TestCollectInterval)

				metric1 := &models.AppInstanceMetric{
					AppId:         "an-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameThroughput,
					Unit:          models.UnitRPS,
					Value:         "0",
					Timestamp:     fclock.Now().UnixNano(),
				}

				metric2 := &models.AppInstanceMetric{
					AppId:         "an-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameResponseTime,
					Unit:          models.UnitMilliseconds,
					Value:         "0",
					Timestamp:     fclock.Now().UnixNano(),
				}

				Expect(<-dataChan).To(Equal(metric1))
				Expect(<-dataChan).To(Equal(metric2))

				data, ok := streamer.Query(0, fclock.Now().UnixNano(), map[string]string{})
				Expect(ok).To(BeFalse())
				Expect(data).To(Equal([]collection.TSD{metric1, metric2}))
			})
		})
		Context("when there is error streaming events", func() {
			BeforeEach(func() {
				errChan <- errors.New("an error")
			})
			It("logs the error and reconnect in next tick", func() {
				Eventually(buffer).Should(gbytes.Say("stream-metrics"))
				Eventually(buffer).Should(gbytes.Say("an-app-id"))
				Eventually(buffer).Should(gbytes.Say("an error"))

				fclock.WaitForWatcherAndIncrement(TestCollectInterval)
				Eventually(noaaConsumer.CloseCallCount).Should(Equal(1))
				Eventually(noaaConsumer.StreamCallCount).Should(Equal(2))
				Eventually(buffer).Should(gbytes.Say("noaa-reconnected"))
				Eventually(buffer).Should(gbytes.Say("an-app-id"))
				Consistently(buffer).ShouldNot(gbytes.Say("compute-and-save-metrics"))
			})
		})
	})

	Describe("Stop", func() {
		BeforeEach(func() {
			streamer = NewAppStreamer(logger, "an-app-id", TestCollectInterval, cacheSize, isMetricsPersistencySupported, cfc, noaaConsumer, fclock, dataChan)
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
				Eventually(buffer).Should(gbytes.Say("close-noaa-connection"))
				Eventually(buffer).Should(gbytes.Say("an-app-id"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})
		Context("when closing the connection succeeds", func() {
			It("logs the message", func() {
				Eventually(buffer).Should(gbytes.Say("noaa-connection-closed"))
				Eventually(buffer).Should(gbytes.Say("an-app-id"))
				Eventually(buffer).Should(gbytes.Say("app-streamer-stopped"))
			})
		})

	})

})
