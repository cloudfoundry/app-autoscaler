package integration

import (
	"autoscaler/cf"
	"autoscaler/metricscollector/config"
	"autoscaler/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

type AppInstanceMetricResult struct {
	TotalResults int                        `json:"total_results"`
	TotalPages   int                        `json:"total_pages"`
	Page         int                        `json:"page"`
	PrevUrl      string                     `json:"prev_url"`
	NextUrl      string                     `json:"next_url"`
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
			refreshInterval, saveInterval, collectMethod, tmpDir)
		startMetricsCollector()

		apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], components.Ports[APIPublicServer], false, 200, fakeCCNOAAUAA.URL(), dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ServiceBrokerInternal]), true, tmpDir)
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
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "asc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 500", func() {
				By("check public api")
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{})
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
				By("check public api")
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{})
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
				fakeCCNOAAUAA.RouteToHandler("GET", "/userinfo", ghttp.RespondWithJSONEncoded(http.StatusUnauthorized, struct{}{}))
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "asc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 401", func() {
				By("check public api")
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, pathVariables, parameters, http.StatusUnauthorized, map[string]interface{}{})
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
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "asc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 401", func() {
				By("check public api")
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, pathVariables, parameters, http.StatusUnauthorized, map[string]interface{}{})
			})
		})

		Context("MetricsCollector is down", func() {
			JustBeforeEach(func() {
				stopMetricsCollector()
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "asc", "page": "1", "results-per-page": "5"}
			})

			It("should error with status code 500", func() {
				By("check internal api")
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{"error": fmt.Sprintf("connect ECONNREFUSED 127.0.0.1:%d", components.Ports[MetricsCollector])})
				By("check public api")
				checkPublicAPIResponseContentWithParameters(getAppInstanceMetrics, pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{"error": fmt.Sprintf("connect ECONNREFUSED 127.0.0.1:%d", components.Ports[MetricsCollector])})

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

					By("check public api")
					checkAppInstanceMetricResult(pathVariables, parameters, result)

					By("get the 2nd page")
					parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "2", "results-per-page": "2"}
					result = AppInstanceMetricResult{
						TotalResults: 5,
						TotalPages:   3,
						Page:         2,
						PrevUrl:      getInstanceMetricsUrl(appId, metricType, parameters, 1),
						NextUrl:      getInstanceMetricsUrl(appId, metricType, parameters, 3),
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
					By("check public api")
					checkAppInstanceMetricResult(pathVariables, parameters, result)

					By("get the 3rd page")
					parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "3", "results-per-page": "2"}
					result = AppInstanceMetricResult{
						TotalResults: 5,
						TotalPages:   3,
						Page:         3,
						PrevUrl:      getInstanceMetricsUrl(appId, metricType, parameters, 2),
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
					By("check public api")
					checkAppInstanceMetricResult(pathVariables, parameters, result)

					By("the 4th page should be empty")
					parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "4", "results-per-page": "2"}
					result = AppInstanceMetricResult{
						TotalResults: 5,
						TotalPages:   3,
						Page:         4,
						PrevUrl:      getInstanceMetricsUrl(appId, metricType, parameters, 3),
						Resources:    []models.AppInstanceMetric{},
					}
					By("check public api")
					checkAppInstanceMetricResult(pathVariables, parameters, result)

					By("the 5th page should be empty")
					parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "5", "results-per-page": "2"}
					result = AppInstanceMetricResult{
						TotalResults: 5,
						TotalPages:   3,
						Page:         5,
						Resources:    []models.AppInstanceMetric{},
					}
					By("check public api")
					checkAppInstanceMetricResult(pathVariables, parameters, result)
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
							models.AppInstanceMetric{
								AppId:         appId,
								InstanceIndex: 1,
								CollectedAt:   111111,
								Name:          models.MetricNameMemoryUsed,
								Unit:          models.UnitMegaBytes,
								Value:         "123456",
								Timestamp:     444444,
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

					By("check public api")
					checkAppInstanceMetricResult(pathVariables, parameters, result)

					By("get the 2nd page")
					parameters = map[string]string{"instance-index": "1", "start-time": "111111", "end-time": "999999", "order-direction": "asc", "page": "2", "results-per-page": "2"}
					result = AppInstanceMetricResult{
						TotalResults: 2,
						TotalPages:   1,
						Page:         2,
						PrevUrl:      getInstanceMetricsUrlWithInstanceIndex(appId, metricType, parameters, 1),
						Resources:    []models.AppInstanceMetric{},
					}
					By("check public api")
					checkAppInstanceMetricResult(pathVariables, parameters, result)
				})
			})

		})
	})
})

func getInstanceMetricsUrl(appId string, metricType string, parameteters map[string]string, pageNo int) string {
	return fmt.Sprintf("/v1/apps/%s/metric_histories/%s?any=any&start-time=%s&end-time=%s&order-direction=%s&page=%d&results-per-page=%s", appId, metricType, parameteters["start-time"], parameteters["end-time"], parameteters["order-direction"], pageNo, parameteters["results-per-page"])
}

func getInstanceMetricsUrlWithInstanceIndex(appId string, metricType string, parameteters map[string]string, pageNo int) string {
	return fmt.Sprintf("/v1/apps/%s/metric_histories/%s?any=any&instance-index=%s&start-time=%s&end-time=%s&order-direction=%s&page=%d&results-per-page=%s", appId, metricType, parameteters["instance-index"], parameteters["start-time"], parameteters["end-time"], parameteters["order-direction"], pageNo, parameteters["results-per-page"])
}

func compareAppInstanceMetricResult(o1, o2 AppInstanceMetricResult) {
	Expect(o1.Page).To(Equal(o2.Page))
	Expect(o1.TotalPages).To(Equal(o2.TotalPages))
	Expect(o1.TotalResults).To(Equal(o2.TotalResults))
	Expect(o1.Resources).To(Equal(o2.Resources))

	prevUrl1, err1 := url.Parse(o1.PrevUrl)
	Expect(err1).NotTo(HaveOccurred())
	prevUrl2, err2 := url.Parse(o2.PrevUrl)
	Expect(err2).NotTo(HaveOccurred())
	queries1 := prevUrl1.Query()
	queries2 := prevUrl2.Query()
	Expect(queries1).To(Equal(queries2))

	nextUrl1, err1 := url.Parse(o1.NextUrl)
	Expect(err1).NotTo(HaveOccurred())
	nextUrl2, err2 := url.Parse(o2.NextUrl)
	Expect(err2).NotTo(HaveOccurred())
	queries1 = nextUrl1.Query()
	queries2 = nextUrl2.Query()
	Expect(queries1).To(Equal(queries2))

}
func checkAppInstanceMetricResult(pathVariables []string, parameters map[string]string, result AppInstanceMetricResult) {
	var actual AppInstanceMetricResult
	resp, err := getAppInstanceMetrics(pathVariables, parameters)
	defer resp.Body.Close()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred())
	compareAppInstanceMetricResult(actual, result)

}
