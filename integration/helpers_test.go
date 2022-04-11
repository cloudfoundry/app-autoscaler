package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type AppInstanceMetricResult struct {
	TotalResults int                        `json:"total_results"`
	TotalPages   int                        `json:"total_pages"`
	Page         int                        `json:"page"`
	PrevUrl      string                     `json:"prev_url"`
	NextUrl      string                     `json:"next_url"`
	Resources    []models.AppInstanceMetric `json:"resources"`
}

type AppAggregatedMetricResult struct {
	TotalResults int                `json:"total_results"`
	TotalPages   int                `json:"total_pages"`
	Page         int                `json:"page"`
	PrevUrl      string             `json:"prev_url"`
	NextUrl      string             `json:"next_url"`
	Resources    []models.AppMetric `json:"resources"`
}

type ScalingHistoryResult struct {
	TotalResults int                        `json:"total_results"`
	TotalPages   int                        `json:"total_pages"`
	Page         int                        `json:"page"`
	PrevUrl      string                     `json:"prev_url"`
	NextUrl      string                     `json:"next_url"`
	Resources    []models.AppScalingHistory `json:"resources"`
}

func getAppAggregatedMetricUrl(appId string, metricType string, parameteters map[string]string, pageNo int) string {
	return fmt.Sprintf("/v1/apps/%s/aggregated_metric_histories/%s?any=any&start-time=%s&end-time=%s&order-direction=%s&page=%d&results-per-page=%s", appId, metricType, parameteters["start-time"], parameteters["end-time"], parameteters["order-direction"], pageNo, parameteters["results-per-page"])
}

func compareAppAggregatedMetricResult(o1, o2 AppAggregatedMetricResult) {
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

func checkAggregatedMetricResult(apiServerPort int, pathVariables []string, parameters map[string]string, result AppAggregatedMetricResult) {
	var actual AppAggregatedMetricResult
	resp, err := getAppAggregatedMetrics(apiServerPort, pathVariables, parameters)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred())
	compareAppAggregatedMetricResult(actual, result)
}

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

func checkAppInstanceMetricResult(apiServerPort int, pathVariables []string, parameters map[string]string, result AppInstanceMetricResult) {
	var actual AppInstanceMetricResult
	resp, err := getAppInstanceMetrics(apiServerPort, pathVariables, parameters)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred())
	compareAppInstanceMetricResult(actual, result)
}

func getScalingHistoriesUrl(appId string, parameteters map[string]string, pageNo int) string {
	return fmt.Sprintf("/v1/apps/%s/scaling_histories?any=any&start-time=%s&end-time=%s&order-direction=%s&page=%d&results-per-page=%s", appId, parameteters["start-time"], parameteters["end-time"], parameteters["order-direction"], pageNo, parameteters["results-per-page"])
}

func compareScalingHistoryResult(o1, o2 ScalingHistoryResult) {
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

func checkScalingHistoryResult(apiServerPort int, pathVariables []string, parameters map[string]string, result ScalingHistoryResult) {
	var actual ScalingHistoryResult
	resp, err := getScalingHistories(apiServerPort, pathVariables, parameters)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred())
	compareScalingHistoryResult(actual, result)
}

func doAttachPolicy(appId string, policyStr []byte, statusCode int, apiServerPort int, httpClient *http.Client) {
	resp, err := attachPolicy(appId, policyStr, apiServerPort, httpClient)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, resp.StatusCode).To(Equal(statusCode))
	resp.Body.Close()
}

func doDetachPolicy(appId string, statusCode int, msg string, apiServerPort int, httpClient *http.Client) {
	resp, err := detachPolicy(appId, apiServerPort, httpClient)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(statusCode))
	if msg != "" {
		respBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(respBody)).To(Equal(msg))
	}
	resp.Body.Close()
}

func checkApiServerStatus(appId string, statusCode int, apiServerPort int, httpClient *http.Client) {
	By("checking the API Server")
	resp, err := getPolicy(appId, apiServerPort, httpClient)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(statusCode))
	resp.Body.Close()
}

func checkApiServerContent(appId string, policyStr []byte, statusCode int, port int, httpClient *http.Client) {
	By("checking the API Server")
	var expected map[string]interface{}
	err := json.Unmarshal(policyStr, &expected)
	Expect(err).NotTo(HaveOccurred())
	checkResponseContent(getPolicy, appId, statusCode, expected, port, httpClient)
}

func checkSchedulerStatus(appId string, statusCode int) {
	By("checking the Scheduler")
	resp, err := getSchedules(appId)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(statusCode))
	resp.Body.Close()
}
