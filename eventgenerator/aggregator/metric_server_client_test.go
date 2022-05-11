package aggregator_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricServerClient", func() {
	var (
		msc    *MetricServerClient
		logger lager.Logger
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("MetricPoller-test")
		msc = NewMetricServerClient(logger, "https://some-metric-server-url/", &models.TLSCerts{})
	})

	It("Can be assign to a Metric Client type", func() {
		var mc MetricClient = msc
		Expect(msc).To(BeAssignableToTypeOf(mc))
	})
})
