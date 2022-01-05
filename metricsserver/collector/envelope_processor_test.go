package collector_test

import (
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsserver/collector"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EnvelopeProcessor", func() {

	var (
		logger         *lagertest.TestLogger
		fclock         *fakeclock.FakeClock
		processorIndex int
		numProcessors  int
		envelopeChan   chan *loggregator_v2.Envelope
		metricChan     chan *models.AppInstanceMetric
		getAppIDs      func() map[string]bool
		processor      EnvelopeProcessor
	)

	BeforeEach(func() {
		numProcessors = 1
		processorIndex = 0
		logger = lagertest.NewTestLogger("collector-test")
		fclock = fakeclock.NewFakeClock(time.Now())
		envelopeChan = make(chan *loggregator_v2.Envelope, 10)
		metricChan = make(chan *models.AppInstanceMetric, 10)
		getAppIDs = func() map[string]bool {
			return map[string]bool{}
		}
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			processor = NewEnvelopeProcessor(logger, TestCollectInterval, fclock,
				processorIndex, numProcessors, envelopeChan, metricChan, getAppIDs)
			processor.Start()
			// make sure the envelopes have been processed
			time.Sleep(2 * time.Second)
		})
		AfterEach(func() {
			processor.Stop()
		})

		Context("processing container metrics", func() {
			BeforeEach(func() {
				Expect(envelopeChan).Should(BeSent(GenerateContainerMetrics("test-app-id", "0", 10.2, 10*1024*1024, 20*1024*1024, 1111)))
				Expect(envelopeChan).Should(BeSent(GenerateContainerMetrics("test-app-id", "1", 10.6, 10.2*1024*1024, 20*1024*1024, 1111)))

			})
			It("sends standard app instance metrics to channel", func() {
				Eventually(metricChan).Should(Receive(Equal(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameMemoryUsed,
					Unit:          models.UnitMegaBytes,
					Value:         "10",
					Timestamp:     1111,
				})))

				Eventually(metricChan).Should(Receive(Equal(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameMemoryUtil,
					Unit:          models.UnitPercentage,
					Value:         "50",
					Timestamp:     1111,
				})))

				Eventually(metricChan).Should(Receive(Equal(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameCPUUtil,
					Unit:          models.UnitPercentage,
					Value:         "11",
					Timestamp:     1111,
				})))

				Eventually(metricChan).Should(Receive(Equal(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameMemoryUsed,
					Unit:          models.UnitMegaBytes,
					Value:         "11",
					Timestamp:     1111,
				})))

				Eventually(metricChan).Should(Receive(Equal(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameMemoryUtil,
					Unit:          models.UnitPercentage,
					Value:         "51",
					Timestamp:     1111,
				})))

				Eventually(metricChan).Should(Receive(Equal(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameCPUUtil,
					Unit:          models.UnitPercentage,
					Value:         "11",
					Timestamp:     1111,
				})))

			})
		})

		Context("processing custom metrics", func() {
			BeforeEach(func() {
				Expect(envelopeChan).Should(BeSent(GenerateCustomMetrics("test-app-id", "1", "custom_name", "custom_unit", 11.88, 1111)))
				Expect(envelopeChan).Should(BeSent(GenerateCustomMetrics("test-app-id", "0", "custom_name", "custom_unit", 11.08, 1111)))
			})
			It("sends standard app instance metrics to channel", func() {
				Eventually(metricChan).Should(Receive(Equal(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          "custom_name",
					Unit:          "custom_unit",
					Value:         "12",
					Timestamp:     1111,
				})))

				Eventually(metricChan).Should(Receive(Equal(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          "custom_name",
					Unit:          "custom_unit",
					Value:         "12",
					Timestamp:     1111,
				})))

			})
		})

		Context("Computing throughput and responsetime", func() {
			BeforeEach(func() {
				getAppIDs = func() map[string]bool {
					return map[string]bool{"test-app-id": true}
				}
				Expect(envelopeChan).Should(BeSent(GenerateHttpStartStopEnvelope("test-app-id", "0", 10*1000*1000, 25*1000*1000, 1111)))
				Expect(envelopeChan).Should(BeSent(GenerateHttpStartStopEnvelope("test-app-id", "1", 10*1000*1000, 30*1000*1000, 1111)))
				Expect(envelopeChan).Should(BeSent(GenerateHttpStartStopEnvelope("test-app-id", "0", 20*1000*1000, 30*1000*1000, 1111)))
				Expect(envelopeChan).Should(BeSent(GenerateHttpStartStopEnvelope("test-app-id", "1", 20*1000*1000, 50*1000*1000, 1111)))
				Expect(envelopeChan).Should(BeSent(GenerateHttpStartStopEnvelope("test-app-id", "1", 20*1000*1000, 30*1000*1000, 1111)))

			})
			It("sends throughput and responsetime metric to channel", func() {
				Consistently(metricChan).ShouldNot(Receive())

				fclock.WaitForWatcherAndIncrement(TestCollectInterval)
				metrics := []*models.AppInstanceMetric{}

				Eventually(func() int {
					metrics = append(metrics, <-metricChan)
					return len(metrics)
				}).Should(Equal(4))
				Expect(metrics).To(ContainElement(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameThroughput,
					Unit:          models.UnitRPS,
					Value:         "2",
					Timestamp:     fclock.Now().UnixNano(),
				}))

				Expect(metrics).To(ContainElement(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameResponseTime,
					Unit:          models.UnitMilliseconds,
					Value:         "13",
					Timestamp:     fclock.Now().UnixNano(),
				}))

				Expect(metrics).To(ContainElement(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameThroughput,
					Unit:          models.UnitRPS,
					Value:         "3",
					Timestamp:     fclock.Now().UnixNano(),
				}))

				Expect(metrics).To(ContainElement(&models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   fclock.Now().UnixNano(),
					Name:          models.MetricNameResponseTime,
					Unit:          models.UnitMilliseconds,
					Value:         "20",
					Timestamp:     fclock.Now().UnixNano(),
				}))

			})

			Context("when the app does not have http requests", func() {
				BeforeEach(func() {
					getAppIDs = func() map[string]bool {
						return map[string]bool{"another-test-app-id": true}
					}
				})

				Context("when the current processor is responsible for the app", func() {
					BeforeEach(func() {
						numProcessors = 3
						processorIndex = 0
					})

					It("sends send 0 throughput and responsetime metric", func() {
						Consistently(metricChan).ShouldNot(Receive())
						fclock.WaitForWatcherAndIncrement(TestCollectInterval)
						Eventually(metricChan).Should(Receive(Equal(&models.AppInstanceMetric{
							AppId:         "another-test-app-id",
							InstanceIndex: 0,
							CollectedAt:   fclock.Now().UnixNano(),
							Name:          models.MetricNameThroughput,
							Unit:          models.UnitRPS,
							Value:         "0",
							Timestamp:     fclock.Now().UnixNano(),
						})))
						Eventually(metricChan).Should(Receive(Equal(&models.AppInstanceMetric{
							AppId:         "another-test-app-id",
							InstanceIndex: 0,
							CollectedAt:   fclock.Now().UnixNano(),
							Name:          models.MetricNameResponseTime,
							Unit:          models.UnitMilliseconds,
							Value:         "0",
							Timestamp:     fclock.Now().UnixNano(),
						})))

					})
				})

				Context("when the current processor is not responsible for the app", func() {
					BeforeEach(func() {
						numProcessors = 3
						processorIndex = 1
					})

					It("sends nothing", func() {
						Consistently(metricChan).ShouldNot(Receive())
						fclock.WaitForWatcherAndIncrement(TestCollectInterval)
						Consistently(metricChan).ShouldNot(Receive())
					})
				})

			})

		})

	})
	Describe("Stop", func() {
		BeforeEach(func() {
			getAppIDs = func() map[string]bool {
				return map[string]bool{"test-app-id": true}
			}

			processor = NewEnvelopeProcessor(logger, TestCollectInterval, fclock,
				processorIndex, numProcessors, envelopeChan, metricChan, getAppIDs)
			processor.Start()
		})
		It("Stops processing the envelops", func() {
			Expect(envelopeChan).Should(BeSent(GenerateContainerMetrics("test-app-id", "0", 10.2, 10*1024*1024, 20*1024*1024, 1111)))
			Eventually(metricChan).Should(Receive())
			Eventually(metricChan).Should(Receive())
			Eventually(metricChan).Should(Receive())

			Expect(envelopeChan).Should(BeSent(GenerateHttpStartStopEnvelope("test-app-id", "0", 10*1000*1000, 20*1000*1000, 1111)))
			fclock.WaitForWatcherAndIncrement(TestCollectInterval)
			Eventually(metricChan).Should(Receive())
			Eventually(metricChan).Should(Receive())

			processor.Stop()
			fclock.Increment(TestCollectInterval)
			Consistently(metricChan).ShouldNot(Receive())
			Consistently(metricChan).ShouldNot(Receive())
		})
	})
})

func GenerateContainerMetrics(sourceID, instance string, cpu, memory, memoryQuota float64, timestamp int64) *loggregator_v2.Envelope {
	e := &loggregator_v2.Envelope{
		SourceId:   sourceID,
		InstanceId: instance,
		Message: &loggregator_v2.Envelope_Gauge{
			Gauge: &loggregator_v2.Gauge{
				Metrics: map[string]*loggregator_v2.GaugeValue{
					"cpu": {
						Unit:  "percentage",
						Value: cpu,
					},
					"memory": {
						Unit:  "bytes",
						Value: memory,
					},
					"memory_quota": {
						Unit:  "bytes",
						Value: memoryQuota,
					},
				},
			},
		},
		Timestamp: timestamp,
	}
	return e
}

func GenerateCustomMetrics(sourceID, instance, name, unit string, value float64, timestamp int64) *loggregator_v2.Envelope {
	e := &loggregator_v2.Envelope{
		SourceId:   sourceID,
		InstanceId: instance,
		Message: &loggregator_v2.Envelope_Gauge{
			Gauge: &loggregator_v2.Gauge{
				Metrics: map[string]*loggregator_v2.GaugeValue{
					name: {
						Unit:  unit,
						Value: value,
					},
				},
			},
		},
		Timestamp: timestamp,
	}
	return e
}

func GenerateHttpStartStopEnvelope(sourceID, instance string, start, stop, timestamp int64) *loggregator_v2.Envelope {
	e := &loggregator_v2.Envelope{
		SourceId:   sourceID,
		InstanceId: instance,
		Message: &loggregator_v2.Envelope_Timer{
			Timer: &loggregator_v2.Timer{
				Name:  "http",
				Start: start,
				Stop:  stop,
			},
		},
		Timestamp: timestamp,
	}
	return e
}
