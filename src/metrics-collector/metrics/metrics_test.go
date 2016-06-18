package metrics_test

import (
	. "metrics-collector/metrics"

	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func NewContainerMetric(appId string, index int32, cpu float64, memory uint64, disk uint64) *events.ContainerMetric {
	return &events.ContainerMetric{
		ApplicationId: &appId,
		InstanceIndex: &index,
		CpuPercentage: &cpu,
		MemoryBytes:   &memory,
		DiskBytes:     &disk,
	}
}

var _ = Describe("Metrics", func() {
	Describe("GetMemoryMetricFromContainerMetrics", func() {
		var (
			containerMetrics []*events.ContainerMetric
			metric           *Metric
		)

		JustBeforeEach(func() {
			metric = GetMemoryMetricFromContainerMetrics("app-id", containerMetrics)
		})

		Context("when metrics is empty", func() {

			BeforeEach(func() {
				containerMetrics = []*events.ContainerMetric{}
			})

			It("should return memory metric with empty instance metrics", func() {
				Expect(metric).NotTo(BeNil())
				Expect(metric.AppId).To(Equal("app-id"))
				Expect(metric.Name).To(Equal(MEMORY_METRIC_NAME))
				Expect(len(metric.Instances)).To(Equal(0))
			})
		})

		Context("when no metric is from the given app", func() {
			BeforeEach(func() {
				containerMetrics = []*events.ContainerMetric{
					NewContainerMetric("different-app-id", 0, 12.11, 622222, 233300000),
					NewContainerMetric("different-app-id", 1, 31.21, 23662, 3424553333),
					NewContainerMetric("another-different-app-id", 0, 0.211, 88623692, 9876384949),
				}
			})

			It("should return memory metric with empty instance metrics", func() {
				Expect(metric).NotTo(BeNil())
				Expect(metric.AppId).To(Equal("app-id"))
				Expect(metric.Name).To(Equal(MEMORY_METRIC_NAME))
				Expect(len(metric.Instances)).To(Equal(0))
			})
		})

		Context("when metrics from both given app and other apps", func() {
			BeforeEach(func() {
				containerMetrics = []*events.ContainerMetric{
					NewContainerMetric("app-id", 0, 12.11, 622222, 233300000),
					NewContainerMetric("app-id", 1, 31.21, 23662, 3424553333),
					NewContainerMetric("different-app-id", 2, 0.211, 88623692, 9876384949),
				}
			})

			It("should return memory metrics from given app only", func() {
				Expect(metric).NotTo(BeNil())
				Expect(metric.AppId).To(Equal("app-id"))
				Expect(metric.Name).To(Equal(MEMORY_METRIC_NAME))
				Expect(len(metric.Instances)).To(Equal(2))
				Expect(metric.Instances).To(ConsistOf(InstanceMetric{Index: 0, Value: "622222"}, InstanceMetric{Index: 1, Value: "23662"}))
			})
		})

	})
})
