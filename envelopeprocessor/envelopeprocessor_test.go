package envelopeprocessor_test

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	TestCollectInterval = 1 * time.Second
)

var _ = Describe("Envelopeprocessor", func() {

	var envelopes []*loggregator_v2.Envelope
	var processor envelopeprocessor.Processor
	var logger *lagertest.TestLogger

	AfterEach(func() {
		envelopes = nil
	})

	Describe("#GetGaugeInstanceMetrics", func() {
		Context("processing custom metrics", func() {
			BeforeEach(func() {
				envelopes = append(envelopes, GenerateCustomMetrics("test-app-id", "1", "custom_name", "custom_unit", 11.88, 1111))
				envelopes = append(envelopes, GenerateCustomMetrics("test-app-id", "0", "custom_name", "custom_unit", 11.08, 1111))
				logger = lagertest.NewTestLogger("envelopeProcessor")
				processor = envelopeprocessor.NewProcessor(logger)
			})

			It("sends standard app instance metrics to channel", func() {
				timestamp := time.Now().UnixNano()
				metrics, err := envelopeprocessor.GetGaugeInstanceMetrics(envelopes, timestamp)
				Expect(err).NotTo(HaveOccurred())
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   timestamp,
					Name:          "custom_name",
					Unit:          "custom_unit",
					Value:         "12",
					Timestamp:     1111,
				}))

				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   timestamp,
					Name:          "custom_name",
					Unit:          "custom_unit",
					Value:         "12",
					Timestamp:     1111,
				}))

			})
		})

		Context("processing container metrics", func() {
			BeforeEach(func() {
				envelopes = append(envelopes, GenerateContainerMetrics("test-app-id", "0", 10.2, 10*1024*1024, 20*1024*1024, 1111))
				envelopes = append(envelopes, GenerateContainerMetrics("test-app-id", "1", 10.6, 10.2*1024*1024, 20*1024*1024, 1111))
			})

			It("sends standard app instance metrics to channel", func() {
				timestamp := time.Now().UnixNano()
				metrics, err := processor.GetGaugeInstanceMetrics(envelopes, timestamp)
				Expect(err).NotTo(HaveOccurred())
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   timestamp,
					Name:          models.MetricNameMemoryUsed,
					Unit:          models.UnitMegaBytes,
					Value:         "10",
					Timestamp:     1111,
				}))

				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   timestamp,
					Name:          models.MetricNameMemoryUtil,
					Unit:          models.UnitPercentage,
					Value:         "50",
					Timestamp:     1111,
				}))

				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   timestamp,
					Name:          models.MetricNameCPUUtil,
					Unit:          models.UnitPercentage,
					Value:         "11",
					Timestamp:     1111,
				}))
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   timestamp,
					Name:          models.MetricNameMemoryUsed,
					Unit:          models.UnitMegaBytes,
					Value:         "11",
					Timestamp:     1111,
				}))

				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   timestamp,
					Name:          models.MetricNameMemoryUtil,
					Unit:          models.UnitPercentage,
					Value:         "51",
					Timestamp:     1111,
				}))

				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   timestamp,
					Name:          models.MetricNameCPUUtil,
					Unit:          models.UnitPercentage,
					Value:         "11",
					Timestamp:     1111,
				}))

			})
		})
	})

	Describe("#GetHttpStartStopInstanceMetrics", func() {
		BeforeEach(func() {
			envelopes = append(envelopes, GenerateHttpStartStopEnvelope("test-app-id", "0", 10*1000*1000, 25*1000*1000, 1111))
			envelopes = append(envelopes, GenerateHttpStartStopEnvelope("test-app-id", "1", 10*1000*1000, 30*1000*1000, 1111))
			envelopes = append(envelopes, GenerateHttpStartStopEnvelope("test-app-id", "0", 20*1000*1000, 30*1000*1000, 1111))
			envelopes = append(envelopes, GenerateHttpStartStopEnvelope("test-app-id", "1", 20*1000*1000, 50*1000*1000, 1111))
			envelopes = append(envelopes, GenerateHttpStartStopEnvelope("test-app-id", "1", 20*1000*1000, 30*1000*1000, 1111))
		})

		It("sends throughput and responsetime metric to channel", func() {
			timestamp := time.Now().UnixNano()
			metrics := processor.GetHttpStartStopInstanceMetrics(envelopes, "test-app-id", timestamp, TestCollectInterval)

			Expect(metrics).To(ContainElement(models.AppInstanceMetric{
				AppId:         "test-app-id",
				InstanceIndex: 0,
				CollectedAt:   timestamp,
				Name:          models.MetricNameThroughput,
				Unit:          models.UnitRPS,
				Value:         "2",
				Timestamp:     timestamp,
			}))

			Expect(metrics).To(ContainElement(models.AppInstanceMetric{
				AppId:         "test-app-id",
				InstanceIndex: 0,
				CollectedAt:   timestamp,
				Name:          models.MetricNameResponseTime,
				Unit:          models.UnitMilliseconds,
				Value:         "13",
				Timestamp:     timestamp,
			}))

			Expect(metrics).To(ContainElement(models.AppInstanceMetric{
				AppId:         "test-app-id",
				InstanceIndex: 1,
				CollectedAt:   timestamp,
				Name:          models.MetricNameThroughput,
				Unit:          models.UnitRPS,
				Value:         "3",
				Timestamp:     timestamp,
			}))

			Expect(metrics).To(ContainElement(models.AppInstanceMetric{
				AppId:         "test-app-id",
				InstanceIndex: 1,
				CollectedAt:   timestamp,
				Name:          models.MetricNameResponseTime,
				Unit:          models.UnitMilliseconds,
				Value:         "20",
				Timestamp:     timestamp,
			}))

		})

		Context("when no available envelopes for app", func() {
			BeforeEach(func() {
				envelopes = []*loggregator_v2.Envelope{}
			})

			It("sends send 0 throughput and responsetime metric", func() {
				timestamp := time.Now().UnixNano()
				metrics := processor.GetHttpStartStopInstanceMetrics(envelopes, "another-test-app-id", timestamp, TestCollectInterval)
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "another-test-app-id",
					InstanceIndex: 0,
					CollectedAt:   timestamp,
					Name:          models.MetricNameThroughput,
					Unit:          models.UnitRPS,
					Value:         "0",
					Timestamp:     timestamp,
				}))
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "another-test-app-id",
					InstanceIndex: 0,
					CollectedAt:   timestamp,
					Name:          models.MetricNameResponseTime,
					Unit:          models.UnitMilliseconds,
					Value:         "0",
					Timestamp:     timestamp,
				}))

			})

		})
	})
})

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
