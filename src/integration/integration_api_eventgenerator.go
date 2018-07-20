package integration

import (
	"autoscaler/cf"
	"autoscaler/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

type AppAggregatedMetricResult struct {
	TotalResults int                `json:"total_results"`
	TotalPages   int                `json:"total_pages"`
	Page         int                `json:"page"`
	Resources    []models.AppMetric `json:"resources"`
}

var _ = Describe("Integration_Api_EventGenerator", func() {
	var (
		appId             string
		pathVariables     []string
		parameters        map[string]string
		metric            *models.AppMetric
		metricType        string = "memoryused"
		initInstanceCount int    = 2
	)

	BeforeEach(func() {
		startFakeCCNOAAUAA(initInstanceCount)
		initializeHttpClient("api.crt", "api.key", "autoscaler-ca.crt", apiEventGeneratorHttpRequestTimeout)
		initializeHttpClientForPublicApi("api_public.crt", "api_public.key", "autoscaler-ca.crt", apiEventGeneratorHttpRequestTimeout)

		eventGeneratorConfPath = components.PrepareEventGeneratorConfig(dbUrl, components.Ports[EventGenerator], fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), aggregatorExecuteInterval, policyPollerInterval, saveInterval, evaluationManagerInterval, tmpDir)
		startEventGenerator()
		apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], components.Ports[APIPublicServer], false, fakeCCNOAAUAA.URL(), dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ServiceBrokerInternal]), true, tmpDir)
		startApiServer()
		appId = getRandomId()
		pathVariables = []string{appId, metricType}

	})

	AfterEach(func() {
		stopApiServer()
		stopEventGenerator()
	})
	Describe("Get App Metrics", func() {

		Context("Cloud Controller api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
			})
			It("should error with status code 500", func() {
				By("check public api")
				checkResponseContentWithParameters(getAppAggregatedMetrics, pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{}, PUBLIC)
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
			})
			It("should error with status code 500", func() {
				By("check public api")
				checkResponseContentWithParameters(getAppAggregatedMetrics, pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{}, PUBLIC)
			})
		})
		Context("UAA api returns 401", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
					cf.Endpoints{
						AuthEndpoint:    fakeCCNOAAUAA.URL(),
						DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
					}))
				fakeCCNOAAUAA.RouteToHandler("GET", "/userinfo", ghttp.RespondWithJSONEncoded(http.StatusUnauthorized, struct{}{}))
			})
			It("should error with status code 401", func() {
				By("check public api")
				checkResponseContentWithParameters(getAppAggregatedMetrics, pathVariables, parameters, http.StatusUnauthorized, map[string]interface{}{}, PUBLIC)
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
			})
			It("should error with status code 401", func() {
				By("check public api")
				checkResponseContentWithParameters(getAppAggregatedMetrics, pathVariables, parameters, http.StatusUnauthorized, map[string]interface{}{}, PUBLIC)
			})
		})

		Context("EventGenerator is down", func() {
			JustBeforeEach(func() {
				stopEventGenerator()
			})

			It("should error with status code 500", func() {
				By("check internal api")
				checkResponseContentWithParameters(getAppAggregatedMetrics, pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{"error": fmt.Sprintf("connect ECONNREFUSED 127.0.0.1:%d", components.Ports[EventGenerator])}, INTERNAL)
				By("check public api")
				checkResponseContentWithParameters(getAppAggregatedMetrics, pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{"error": fmt.Sprintf("connect ECONNREFUSED 127.0.0.1:%d", components.Ports[EventGenerator])}, PUBLIC)
			})
		})

		Context("Get aggregated metrics", func() {
			BeforeEach(func() {
				metric = &models.AppMetric{
					AppId:      appId,
					MetricType: models.MetricNameMemoryUsed,
					Unit:       models.UnitMegaBytes,
					Value:      "123456",
				}

				metric.Timestamp = 666666
				insertAppMetric(metric)

				metric.Timestamp = 555555
				insertAppMetric(metric)

				metric.Timestamp = 555555
				insertAppMetric(metric)

				metric.Timestamp = 333333
				insertAppMetric(metric)

				metric.Timestamp = 444444
				insertAppMetric(metric)

				//add some other metric-type
				metric.MetricType = models.MetricNameThroughput
				metric.Unit = models.UnitNum
				metric.Timestamp = 444444
				insertAppMetric(metric)
				//add some  other appId
				metric.AppId = "some-other-app-id"
				metric.MetricType = models.MetricNameMemoryUsed
				metric.Unit = models.UnitMegaBytes
				metric.Timestamp = 444444
				insertAppMetric(metric)
			})
			It("should get the metrics ", func() {
				By("get the 1st page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "metric-type": metricType, "order": "asc", "page": "1", "results-per-page": "2"}
				result := AppAggregatedMetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         1,
					Resources: []models.AppMetric{
						models.AppMetric{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  333333,
						},
						models.AppMetric{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  444444,
						},
					},
				}
				By("check internal api")
				checkAggregatedMetricResult(pathVariables, parameters, result, INTERNAL)
				By("check public api")
				checkAggregatedMetricResult(pathVariables, parameters, result, PUBLIC)

				By("get the 2nd page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "metric-type": metricType, "order": "asc", "page": "2", "results-per-page": "2"}
				result = AppAggregatedMetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         2,
					Resources: []models.AppMetric{
						models.AppMetric{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  555555,
						},
						models.AppMetric{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  555555,
						},
					},
				}
				By("check internal api")
				checkAggregatedMetricResult(pathVariables, parameters, result, INTERNAL)
				By("check public api")
				checkAggregatedMetricResult(pathVariables, parameters, result, PUBLIC)

				By("get the 3rd page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "metric-type": metricType, "order": "asc", "page": "3", "results-per-page": "2"}
				result = AppAggregatedMetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         3,
					Resources: []models.AppMetric{
						models.AppMetric{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  666666,
						},
					},
				}
				By("check internal api")
				checkAggregatedMetricResult(pathVariables, parameters, result, INTERNAL)
				By("check public api")
				checkAggregatedMetricResult(pathVariables, parameters, result, PUBLIC)

				By("the 4th page should be empty")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "metric-type": metricType, "order": "asc", "page": "4", "results-per-page": "2"}
				result = AppAggregatedMetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         4,
					Resources:    []models.AppMetric{},
				}
				By("check internal api")
				checkAggregatedMetricResult(pathVariables, parameters, result, INTERNAL)
				By("check public api")
				checkAggregatedMetricResult(pathVariables, parameters, result, PUBLIC)
			})

		})
	})
})

func checkAggregatedMetricResult(pathVariables []string, parameters map[string]string, result AppAggregatedMetricResult, apiType APIType) {
	var actual AppAggregatedMetricResult
	resp, err := getAppAggregatedMetrics(pathVariables, parameters, apiType)
	defer resp.Body.Close()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred())
	Expect(actual).To(Equal(result))

}
