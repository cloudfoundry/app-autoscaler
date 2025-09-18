package server_test

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/server"
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
		serverUrl     *url.URL
		server        *Server
		serverProcess ifrit.Process

		conf *config.Config

		rsp                 *http.Response
		err                 error
		policyDB            *fakes.FakePolicyDB
		httpStatusCollector *fakes.FakeHTTPStatusCollector
		xfccAuthMiddleware  *fakes.FakeXFCCAuthMiddleware

		appMetricDB     *fakes.FakeAppMetricDB
		queryAppMetrics aggregator.QueryAppMetricsFunc
	)

	BeforeEach(func() {

		conf = &config.Config{
			BaseConfig: configutil.BaseConfig{
				Health: helpers.HealthConfig{
					ServerConfig: helpers.ServerConfig{
						Port: 2222 + GinkgoParallelProcess(),
					},
					BasicAuth: models.BasicAuth{
						Username: "user",
						Password: "password",
					},
				},
				Server: helpers.ServerConfig{
					Port: 1111 + GinkgoParallelProcess(),
				},
				CFServer: helpers.ServerConfig{
					Port: 3333 + GinkgoParallelProcess(),
				},
			},
		}

		xfccAuthMiddleware = &fakes.FakeXFCCAuthMiddleware{}

		queryAppMetrics = func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
			return nil, nil
		}

		httpStatusCollector = &fakes.FakeHTTPStatusCollector{}
		policyDB = &fakes.FakePolicyDB{}
		appMetricDB = &fakes.FakeAppMetricDB{}

		server = NewServer(lager.NewLogger("test"), conf, appMetricDB, policyDB, queryAppMetrics, httpStatusCollector)
	})

	AfterEach(func() {
		ginkgomon_v2.Interrupt(serverProcess)
	})

	Describe("#CreateMTLSServer", func() {
		BeforeEach(func() {
			httpServer, err := server.CreateMtlsServer()
			Expect(err).NotTo(HaveOccurred())

			serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(conf.Server.Port))
			Expect(err).ToNot(HaveOccurred())

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

	Describe("#CreateCFServer", func() {
		BeforeEach(func() {
			xfccAuthMiddleware.XFCCAuthenticationMiddlewareReturns(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.RequestURI, "invalid-guid") {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}))
			httpServer, err := server.CreateCFServer(xfccAuthMiddleware)
			Expect(err).NotTo(HaveOccurred())
			serverProcess = ginkgomon_v2.Invoke(httpServer)
			serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(conf.CFServer.Port))
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("GET /health", func() {
			var (
				actualUsername = "user"
				actualPassword = "password"
			)

			BeforeEach(func() {
				serverUrl.Path = "/health"
			})

			JustBeforeEach(func() {
				req, err := http.NewRequest("GET", serverUrl.String(), nil)
				Expect(err).ToNot(HaveOccurred())

				req.SetBasicAuth(actualUsername, actualPassword)

				client := &http.Client{}
				rsp, err = client.Do(req)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return 200", func() {
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})

			When("auth is incorrect", func() {
				BeforeEach(func() {
					actualUsername = "wrong-username"
					actualPassword = "wrong-password"
				})

				It("should return 401", func() {
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})
		})

		Describe("GET /v1/apps/{GUID}/aggregated_metric_histories/a-metric-type", func() {
			Describe("when XFCC authentication is ok", func() {
				BeforeEach(func() {
					serverUrl.Path = "/v1/apps/valid-guid/aggregated_metric_histories/a-metric-type"
				})

				JustBeforeEach(func() {
					rsp, err = http.Get(serverUrl.String())
				})

				It("should return 200", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					rsp.Body.Close()
				})
			})

			Describe("when XFCC authentication fails", func() {
				BeforeEach(func() {
					serverUrl.Path = "/v1/apps/invalid-guid/aggregated_metric_histories/a-metric-type"
				})

				JustBeforeEach(func() {
					rsp, err = http.Get(serverUrl.String())
				})

				It("should return 401", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
					rsp.Body.Close()
				})
			})
		})
	})

})
