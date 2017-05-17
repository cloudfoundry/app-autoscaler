package metrics_test

import (
	. "github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/metrics"

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
	Describe("get memory metric from container metrics", func() {
		var (
			containerMetrics []*events.ContainerMetric
			metric           Metric
		)

		JustBeforeEach(func() {
			metric = GetMemoryMetricFromContainerMetrics("app-id", containerMetrics)
		})

		Context("when no metrics included", func() {

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

		Context("when metrics are not from the given app", func() {
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

		Context("when all metrics from the given app", func() {
			BeforeEach(func() {
				containerMetrics = []*events.ContainerMetric{
					NewContainerMetric("app-id", 0, 12.11, 622222, 233300000),
					NewContainerMetric("app-id", 1, 31.21, 23662, 3424553333),
					NewContainerMetric("app-id", 2, 0.211, 88623692, 9876384949),
				}
			})

			It("should return all the memory metrics", func() {
				Expect(metric).NotTo(BeNil())
				Expect(metric.AppId).To(Equal("app-id"))
				Expect(metric.Name).To(Equal(MEMORY_METRIC_NAME))
				Expect(len(metric.Instances)).To(Equal(3))
			})
		})

		Context("when metrics from both  given app and other apps", func() {
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
				Expect(metric.Instances[0].Value).To(Equal("622222"))
				Expect(metric.Instances[1].Value).To(Equal("23662"))
			})
		})

	})
})
