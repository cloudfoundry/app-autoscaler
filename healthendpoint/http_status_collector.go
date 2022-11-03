package healthendpoint

import (
	"github.com/prometheus/client_golang/prometheus"
)

type HTTPStatusCollector interface {
	prometheus.Collector
	IncConcurrentHTTPRequest()
	DecConcurrentHTTPRequest()
}
type httpStatusCollector struct {
	concurrentHTTPRequestGuage prometheus.Gauge
}

func NewHTTPStatusCollector(namespace, subSystem string) HTTPStatusCollector {
	return &httpStatusCollector{
		concurrentHTTPRequestGuage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subSystem,
				Name:      "concurrent_http_request",
				Help:      "Number of concurrent http request",
			}),
	}
}

func (c *httpStatusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.concurrentHTTPRequestGuage.Desc()
}

func (c *httpStatusCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- c.concurrentHTTPRequestGuage
}

func (c *httpStatusCollector) IncConcurrentHTTPRequest() {
	c.concurrentHTTPRequestGuage.Inc()
}
func (c *httpStatusCollector) DecConcurrentHTTPRequest() {
	c.concurrentHTTPRequestGuage.Dec()
}
