package envelopeprocessor_test

import (
	"time"

	"golang.org/x/exp/maps"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
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

	JustBeforeEach(func() {
		logger = lagertest.NewTestLogger("envelopeProcessor")
		processor = envelopeprocessor.NewProcessor(logger)
	})

	Describe("#GetGaugeMetrics", func() {
		Context("processing custom metrics", func() {
			BeforeEach(func() {
				envelopes = append(envelopes, generateCustomMetrics("test-app-id", "1", "custom_name", "custom_unit", 11.88, 1111))
				envelopes = append(envelopes, generateCustomMetrics("test-app-id", "0", "custom_name", "custom_unit", 11.08, 1111))
			})

			It("sends standard app instance metrics to channel", func() {
				timestamp := time.Now().UnixNano()
				metrics := envelopeprocessor.GetGaugeInstanceMetrics(envelopes, timestamp)
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
				envelopes = append(envelopes, generateContainerMetricsEnvelope1("test-app-id", "0", 10.2, 5*MiB, 10*MiB, 10*MiB, 20*MiB, 1111))
				envelopes = append(envelopes, generateContainerMetricsEnvelope2("test-app-id", "0", 50, 1, 1111))

				envelopes = append(envelopes, generateContainerMetricsEnvelope1("test-app-id", "1", 10.6, 3*MiB, 33*MiB, 10.2*MiB, 20*MiB, 1111))
				envelopes = append(envelopes, generateContainerMetricsEnvelope2("test-app-id", "1", 51, 1, 1111))

				envelopes = append(envelopes, generateMemoryContainerMetrics("test-app-id", "2", 10.2*MiB, 1111))
				envelopes = append(envelopes, generateMemoryQuotaContainerMetrics("test-app-id", "2", 20*MiB, 1111))

				envelopes = append(envelopes, generateCPUContainerMetrics("test-app-id", "3", 1, 1111))

				envelopes = append(envelopes, generateCPUEntitlementContainerMetrics("test-app-id", "4", 1, 1111))

				envelopes = append(envelopes, generateDiskContainerMetrics("test-app-id", "5", 4*MiB, 1111))
				envelopes = append(envelopes, generateDiskQuotaContainerMetrics("test-app-id", "5", 10*MiB, 1111))
			})

			It("sends standard app instance metrics to channel", func() {
				timestamp := time.Now().UnixNano()
				metrics := processor.GetGaugeMetrics(envelopes, timestamp)
				Expect(len(metrics)).To(Equal(18))
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
					Name:          models.MetricNameCPU,
					Unit:          models.UnitPercentage,
					Value:         "11",
					Timestamp:     1111,
				}))
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   timestamp,
					Name:          models.MetricNameCPUUtil,
					Unit:          models.UnitPercentage,
					Value:         "50",
					Timestamp:     1111,
				}))
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   timestamp,
					Name:          models.MetricNameDiskUtil,
					Unit:          models.UnitPercentage,
					Value:         "50",
					Timestamp:     1111,
				}))
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 0,
					CollectedAt:   timestamp,
					Name:          models.MetricNameDisk,
					Unit:          models.UnitMegaBytes,
					Value:         "5",
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
					Name:          models.MetricNameCPU,
					Unit:          models.UnitPercentage,
					Value:         "11",
					Timestamp:     1111,
				}))

				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   timestamp,
					Name:          models.MetricNameCPUUtil,
					Unit:          models.UnitPercentage,
					Value:         "51",
					Timestamp:     1111,
				}))
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   timestamp,
					Name:          models.MetricNameDiskUtil,
					Unit:          models.UnitPercentage,
					Value:         "10",
					Timestamp:     1111,
				}))
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 1,
					CollectedAt:   timestamp,
					Name:          models.MetricNameDisk,
					Unit:          models.UnitMegaBytes,
					Value:         "3",
					Timestamp:     1111,
				}))
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 2,
					CollectedAt:   timestamp,
					Name:          models.MetricNameMemoryUtil,
					Unit:          models.UnitPercentage,
					Value:         "51",
					Timestamp:     1111,
				}))

				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 3,
					CollectedAt:   timestamp,
					Name:          models.MetricNameCPU,
					Unit:          models.UnitPercentage,
					Value:         "1",
					Timestamp:     1111,
				}))

				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 4,
					CollectedAt:   timestamp,
					Name:          models.MetricNameCPUUtil,
					Unit:          models.UnitPercentage,
					Value:         "1",
					Timestamp:     1111,
				}))

				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 5,
					CollectedAt:   timestamp,
					Name:          models.MetricNameDiskUtil,
					Unit:          models.UnitPercentage,
					Value:         "40",
					Timestamp:     1111,
				}))
				Expect(metrics).To(ContainElement(models.AppInstanceMetric{
					AppId:         "test-app-id",
					InstanceIndex: 5,
					CollectedAt:   timestamp,
					Name:          models.MetricNameDisk,
					Unit:          models.UnitMegaBytes,
					Value:         "4",
					Timestamp:     1111,
				}))
			})
		})
	})

	Describe("#CompactEnvelopes", func() {
		BeforeEach(func() {
			envelopes = append(envelopes, generateMemoryContainerMetrics("test-app-id", "0", 10*1024*1024, 1111))
			envelopes = append(envelopes, generateMemoryQuotaContainerMetrics("test-app-id", "0", 20*1024*1024, 1111))
			envelopes = append(envelopes, generateCPUContainerMetrics("test-app-id", "0", 20*1024*1024, 1111))
			envelopes = append(envelopes, generateMemoryQuotaContainerMetrics("test-app-id", "1", 10.2, 1111))
		})

		It("Should return a list of envelopes with matching timestamp, source_id and instance_id ", func() {
			expectedEnvelopes := processor.CompactEnvelopes(envelopes)
			Expect(len(expectedEnvelopes)).To(Equal(2))

			for i := range expectedEnvelopes {
				expectedEnvelopeMetric := expectedEnvelopes[i].GetGauge().GetMetrics()
				expectedEnvelopeMetricKeys := maps.Keys(expectedEnvelopeMetric)

				switch len(expectedEnvelopeMetric) {
				case 3:
					Expect(expectedEnvelopeMetricKeys).To(ContainElement("cpu"))
					Expect(expectedEnvelopeMetricKeys).To(ContainElement("memory"))
					Expect(expectedEnvelopeMetricKeys).To(ContainElement("memory_quota"))

				case 1:
					Expect(expectedEnvelopeMetricKeys).To(ContainElement("memory_quota"))
				}

			}
		})
	})
})

func generateMetrics(sourceID string, instance string, metrics map[string]*loggregator_v2.GaugeValue, timestamp int64) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId:   sourceID,
		InstanceId: instance,
		Message: &loggregator_v2.Envelope_Gauge{
			Gauge: &loggregator_v2.Gauge{
				Metrics: metrics,
			},
		},
		Timestamp: timestamp,
	}
}

// generateContainerMetricsEnvelope1 generates an envelope according to https://github.com/cloudfoundry/docs-loggregator/blob/7a864253547a84bd60a3192e9f5b409013f38a37/container-metrics.html.md.erb#L122
func generateContainerMetricsEnvelope1(sourceID, instance string, cpu, disk, diskQuota, memory, memoryQuota float64, timestamp int64) *loggregator_v2.Envelope {
	return generateMetrics(sourceID, instance, map[string]*loggregator_v2.GaugeValue{
		"cpu": {
			Unit:  "percentage",
			Value: cpu,
		},
		"disk": {
			Unit:  "bytes",
			Value: disk,
		},
		"disk_quota": {
			Unit:  "bytes",
			Value: diskQuota,
		},
		"memory": {
			Unit:  "bytes",
			Value: memory,
		},
		"memory_quota": {
			Unit:  "bytes",
			Value: memoryQuota,
		},
	}, timestamp)
}

// generateContainerMetricsEnvelope2 generates an envelope according to https://github.com/cloudfoundry/docs-loggregator/blob/7a864253547a84bd60a3192e9f5b409013f38a37/container-metrics.html.md.erb#L123
func generateContainerMetricsEnvelope2(sourceID, instance string, cpuEntitlement, containerAge float64, timestamp int64) *loggregator_v2.Envelope {
	return generateMetrics(sourceID, instance, map[string]*loggregator_v2.GaugeValue{
		"cpu_entitlement": {
			Unit:  "percentage",
			Value: cpuEntitlement,
		},
		"container_age": {
			Unit:  "nanoseconds",
			Value: containerAge,
		},
	}, timestamp)
}

func generateMemoryContainerMetrics(sourceID, instance string, memory float64, timestamp int64) *loggregator_v2.Envelope {
	return generateMetrics(sourceID, instance, map[string]*loggregator_v2.GaugeValue{
		"memory": {
			Unit:  "bytes",
			Value: memory,
		},
	}, timestamp)
}

func generateMemoryQuotaContainerMetrics(sourceID, instance string, memoryQuota float64, timestamp int64) *loggregator_v2.Envelope {
	return generateMetrics(sourceID, instance, map[string]*loggregator_v2.GaugeValue{
		"memory_quota": {
			Unit:  "bytes",
			Value: memoryQuota,
		},
	}, timestamp)
}
func generateCPUContainerMetrics(sourceID, instance string, cpu float64, timestamp int64) *loggregator_v2.Envelope {
	return generateMetrics(sourceID, instance, map[string]*loggregator_v2.GaugeValue{
		"cpu": {
			Unit:  "percentage",
			Value: cpu,
		},
	}, timestamp)
}

func generateCPUEntitlementContainerMetrics(sourceID, instance string, cpuEntitlement float64, timestamp int64) *loggregator_v2.Envelope {
	return generateMetrics(sourceID, instance, map[string]*loggregator_v2.GaugeValue{
		"cpu_entitlement": {
			Unit:  "percentage",
			Value: cpuEntitlement,
		},
	}, timestamp)
}

func generateDiskContainerMetrics(sourceID, instance string, disk float64, timestamp int64) *loggregator_v2.Envelope {
	return generateMetrics(sourceID, instance, map[string]*loggregator_v2.GaugeValue{
		"disk": {
			Unit:  "bytes",
			Value: disk,
		},
	}, timestamp)
}

func generateDiskQuotaContainerMetrics(sourceID, instance string, diskQuota float64, timestamp int64) *loggregator_v2.Envelope {
	return generateMetrics(sourceID, instance, map[string]*loggregator_v2.GaugeValue{
		"disk_quota": {
			Unit:  "bytes",
			Value: diskQuota,
		},
	}, timestamp)
}

func generateCustomMetrics(sourceID, instance, name, unit string, value float64, timestamp int64) *loggregator_v2.Envelope {
	return generateMetrics(sourceID, instance, map[string]*loggregator_v2.GaugeValue{
		name: {
			Unit:  unit,
			Value: value,
		},
	}, timestamp)
}

const MiB = 1024 * 1024
