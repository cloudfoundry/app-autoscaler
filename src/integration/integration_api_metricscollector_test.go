package integration_test

import (
	"autoscaler/cf"
	"autoscaler/metricscollector/config"
	"autoscaler/models"
	"code.cloudfoundry.org/locket"
	"encoding/json"
	"fmt"
	. "integration"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

type MetricResult struct {
	TotalResults int                        `json:"total_results"`
	TotalPages   int                        `json:"total_pages"`
	Page         int                        `json:"page"`
	Resources    []models.AppInstanceMetric `json:"resources"`
}

var _ = Describe("Integration_Api_MetricsCollector", func() {
	var (
		appId             string
		pathVariables     []string
		parameters        map[string]string
		metric            *models.AppInstanceMetric
		metricType        string = "memoryused"
		collectMethod     string = config.CollectMethodPolling
		initInstanceCount int    = 2
	)

	BeforeEach(func() {
		startFakeCCNOAAUAA(initInstanceCount)
		fakeMetricsPolling(appId, 400*1024*1024, 600*1024*1024)
		initializeHttpClient("api.crt", "api.key", "autoscaler-ca.crt", apiMetricsCollectorHttpRequestTimeout)
		initializeHttpClientForPublicApi("api_public.crt", "api_public.key", "autoscaler-ca.crt", apiMetricsCollectorHttpRequestTimeout)
		metricsCollectorConfPath = components.PrepareMetricsCollectorConfig(dbUrl, components.Ports[MetricsCollector], fakeCCNOAAUAA.URL(), cf.GrantTypePassword, collectInterval,
			refreshInterval, collectMethod, tmpDir, locket.DefaultSessionTTL, locket.RetryInterval, consulRunner.ConsulCluster())
		startMetricsCollector()

		apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], components.Ports[APIPublicServer], fakeCCNOAAUAA.URL(), dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), tmpDir)
		startApiServer()
		appId = getRandomId()
		pathVariables = []string{appId, metricType}

	})

	AfterEach(func() {
		stopApiServer()
		stopMetricsCollector()
	})
	Describe("Get metrics", func() {

		Context("Cloud Controller api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "metric-type": metricType, "order": "asc", "page": "1", "results-per-page": "5"}
			})
			It("should error when access public api", func() {
				By("check public api")
				checkResponseContentWithParameters(getAppMetrics, pathVariables, parameters, http.StatusBadRequest, map[string]interface{}{}, PUBLIC)
			})
		})

		Context("UAA api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
					cf.Endpoints{
						AuthEndpoint:    fakeCCNOAAUAA.URL(),
						DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
					}))
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "metric-type": metricType, "order": "asc", "page": "1", "results-per-page": "5"}
			})
			It("should error when access public api", func() {
				By("check public api")
				checkResponseContentWithParameters(getAppMetrics, pathVariables, parameters, http.StatusBadRequest, map[string]interface{}{}, PUBLIC)
			})
		})

		Context("Check permission not passed", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.RouteToHandler("GET", checkUserSpaceRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
					struct {
						TotalResults int `json:"total_results"`
					}{
						0,
					}))
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "metric-type": metricType, "order": "asc", "page": "1", "results-per-page": "5"}
			})
			It("should error when access public api", func() {
				By("check public api")
				checkResponseContentWithParameters(getAppMetrics, pathVariables, parameters, http.StatusUnauthorized, map[string]interface{}{}, PUBLIC)
			})
		})

		Context("MetricsCollector is down", func() {
			JustBeforeEach(func() {
				stopMetricsCollector()
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "metric-type": metricType, "order": "asc", "page": "1", "results-per-page": "5"}
			})

			It("should error", func() {
				By("check internal api")
				checkResponseContentWithParameters(getAppMetrics, pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{"description": fmt.Sprintf("connect ECONNREFUSED 127.0.0.1:%d", components.Ports[MetricsCollector])}, INTERNAL)
				By("check public api")
				checkResponseContentWithParameters(getAppMetrics, pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{"description": fmt.Sprintf("connect ECONNREFUSED 127.0.0.1:%d", components.Ports[MetricsCollector])}, PUBLIC)

			})

		})

		Context("Get metrics", func() {
			BeforeEach(func() {
				metric = &models.AppInstanceMetric{
					AppId:       appId,
					CollectedAt: 111111,
					Name:        models.MetricNameMemoryUsed,
					Unit:        models.UnitMegaBytes,
					Value:       "123456",
				}

				metric.Timestamp = 666666
				metric.InstanceIndex = 0
				insertAppInstanceMetric(metric)

				metric.Timestamp = 555555
				metric.InstanceIndex = 1
				insertAppInstanceMetric(metric)

				metric.Timestamp = 555555
				metric.InstanceIndex = 0
				insertAppInstanceMetric(metric)

				metric.Timestamp = 333333
				metric.InstanceIndex = 0
				insertAppInstanceMetric(metric)

				metric.Timestamp = 444444
				metric.InstanceIndex = 1
				insertAppInstanceMetric(metric)

				//add some other metric-type
				metric.Name = models.MetricNameThroughput
				metric.Unit = models.UnitNum
				metric.Timestamp = 444444
				metric.InstanceIndex = 1
				insertAppInstanceMetric(metric)
				//add some  other appId
				metric.AppId = "some-other-app-id"
				metric.Name = models.MetricNameMemoryUsed
				metric.Unit = models.UnitMegaBytes
				metric.Timestamp = 444444
				metric.InstanceIndex = 1
				insertAppInstanceMetric(metric)
			})
			It("should get the metrics ", func() {
				By("get the 1st page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "metric-type": metricType, "order": "asc", "page": "1", "results-per-page": "2"}
				result := MetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         1,
					Resources: []models.AppInstanceMetric{
						models.AppInstanceMetric{
							AppId:         appId,
							InstanceIndex: 0,
							CollectedAt:   111111,
							Name:          models.MetricNameMemoryUsed,
							Unit:          models.UnitMegaBytes,
							Value:         "123456",
							Timestamp:     333333,
						},
						models.AppInstanceMetric{
							AppId:         appId,
							InstanceIndex: 1,
							CollectedAt:   111111,
							Name:          models.MetricNameMemoryUsed,
							Unit:          models.UnitMegaBytes,
							Value:         "123456",
							Timestamp:     444444,
						},
					},
				}
				By("check internal api")
				checkMetricResult(pathVariables, parameters, result, INTERNAL)
				By("check public api")
				checkMetricResult(pathVariables, parameters, result, PUBLIC)

				By("get the 2nd page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "metric-type": metricType, "order": "asc", "page": "2", "results-per-page": "2"}
				result = MetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         2,
					Resources: []models.AppInstanceMetric{
						models.AppInstanceMetric{
							AppId:         appId,
							InstanceIndex: 0,
							CollectedAt:   111111,
							Name:          models.MetricNameMemoryUsed,
							Unit:          models.UnitMegaBytes,
							Value:         "123456",
							Timestamp:     555555,
						},
						models.AppInstanceMetric{
							AppId:         appId,
							InstanceIndex: 1,
							CollectedAt:   111111,
							Name:          models.MetricNameMemoryUsed,
							Unit:          models.UnitMegaBytes,
							Value:         "123456",
							Timestamp:     555555,
						},
					},
				}
				By("check internal api")
				checkMetricResult(pathVariables, parameters, result, INTERNAL)
				By("check public api")
				checkMetricResult(pathVariables, parameters, result, PUBLIC)

				By("get the 3rd page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "metric-type": metricType, "order": "asc", "page": "3", "results-per-page": "2"}
				result = MetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         3,
					Resources: []models.AppInstanceMetric{
						models.AppInstanceMetric{
							AppId:         appId,
							InstanceIndex: 0,
							CollectedAt:   111111,
							Name:          models.MetricNameMemoryUsed,
							Unit:          models.UnitMegaBytes,
							Value:         "123456",
							Timestamp:     666666,
						},
					},
				}
				By("check internal api")
				checkMetricResult(pathVariables, parameters, result, INTERNAL)
				By("check public api")
				checkMetricResult(pathVariables, parameters, result, PUBLIC)

				By("the 4th page should be empty")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "metric-type": metricType, "order": "asc", "page": "4", "results-per-page": "2"}
				result = MetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         4,
					Resources:    []models.AppInstanceMetric{},
				}
				By("check internal api")
				checkMetricResult(pathVariables, parameters, result, INTERNAL)
				By("check public api")
				checkMetricResult(pathVariables, parameters, result, PUBLIC)
			})

		})
	})
})

func checkMetricResult(pathVariables []string, parameters map[string]string, result MetricResult, apiType APIType) {
	var actual MetricResult
	resp, err := getAppMetrics(pathVariables, parameters, apiType)
	defer resp.Body.Close()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred())
	Expect(actual).To(Equal(result))

}
