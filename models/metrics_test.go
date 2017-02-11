package models_test

import (
	. "autoscaler/models"

	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newContainerEnvelope(timestamp int64, appId string, index int32, cpu float64, memory uint64, disk uint64) *events.Envelope {
	return &events.Envelope{
		Timestamp: &timestamp,
		ContainerMetric: &events.ContainerMetric{
			ApplicationId: &appId,
			InstanceIndex: &index,
			CpuPercentage: &cpu,
			MemoryBytes:   &memory,
			DiskBytes:     &disk,
		},
	}
}

var _ = Describe("Metrics", func() {

	Describe("GetInstanceMemoryMetricFromContainerEnvelopes", func() {
		var (
			containerEnvelops []*events.Envelope
			metrics           []*AppInstanceMetric
		)

		JustBeforeEach(func() {
			metrics = GetInstanceMemoryMetricFromContainerEnvelopes(123456, "an-app-id", containerEnvelops)
		})

		Context("when metrics are empty", func() {
			BeforeEach(func() {
				containerEnvelops = []*events.Envelope{}
			})

			It("should return empty instance memory metrics", func() {
				Expect(metrics).To(BeEmpty())
			})
		})

		Context("when no metric is available for the given app", func() {
			BeforeEach(func() {
				containerEnvelops = []*events.Envelope{
					newContainerEnvelope(111111, "different-app-id", 0, 12.11, 6222220, 233300000),
					newContainerEnvelope(222222, "different-app-id", 1, 31.21, 2366200, 3424553333),
					newContainerEnvelope(333333, "another-different-app-id", 0, 0.211, 88623692, 9876384949),
				}
			})

			It("should return empty instance memory metrics", func() {
				Expect(metrics).To(BeEmpty())
			})
		})

		Context("when metrics from both given app and other apps", func() {
			BeforeEach(func() {
				containerEnvelops = []*events.Envelope{
					newContainerEnvelope(111111, "an-app-id", 0, 12.11, 622222, 233300000),
					newContainerEnvelope(222222, "different-app-id", 2, 0.211, 2366200, 9876384949),
					newContainerEnvelope(333333, "an-app-id", 1, 31.21, 88623692, 3424553333),
				}
			})

			It("should return instance memory metrics from given app", func() {
				Expect(metrics).To(ConsistOf(
					&AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   123456,
						Name:          MetricNameMemory,
						Unit:          UnitMegaBytes,
						Value:         "1",
						Timestamp:     111111,
					},
					&AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 1,
						CollectedAt:   123456,
						Name:          MetricNameMemory,
						Unit:          UnitMegaBytes,
						Value:         "85",
						Timestamp:     333333,
					},
				))
			})
		})
	})

})
