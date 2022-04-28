package aggregator_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"net/http"
)

var _ = Describe("MetricClient", func() {
	var (
		mc            MetricClient
		config        Config
		metricsServer *ghttp.Server
	)

	BeforeEach(func() {
		config = Config{}
		metricsServer = ghttp.NewUnstartedServer()
	})

	JustBeforeEach(func() {
		mc = NewMetricClient(config)
	})

	It("LogCache is disable by default", func() {
		Expect(mc.EnableLogCache()).To(BeFalse())
	})

	Describe("When LogCache is enabled", func() {
		var logCacheResponse LogCacheResponse

		BeforeEach(func() {
			logCacheResponse = LogCacheResponse{}
			config.UseLogCache = true
			config.MetricCollector.MetricCollectorURL = "https://log-cache.some-sys-domain.com"
		})

		JustBeforeEach(func() {
			metricsServer.RouteToHandler("GET", config.MetricCollector.MetricCollectorURL,
				ghttp.RespondWithJSONEncoded(http.StatusOK, &logCacheResponse))
		})

		It("creates request with log cache url", func() {
			Expect(mc.GetAddrs()).To(BeEquivalentTo("https://log-cache.some-sys-domain.com"))
			Expect(mc.EnableLogCache()).To(BeTrue())
		})

	})
})
