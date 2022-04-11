package healthendpoint_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("HTTPStatusCollector", func() {
	var (
		httpStatusCollector HTTPStatusCollector
		namespace           = "test_name_space"
		subSystem           = "test_sub_system"
		descChan            chan *prometheus.Desc
		metricChan          chan prometheus.Metric
		//describe
		concurrentHTTPRequestDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subSystem, "concurrent_http_request"),
			"Number of concurrent http request",
			nil,
			nil,
		)
		// metrics
		concurrentHTTPRequestMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subSystem,
				Name:      "concurrent_http_request",
				Help:      "Number of concurrent http request",
			})
	)
	BeforeEach(func() {
		httpStatusCollector = NewHTTPStatusCollector(namespace, subSystem)
		descChan = make(chan *prometheus.Desc, 10)
		metricChan = make(chan prometheus.Metric, 100)

	})
	Context("Describe", func() {
		BeforeEach(func() {
			httpStatusCollector.Describe(descChan)
		})
		It("Receive descs", func() {
			Eventually(descChan).Should(Receive(&concurrentHTTPRequestDesc))
		})
	})

	Context("Collect", func() {
		Context("IncConcurrentHTTPRequest", func() {
			BeforeEach(func() {
				var count = 100
				for i := 0; i < count; i++ {
					httpStatusCollector.IncConcurrentHTTPRequest()
				}
				concurrentHTTPRequestMetric.Set(float64(count))
				httpStatusCollector.Collect(metricChan)
			})
			It("Receive metrics", func() {
				Eventually(metricChan).Should(Receive(Equal(concurrentHTTPRequestMetric)))
			})
		})

		Context("DecConcurrentHTTPRequest", func() {
			BeforeEach(func() {
				var count = 100
				for i := 0; i < count; i++ {
					httpStatusCollector.DecConcurrentHTTPRequest()
				}
				concurrentHTTPRequestMetric.Set(float64(0 - count))
				httpStatusCollector.Collect(metricChan)

			})
			It("Receive metrics", func() {
				Eventually(metricChan).Should(Receive(Equal(concurrentHTTPRequestMetric)))
			})
		})

	})
})
