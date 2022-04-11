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

var _ = Describe("Integration_GolangApi_MetricsServer", func() {
	var (
		appId             string
		pathVariables     []string
		parameters        map[string]string
		metric            *models.AppInstanceMetric
		metricType        = "memoryused"
		initInstanceCount = 2
	)

	BeforeEach(func() {
		startFakeCCNOAAUAA(initInstanceCount)
		fakeMetricsPolling(appId, 400*1024*1024, 600*1024*1024)
		initializeHttpClient("api.crt", "api.key", "autoscaler-ca.crt", apiMetricsServerHttpRequestTimeout)
		initializeHttpClientForPublicApi("api_public.crt", "api_public.key", "autoscaler-ca.crt", apiMetricsServerHttpRequestTimeout)
		metricsServerConfPath = components.PrepareMetricsServerConfig(dbUrl, defaultHttpClientTimeout, components.Ports[MetricsServerHTTP], components.Ports[MetricsServerWS], tmpDir)
		startMetricsServer()

		golangApiServerConfPath = components.PrepareGolangApiServerConfig(
			dbUrl,
			components.Ports[GolangAPIServer],
			components.Ports[GolangServiceBroker],
			fakeCCNOAAUAA.URL(),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsServerHTTP]),
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
		stopMetricsServer()
	})
	Describe("Get metrics", func() {

		Context("Cloud Controller api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "asc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 500", func() {
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, components.Ports[GolangAPIServer], pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{
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
						AuthEndpoint:    fakeCCNOAAUAA.URL(),
						TokenEndpoint:   fakeCCNOAAUAA.URL(),
						DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
					}))
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "asc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 500", func() {
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, components.Ports[GolangAPIServer], pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{
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
						AuthEndpoint:    fakeCCNOAAUAA.URL(),
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
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "asc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 401", func() {
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, components.Ports[GolangAPIServer],
					pathVariables, parameters, http.StatusUnauthorized, map[string]interface{}{
						"code":    "Unauthorized",
						"message": "You are not authorized to perform the requested action",
					})
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
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "asc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 401", func() {
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, components.Ports[GolangAPIServer],
					pathVariables, parameters, http.StatusUnauthorized, map[string]interface{}{
						"code":    "Unauthorized",
						"message": "You are not authorized to perform the requested action",
					})
			})
		})

		Context("MetricsServer is down", func() {
			JustBeforeEach(func() {
				stopMetricsServer()
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "asc", "page": "1", "results-per-page": "5"}
			})

			It("should error with status code 500", func() {
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, components.Ports[GolangAPIServer], pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{
					"code":    "Internal Server Error",
					"message": "Error retrieving metrics history from metricscollector",
				})

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
				metric.AppId = getRandomId()
				metric.Name = models.MetricNameMemoryUsed
				metric.Unit = models.UnitMegaBytes
				metric.Timestamp = 444444
				metric.InstanceIndex = 1
				insertAppInstanceMetric(metric)
			})
			Context("instance-index is not provided", func() {
				It("should get the metrics of all instances", func() {
					By("get the 1st page")
					parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "1", "results-per-page": "2"}
					result := AppInstanceMetricResult{
						TotalResults: 5,
						TotalPages:   3,
						Page:         1,
						NextUrl:      getInstanceMetricsUrl(appId, metricType, parameters, 2),
						Resources: []models.AppInstanceMetric{
							{
								AppId:         appId,
								InstanceIndex: 0,
								CollectedAt:   111111,
								Name:          models.MetricNameMemoryUsed,
								Unit:          models.UnitMegaBytes,
								Value:         "123456",
								Timestamp:     333333,
							},
							{
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
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

					By("get the 2nd page")
					parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "2", "results-per-page": "2"}
					result = AppInstanceMetricResult{
						TotalResults: 5,
						TotalPages:   3,
						Page:         2,
						PrevUrl:      getInstanceMetricsUrl(appId, metricType, parameters, 1),
						NextUrl:      getInstanceMetricsUrl(appId, metricType, parameters, 3),
						Resources: []models.AppInstanceMetric{
							{
								AppId:         appId,
								InstanceIndex: 0,
								CollectedAt:   111111,
								Name:          models.MetricNameMemoryUsed,
								Unit:          models.UnitMegaBytes,
								Value:         "123456",
								Timestamp:     555555,
							},
							{
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
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

					By("get the 3rd page")
					parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "3", "results-per-page": "2"}
					result = AppInstanceMetricResult{
						TotalResults: 5,
						TotalPages:   3,
						Page:         3,
						PrevUrl:      getInstanceMetricsUrl(appId, metricType, parameters, 2),
						Resources: []models.AppInstanceMetric{
							{
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
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

					By("the 4th page should be empty")
					parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "4", "results-per-page": "2"}
					result = AppInstanceMetricResult{
						TotalResults: 5,
						TotalPages:   3,
						Page:         4,
						PrevUrl:      getInstanceMetricsUrl(appId, metricType, parameters, 3),
						Resources:    []models.AppInstanceMetric{},
					}
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

					By("the 5th page should be empty")
					parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "5", "results-per-page": "2"}
					result = AppInstanceMetricResult{
						TotalResults: 5,
						TotalPages:   3,
						Page:         5,
						Resources:    []models.AppInstanceMetric{},
					}
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)
				})
				It("should get the metrics of all instances in specified time scope", func() {
					By("get the results from 555555")
					parameters = map[string]string{"start-time": "555555", "order-direction": "asc", "page": "1", "results-per-page": "10"}
					result := AppInstanceMetricResult{
						TotalResults: 3,
						TotalPages:   1,
						Page:         1,
						Resources: []models.AppInstanceMetric{
							{
								AppId:         appId,
								InstanceIndex: 0,
								CollectedAt:   111111,
								Name:          models.MetricNameMemoryUsed,
								Unit:          models.UnitMegaBytes,
								Value:         "123456",
								Timestamp:     555555,
							},
							{
								AppId:         appId,
								InstanceIndex: 1,
								CollectedAt:   111111,
								Name:          models.MetricNameMemoryUsed,
								Unit:          models.UnitMegaBytes,
								Value:         "123456",
								Timestamp:     555555,
							},
							{
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
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

					By("get the results to 444444")
					parameters = map[string]string{"end-time": "444444", "order-direction": "asc", "page": "1", "results-per-page": "10"}
					result = AppInstanceMetricResult{
						TotalResults: 2,
						TotalPages:   1,
						Page:         1,
						Resources: []models.AppInstanceMetric{
							{
								AppId:         appId,
								InstanceIndex: 0,
								CollectedAt:   111111,
								Name:          models.MetricNameMemoryUsed,
								Unit:          models.UnitMegaBytes,
								Value:         "123456",
								Timestamp:     333333,
							},
							{
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
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

					By("get the results from 444444 to 555555")
					parameters = map[string]string{"start-time": "444444", "end-time": "555555", "order-direction": "asc", "page": "1", "results-per-page": "10"}
					result = AppInstanceMetricResult{
						TotalResults: 3,
						TotalPages:   1,
						Page:         1,
						Resources: []models.AppInstanceMetric{
							{
								AppId:         appId,
								InstanceIndex: 1,
								CollectedAt:   111111,
								Name:          models.MetricNameMemoryUsed,
								Unit:          models.UnitMegaBytes,
								Value:         "123456",
								Timestamp:     444444,
							},
							{
								AppId:         appId,
								InstanceIndex: 0,
								CollectedAt:   111111,
								Name:          models.MetricNameMemoryUsed,
								Unit:          models.UnitMegaBytes,
								Value:         "123456",
								Timestamp:     555555,
							},
							{
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
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)
				})
			})
			Context("instance-index is provided", func() {
				It("should get the metrics of the instance", func() {
					By("get the 1st page")
					parameters = map[string]string{"instance-index": "1", "start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "1", "results-per-page": "2"}
					result := AppInstanceMetricResult{
						TotalResults: 2,
						TotalPages:   1,
						Page:         1,
						Resources: []models.AppInstanceMetric{
							{
								AppId:         appId,
								InstanceIndex: 1,
								CollectedAt:   111111,
								Name:          models.MetricNameMemoryUsed,
								Unit:          models.UnitMegaBytes,
								Value:         "123456",
								Timestamp:     444444,
							},
							{
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
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

					By("get the 2nd page")
					parameters = map[string]string{"instance-index": "1", "start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "2", "results-per-page": "2"}
					result = AppInstanceMetricResult{
						TotalResults: 2,
						TotalPages:   1,
						Page:         2,
						PrevUrl:      getInstanceMetricsUrlWithInstanceIndex(appId, metricType, parameters, 1),
						Resources:    []models.AppInstanceMetric{},
					}
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)
				})
				It("should get the metrics of the instance in specified time scope", func() {
					By("get the results from 555555")
					parameters = map[string]string{"instance-index": "1", "start-time": "555555", "order-direction": "asc", "page": "1", "results-per-page": "10"}
					result := AppInstanceMetricResult{
						TotalResults: 1,
						TotalPages:   1,
						Page:         1,
						Resources: []models.AppInstanceMetric{
							{
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
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

					By("get the results to 444444")
					parameters = map[string]string{"instance-index": "1", "end-time": "444444", "order-direction": "asc", "page": "1", "results-per-page": "10"}
					result = AppInstanceMetricResult{
						TotalResults: 1,
						TotalPages:   1,
						Page:         1,
						Resources: []models.AppInstanceMetric{
							{
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
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

					By("get the results from 444444 to 555555")
					parameters = map[string]string{"instance-index": "1", "start-time": "444444", "end-time": "555555", "order-direction": "asc", "page": "1", "results-per-page": "10"}
					result = AppInstanceMetricResult{
						TotalResults: 2,
						TotalPages:   1,
						Page:         1,
						Resources: []models.AppInstanceMetric{
							{
								AppId:         appId,
								InstanceIndex: 1,
								CollectedAt:   111111,
								Name:          models.MetricNameMemoryUsed,
								Unit:          models.UnitMegaBytes,
								Value:         "123456",
								Timestamp:     444444,
							},
							{
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
					checkAppInstanceMetricResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)
				})
			})

		})
	})
})
