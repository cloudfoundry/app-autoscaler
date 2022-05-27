package client_test

import (
	"net/http"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/client"
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
		msc = NewMetricServerClient(logger, "https://some-metric-server-url/", &http.Client{})
	})

	It("Can be assign to a Metric Client type", func() {
		Expect(msc).To(BeAssignableToTypeOf(&MetricServerClient{}))
	})
})
