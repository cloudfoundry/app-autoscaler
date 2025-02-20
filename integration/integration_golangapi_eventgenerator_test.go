package integration_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const metricType = "memoryused"
const initInstanceCount = 2

type testMetrics struct {
	AppId             string
	BindingId         string
	ServiceInstanceId string
	OrgId             string
	SpaceId           string
	PathVariables     []string
}

func (t *testMetrics) InitializeIdentifiers() {
	t.ServiceInstanceId = getRandomIdRef("serviceInstId")
	t.OrgId = getRandomIdRef("orgId")
	t.SpaceId = getRandomIdRef("spaceId")
	t.BindingId = getRandomIdRef("bindingId")
	t.AppId = getRandomIdRef("appId")
	t.PathVariables = []string{t.AppId, metricType}
}

var _ = Describe("Integration_GolangApi_EventGenerator", func() {
	var (
		t                       *testMetrics
		eventGeneratorConfPath  string
		golangApiServerConfPath string
		brokerUrl               *url.URL
		tmpDir                  string
		err                     error
	)

	BeforeEach(func() {
		tmpDir, err = os.MkdirTemp("", "autoscaler")
		Expect(err).NotTo(HaveOccurred())

		brokerUrl, err = url.Parse(fmt.Sprintf("https://127.0.0.1:%d", components.Ports[GolangServiceBroker]))
		Expect(err).NotTo(HaveOccurred())

		t = &testMetrics{}
		setupTestEnvironment(t)
	})

	JustBeforeEach(func() {
		startEventGenerator(eventGeneratorConfPath)
		startGolangApiServer(golangApiServerConfPath)
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
		tearDownTestEnvironment()
	})

	When("using eventgenerator unified CF Server", func() {
		JustBeforeEach(func() {
			bindServiceInstance(brokerUrl, t)
		})

		BeforeEach(func() {
			eventGeneratorConfPath = prepareEventGeneratorConfig(tmpDir)
			golangApiServerConfPath = prepareGolangApiServerConfig(tmpDir)
		})

		Context("Get aggregated metrics", func() {
			var timestamps []int64

			BeforeEach(func() {
				timestamps = []int64{333333, 444444, 555555, 555556, 666666}
				insertTestMetrics(t, timestamps...)
			})

			It("should get the metrics", func() {
				expectedResources := generateResources(t, timestamps...)

				verifyAggregatedMetrics(t, "111111", "999999", "asc", "1", "2", 5, 3, 1, 2, expectedResources[0:2])
				verifyAggregatedMetrics(t, "111111", "999999", "asc", "2", "2", 5, 3, 2, 2, expectedResources[2:4])
				verifyAggregatedMetrics(t, "111111", "999999", "asc", "3", "2", 5, 3, 3, 1, expectedResources[4:5])

				verifyEmptyAggregatedMetrics(t, "111111", "999999", "asc", "4", "2", 5, 3, 4)
			})

			It("should get the metrics in specified time scope", func() {
				expectedResources := generateResources(t, timestamps...)

				verifyMetricsInTimeScope(t, "555555", "10", 3, 1, expectedResources[2:5])
				verifyMetricsInTimeScope(t, "444444", "10", 4, 1, expectedResources[1:5])
				verifyMetricsInTimeScopeWithRange(t, "444444", "555556", "10", 3, 1, expectedResources[1:4])
			})
		})
	})

	When("the using eventgenerator legacy Server", func() {
		BeforeEach(func() {
			eventGeneratorConfPath = prepareEventGeneratorConfig(tmpDir)
			golangApiServerConfPath = prepareGolangApiServerConfig(tmpDir)
		})

		Describe("Get App Metrics", func() {
			Context("Cloud Controller API is not available", func() {
				JustBeforeEach(func() {
					prepareFakeCCNOAAUAA()
				})
				It("should return status code 500", func() {
					verifyErrorResponse(t, http.StatusInternalServerError, "Failed to check if user is admin")
				})
			})

			Context("UAA API is not available", func() {
				JustBeforeEach(func() {
					prepareFakeCCNOAAUAA()
				})

				It("should return status code 500", func() {
					verifyErrorResponse(t, http.StatusInternalServerError, "Failed to check if user is admin")
				})
			})

			Context("UAA API returns 401", func() {
				JustBeforeEach(func() {
					prepareFakeCCNOAAUAAWithUnauthorized()
				})

				XIt("should return status code 401", func() {
					verifyErrorResponse(t, http.StatusUnauthorized, "You are not authorized to perform the requested action")
				})
			})

			Context("Check permission not passed", func() {
				BeforeEach(func() {
					fakeCCNOAAUAA.Add().Roles(http.StatusOK)
				})
				It("should return status code 401", func() {
					verifyErrorResponse(t, http.StatusUnauthorized, "You are not authorized to perform the requested action")
				})
			})

			When("the app is bound to the service instance", func() {
				JustBeforeEach(func() {
					bindServiceInstance(brokerUrl, t)
				})

				Context("EventGenerator is down", func() {
					JustBeforeEach(func() {
						stopEventGenerator()
					})

					It("should return status code 500", func() {
						verifyErrorResponse(t, http.StatusInternalServerError, "Error retrieving metrics history from eventgenerator")
					})
				})

				Context("Get aggregated metrics", func() {
					var timestamps []int64

					BeforeEach(func() {
						timestamps = []int64{333333, 444444, 555555, 555556, 666666}
						insertTestMetrics(t, timestamps...)
					})

					It("should get the metrics", func() {
						expectedResources := generateResources(t, timestamps...)

						verifyAggregatedMetrics(t, "111111", "999999", "asc", "1", "2", 5, 3, 1, 2, expectedResources[0:2])
						verifyAggregatedMetrics(t, "111111", "999999", "asc", "2", "2", 5, 3, 2, 2, expectedResources[2:4])
						verifyAggregatedMetrics(t, "111111", "999999", "asc", "3", "2", 5, 3, 3, 1, expectedResources[4:5])

						verifyEmptyAggregatedMetrics(t, "111111", "999999", "asc", "4", "2", 5, 3, 4)
					})

					It("should get the metrics in specified time scope", func() {
						expectedResources := generateResources(t, timestamps...)

						verifyMetricsInTimeScope(t, "555555", "10", 3, 1, expectedResources[2:5])
						verifyMetricsInTimeScope(t, "444444", "10", 4, 1, expectedResources[1:5])
						verifyMetricsInTimeScopeWithRange(t, "444444", "555556", "10", 3, 1, expectedResources[1:4])
					})
				})
			})
		})

	})
})

func checkAggregatedMetricResult(apiServerPort int, pathVariables []string, parameters map[string]string, result AppAggregatedMetricResult) {
	//GinkgoHelper()
	var actual AppAggregatedMetricResult
	resp, err := getAppAggregatedMetrics(apiServerPort, pathVariables, parameters)
	body := MustReadAll(resp.Body)

	FailOnError(fmt.Sprintf("getAppAggregatedMetrics failed: %d-%s", resp.StatusCode, body), err)
	defer func() { _ = resp.Body.Close() }()
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = json.Unmarshal([]byte(body), &actual)
	Expect(err).NotTo(HaveOccurred())
	compareAppAggregatedMetricResult(actual, result)
}

func setupTestEnvironment(t *testMetrics) {
	GinkgoHelper()
	startFakeCCNOAAUAA(initInstanceCount)
	httpClient = NewApiClient()
	httpClientForPublicApi = NewPublicApiClient()
	t.InitializeIdentifiers()
}

func tearDownTestEnvironment() {
	stopGolangApiServer()
	stopEventGenerator()
}

func prepareFakeCCNOAAUAA() {
	fakeCCNOAAUAA.Reset()
	fakeCCNOAAUAA.AllowUnhandledRequests = true
}

func prepareFakeCCNOAAUAAWithUnauthorized() {
	fakeCCNOAAUAA.Reset()
	fakeCCNOAAUAA.AllowUnhandledRequests = true
}

func bindServiceInstance(brokerUrl *url.URL, t *testMetrics) {
	GinkgoHelper()
	provisionAndBind(brokerUrl, t.ServiceInstanceId, t.OrgId, t.SpaceId, t.BindingId, t.AppId, httpClientForPublicApi)
}

func insertTestMetrics(t *testMetrics, timestamps ...int64) {
	metric := &models.AppMetric{
		AppId:      t.AppId,
		MetricType: models.MetricNameMemoryUsed,
		Unit:       models.UnitMegaBytes,
		Value:      "123456",
	}
	for _, timestamp := range timestamps {
		metric.Timestamp = timestamp
		insertAppMetric(metric)
	}
	metric.MetricType = models.MetricNameThroughput
	metric.Unit = models.UnitNum
	metric.Timestamp = 444444
	insertAppMetric(metric)
	metric.AppId = getRandomIdRef("metric.appId")
	metric.MetricType = models.MetricNameMemoryUsed
	metric.Unit = models.UnitMegaBytes
	metric.Timestamp = 444444
	insertAppMetric(metric)
}

func verifyErrorResponse(t *testMetrics, expectedStatus int, expectedMessage string) {
	GinkgoHelper()
	var expectedCodeMessage string

	switch expectedStatus {
	case http.StatusUnauthorized:
		expectedCodeMessage = "Unauthorized"

	case http.StatusInternalServerError:
		expectedCodeMessage = http.StatusText(expectedStatus)
	}

	parameters := map[string]string{}
	checkPublicAPIResponseContentWithParameters(getAppAggregatedMetrics, components.Ports[GolangAPIServer], t.PathVariables, parameters, expectedStatus, map[string]interface{}{
		"code":    expectedCodeMessage,
		"message": expectedMessage,
	})
}

func verifyAggregatedMetrics(t *testMetrics, startTime, endTime, orderDirection, page, resultsPerPage string, totalResults, totalPages, pageNum, resourcesCount int, expectedResources []models.AppMetric) {
	//GinkgoHelper()

	parameters := map[string]string{"start-time": startTime, "end-time": endTime, "order-direction": orderDirection, "page": page, "results-per-page": resultsPerPage}
	result := AppAggregatedMetricResult{
		TotalResults: totalResults,
		TotalPages:   totalPages,
		Page:         pageNum,
		Resources:    expectedResources,
	}
	if pageNum > 1 {
		result.PrevUrl = getAppAggregatedMetricPrevUrl(t.AppId, metricType, parameters)
	}

	if pageNum != totalPages {
		result.NextUrl = getAppAggregatedMetricNextUrl(t.AppId, metricType, parameters)
	}

	checkAggregatedMetricResult(components.Ports[GolangAPIServer], t.PathVariables, parameters, result)
}

func verifyEmptyAggregatedMetrics(t *testMetrics, startTime, endTime, orderDirection, page, resultsPerPage string, totalResults, totalPages, pageNum int) {
	GinkgoHelper()

	parameters := map[string]string{"start-time": startTime, "end-time": endTime, "order-direction": orderDirection, "page": page, "results-per-page": resultsPerPage}
	result := AppAggregatedMetricResult{
		TotalResults: totalResults,
		TotalPages:   totalPages,
		Page:         pageNum,
		PrevUrl:      getAppAggregatedMetricPrevUrl(t.AppId, metricType, parameters),
		Resources:    []models.AppMetric{},
	}

	checkAggregatedMetricResult(components.Ports[GolangAPIServer], t.PathVariables, parameters, result)
}

func verifyMetricsInTimeScope(t *testMetrics, startTime, resultsPerPage string, totalResults, totalPages int, expectedResources []models.AppMetric) {
	GinkgoHelper()

	parameters := map[string]string{"start-time": startTime, "order-direction": "asc", "page": "1", "results-per-page": resultsPerPage}
	result := AppAggregatedMetricResult{

		TotalResults: totalResults,
		TotalPages:   totalPages,
		Page:         1,
		Resources:    expectedResources,
	}
	checkAggregatedMetricResult(components.Ports[GolangAPIServer], t.PathVariables, parameters, result)
}

func verifyMetricsInTimeScopeWithRange(t *testMetrics, startTime, endTime, resultsPerPage string, totalResults, totalPages int, expectedResources []models.AppMetric) {
	GinkgoHelper()
	parameters := map[string]string{"start-time": startTime, "end-time": endTime, "order-direction": "asc", "page": "1", "results-per-page": resultsPerPage}
	result := AppAggregatedMetricResult{
		TotalResults: totalResults,
		TotalPages:   totalPages,
		Page:         1,
		Resources:    expectedResources,
	}
	checkAggregatedMetricResult(components.Ports[GolangAPIServer], t.PathVariables, parameters, result)
}

func generateResources(t *testMetrics, timestamps ...int64) []models.AppMetric {
	count := len(timestamps)
	resources := make([]models.AppMetric, count)
	for i, timestamp := range timestamps {
		resources[i] = models.AppMetric{
			AppId:      t.AppId,
			MetricType: models.MetricNameMemoryUsed,
			Unit:       models.UnitMegaBytes,
			Value:      "123456",
			Timestamp:  timestamp,
		}
	}

	return resources
}

func prepareEventGeneratorConfig(tmpDir string) string {
	return components.PrepareEventGeneratorConfig(dbUrl,
		components.Ports[EventGenerator],
		fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]),
		fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
		aggregatorExecuteInterval, policyPollerInterval,
		saveInterval, evaluationManagerInterval, defaultHttpClientTimeout,
		tmpDir)
}

func prepareGolangApiServerConfig(tmpDir string) string {
	golangApiServerConfPath := components.PrepareGolangApiServerConfig(
		dbUrl,
		fakeCCNOAAUAA.URL(),
		fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]),
		fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
		fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]),
		tmpDir)

	brokerAuth = base64.StdEncoding.EncodeToString([]byte("broker_username:broker_password"))

	return golangApiServerConfPath
}

func getAppAggregatedMetricNextUrl(appId string, metricType string, params map[string]string) string {
	currentPage, err := strconv.Atoi(params["page"])
	Expect(err).NotTo(HaveOccurred())
	page := strconv.Itoa(currentPage + 1)

	return getAppAggregatedMetricUrl(appId, metricType, params, page)
}

func getAppAggregatedMetricPrevUrl(appId string, metricType string, params map[string]string) string {
	currentPage, err := strconv.Atoi(params["page"])
	Expect(err).NotTo(HaveOccurred())
	page := strconv.Itoa(currentPage - 1)

	return getAppAggregatedMetricUrl(appId, metricType, params, page)
}

func getAppAggregatedMetricUrl(appId string, metricType string, params map[string]string, page string) string {
	return fmt.Sprintf("/v1/apps/%s/aggregated_metric_histories/%s?any=any&end-time=%s&order-direction=%s&page=%s&results-per-page=%s&start-time=%s", appId, metricType, params["end-time"], params["order-direction"], page, params["results-per-page"], params["start-time"])
}
