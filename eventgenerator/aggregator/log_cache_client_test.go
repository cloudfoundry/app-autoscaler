package aggregator_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("LogCacheClient", func() {
	var ()
	var (
		msc    *LogCacheClient
		logger lager.Logger
	)

	Describe("GetMetric", func() {
		It("retrive metrics from logCache", func() {
			logCacheGoClient := newStubLogCacheGoClient()
			logger = lagertest.NewTestLogger("MetricPoller-test")
			msc = NewLogCacheClient(logger, "https://some-metric-server-url/", &models.TLSCerts{})

		})

	})
})

type stubLogCacheGoClient struct {
	logcache.Client
}

func newStubLogCacheGoClient() *stubGrpcLogCache {
	s := &stubLogCacheGoClient{}

	return s
}
