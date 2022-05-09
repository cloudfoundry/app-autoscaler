package aggregator_test

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("MetricClient", func() {
	var ()

	BeforeEach(func() {
		//config = Config{}
	})

	JustBeforeEach(func() {
	})

	//Describe("When LogCache is enabled", func() {
	//	var logCacheResponse LogCacheResponse

	//	BeforeEach(func() {
	//		logCacheResponse = LogCacheResponse{}
	//		config.UseLogCache = true
	//		config.MetricCollector.MetricCollectorURL = "https://log-cache.some-sys-domain.com"
	//	})

	//	JustBeforeEach(func() {
	//		metricsServer.RouteToHandler("GET", config.MetricCollector.MetricCollectorURL,
	//			ghttp.RespondWithJSONEncoded(http.StatusOK, &logCacheResponse))
	//	})

	//	It("creates request with log cache url", func() {
	//		Expect(mc.GetAddrs()).To(BeEquivalentTo("https://log-cache.some-sys-domain.com"))
	//		Expect(mc.EnableLogCache()).To(BeTrue())
	//	})

	//})
})
