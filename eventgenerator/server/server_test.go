package server_test

import (
	"net/http"
	"net/url"
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon_v2"
)

var _ = Describe("Server", func() {
	var (
		rsp                 *http.Response
		err                 error
		serverProcess       ifrit.Process
		serverUrl           *url.URL
		policyDB            *fakes.FakePolicyDB
		httpStatusCollector *fakes.FakeHTTPStatusCollector

		appMetricDB     *fakes.FakeAppMetricDB
		conf            *config.Config
		queryAppMetrics aggregator.QueryAppMetricsFunc
	)

	BeforeEach(func() {
		port := 1111 + GinkgoParallelProcess()
		conf = &config.Config{
			Server: config.ServerConfig{
				ServerConfig: helpers.ServerConfig{
					Port: port,
				},
			},
		}

		serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
		Expect(err).ToNot(HaveOccurred())

		queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
			return nil, nil
		}

		httpStatusCollector = &fakes.FakeHTTPStatusCollector{}
		policyDB = &fakes.FakePolicyDB{}
		appMetricDB = &fakes.FakeAppMetricDB{}

	})

	AfterEach(func() {
		ginkgomon_v2.Interrupt(serverProcess)
	})

	JustBeforeEach(func() {
		httpServer, err := server.NewServer(lager.NewLogger("test"), conf, appMetricDB, policyDB, queryAppMetrics, httpStatusCollector).GetMtlsServer()
		Expect(err).NotTo(HaveOccurred())
		serverProcess = ginkgomon_v2.Invoke(httpServer)
	})

	Describe("request on /v1/apps/an-app-id/aggregated_metric_histories/a-metric-type", func() {
		BeforeEach(func() {
			serverUrl.Path = "/v1/apps/an-app-id/aggregated_metric_histories/a-metric-type"
		})

		JustBeforeEach(func() {
			rsp, err = http.Get(serverUrl.String())
		})

		It("should return 200", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			rsp.Body.Close()
		})
		When("using wrong method to retrieve aggregared metrics history", func() {
			JustBeforeEach(func() {
				rsp, err = http.Post(serverUrl.String(), "garbage", nil)
			})

			It("should return 405", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
				rsp.Body.Close()
			})
		})
	})

	When("requesting the wrong path", func() {
		BeforeEach(func() {
			serverUrl.Path = "/not-exist-path"
		})

		JustBeforeEach(func() {
			rsp, err = http.Get(serverUrl.String())
		})

		It("should return 404", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
			rsp.Body.Close()
		})

	})
})
