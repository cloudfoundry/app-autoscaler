package integration_test

import (
	"fmt"
	"net/http"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Integration_GolangApi_EventGenerator", func() {
	var (
		appId             string
		pathVariables     []string
		parameters        map[string]string
		metric            *models.AppMetric
		metricType        = "memoryused"
		initInstanceCount = 2
	)

	BeforeEach(func() {
		startFakeCCNOAAUAA(initInstanceCount)
		initializeHttpClient("api.crt", "api.key", "autoscaler-ca.crt", apiEventGeneratorHttpRequestTimeout)
		initializeHttpClientForPublicApi("api_public.crt", "api_public.key", "autoscaler-ca.crt", apiEventGeneratorHttpRequestTimeout)

		eventGeneratorConfPath = components.PrepareEventGeneratorConfig(dbUrl, components.Ports[EventGenerator], fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), aggregatorExecuteInterval, policyPollerInterval, saveInterval, evaluationManagerInterval, defaultHttpClientTimeout, tmpDir)
		startEventGenerator()
		golangApiServerConfPath = components.PrepareGolangApiServerConfig(
			dbUrl,
			components.Ports[GolangAPIServer],
			components.Ports[GolangServiceBroker],
			fakeCCNOAAUAA.URL(),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]),
			"https://127.0.0.1:8888",
			true,
			tmpDir)
		startGolangApiServer()
		appId = getRandomId()
		pathVariables = []string{appId, metricType}

	})

	AfterEach(func() {
		stopGolangApiServer()
		stopEventGenerator()
	})
	Describe("Get App Metrics", func() {

		Context("Cloud Controller api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
			})
			It("should error with status code 500", func() {
				checkPublicAPIResponseContentWithParameters(getAppAggregatedMetrics, components.Ports[GolangAPIServer], pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{
					"code":    "Internal-Server-Error",
					"message": "Failed to check if user is admin",
				})
			})
		})

		Context("UAA api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
					cf.Endpoints{
						TokenEndpoint:   fakeCCNOAAUAA.URL(),
						DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
					}))
			})
			It("should error with status code 500", func() {
				checkPublicAPIResponseContentWithParameters(getAppAggregatedMetrics, components.Ports[GolangAPIServer], pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{
					"code":    "Internal-Server-Error",
					"message": "Failed to check if user is admin",
				})
			})
		})
		Context("UAA api returns 401", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
					cf.Endpoints{
						TokenEndpoint:   fakeCCNOAAUAA.URL(),
						DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
					}))
				fakeCCNOAAUAA.RouteToHandler("POST", "/check_token", ghttp.RespondWithJSONEncoded(http.StatusOK,
					struct {
						Scope []string `json:"scope"`
					}{
						[]string{"cloud_controller.read", "cloud_controller.write", "password.write", "openid", "network.admin", "network.write", "uaa.user"},
					}))
				fakeCCNOAAUAA.RouteToHandler("GET", "/userinfo", ghttp.RespondWithJSONEncoded(http.StatusUnauthorized, struct{}{}))
			})
			It("should error with status code 401", func() {
				checkPublicAPIResponseContentWithParameters(getAppAggregatedMetrics, components.Ports[GolangAPIServer], pathVariables,
					parameters, http.StatusUnauthorized, map[string]interface{}{
						"code":    "Unauthorized",
						"message": "You are not authorized to perform the requested action"})
			})
		})

		Context("Check permission not passed", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.RouteToHandler("GET", rolesRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
					struct {
						Pagination struct {
							Total int `json:"total_results"`
						} `json:"pagination"`
					}{}))
			})
			It("should error with status code 401", func() {
				checkPublicAPIResponseContentWithParameters(getAppAggregatedMetrics, components.Ports[GolangAPIServer],
					pathVariables, parameters, http.StatusUnauthorized, map[string]interface{}{
						"code":    "Unauthorized",
						"message": "You are not authorized to perform the requested action",
					})
			})
		})

		Context("EventGenerator is down", func() {
			JustBeforeEach(func() {
				stopEventGenerator()
			})

			It("should error with status code 500", func() {
				checkPublicAPIResponseContentWithParameters(getAppAggregatedMetrics, components.Ports[GolangAPIServer], pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{
					"code":    "Internal Server Error",
					"message": "Error retrieving metrics history from eventgenerator",
				})
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

				metric.Timestamp = 555556
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
				metric.AppId = getRandomId()
				metric.MetricType = models.MetricNameMemoryUsed
				metric.Unit = models.UnitMegaBytes
				metric.Timestamp = 444444
				insertAppMetric(metric)
			})
			It("should get the metrics ", func() {
				By("get the 1st page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "1", "results-per-page": "2"}
				result := AppAggregatedMetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         1,
					NextUrl:      getAppAggregatedMetricUrl(appId, metricType, parameters, 2),
					Resources: []models.AppMetric{
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  333333,
						},
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  444444,
						},
					},
				}
				checkAggregatedMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

				By("get the 2nd page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "2", "results-per-page": "2"}
				result = AppAggregatedMetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         2,
					PrevUrl:      getAppAggregatedMetricUrl(appId, metricType, parameters, 1),
					NextUrl:      getAppAggregatedMetricUrl(appId, metricType, parameters, 3),
					Resources: []models.AppMetric{
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  555555,
						},
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  555556,
						},
					},
				}
				checkAggregatedMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

				By("get the 3rd page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "3", "results-per-page": "2"}
				result = AppAggregatedMetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         3,
					PrevUrl:      getAppAggregatedMetricUrl(appId, metricType, parameters, 2),
					Resources: []models.AppMetric{
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  666666,
						},
					},
				}
				checkAggregatedMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

				By("the 4th page should be empty")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "4", "results-per-page": "2"}
				result = AppAggregatedMetricResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         4,
					PrevUrl:      getAppAggregatedMetricUrl(appId, metricType, parameters, 3),
					Resources:    []models.AppMetric{},
				}
				checkAggregatedMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)
			})
			It("should get the metrics in specified time scope", func() {
				By("get the results from 555555")
				parameters = map[string]string{"start-time": "555555", "order-direction": "asc", "page": "1", "results-per-page": "10"}
				result := AppAggregatedMetricResult{
					TotalResults: 3,
					TotalPages:   1,
					Page:         1,
					Resources: []models.AppMetric{
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  555555,
						},
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  555556,
						},
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  666666,
						},
					},
				}
				checkAggregatedMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

				By("get the results to 444444")
				parameters = map[string]string{"end-time": "444444", "order-direction": "asc", "page": "1", "results-per-page": "10"}
				result = AppAggregatedMetricResult{
					TotalResults: 2,
					TotalPages:   1,
					Page:         1,
					Resources: []models.AppMetric{
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  333333,
						},
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  444444,
						},
					},
				}
				checkAggregatedMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

				By("get the results from 444444 to 555556")
				parameters = map[string]string{"start-time": "444444", "end-time": "555556", "order-direction": "asc", "page": "1", "results-per-page": "10"}
				result = AppAggregatedMetricResult{
					TotalResults: 3,
					TotalPages:   1,
					Page:         1,
					Resources: []models.AppMetric{
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  444444,
						},
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  555555,
						},
						{
							AppId:      appId,
							MetricType: models.MetricNameMemoryUsed,
							Unit:       models.UnitMegaBytes,
							Value:      "123456",
							Timestamp:  555556,
						},
					},
				}
				checkAggregatedMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)
			})
		})
	})
})
