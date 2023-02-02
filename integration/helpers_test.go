package integration_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

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
	compareUrlValues(o1.NextUrl, o2.NextUrl)
	compareUrlValues(o1.PrevUrl, o2.PrevUrl)
	o1.PrevUrl = ""
	o2.PrevUrl = ""
	o1.NextUrl = ""
	o2.NextUrl = ""
	Expect(o1).To(Equal(o2))
}

func compareUrlValues(o1 string, o2 string) {
	prevUrl1, err1 := url.Parse(o1)
	Expect(err1).NotTo(HaveOccurred())
	prevUrl2, err2 := url.Parse(o2)
	Expect(err2).NotTo(HaveOccurred())
	queries1 := prevUrl1.Query()
	queries2 := prevUrl2.Query()
	Expect(queries1).To(Equal(queries2))
}

func checkAggregatedMetricResult(apiServerPort int, pathVariables []string, parameters map[string]string, result AppAggregatedMetricResult) {
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

func getScalingHistoriesUrl(appId string, parameteters map[string]string, pageNo int) string {
	return fmt.Sprintf("/v1/apps/%s/scaling_histories?any=any&start-time=%s&end-time=%s&order-direction=%s&page=%d&results-per-page=%s", appId, parameteters["start-time"], parameteters["end-time"], parameteters["order-direction"], pageNo, parameteters["results-per-page"])
}

func compareScalingHistoryResult(o1, o2 ScalingHistoryResult) {
	compareUrlValues(o1.NextUrl, o2.NextUrl)
	compareUrlValues(o1.PrevUrl, o2.PrevUrl)
	o1.PrevUrl = ""
	o2.PrevUrl = ""
	o1.NextUrl = ""
	o2.NextUrl = ""
	Expect(o1).To(Equal(o2))
}

func checkScalingHistoryResult(apiServerPort int, pathVariables []string, parameters map[string]string, result ScalingHistoryResult) {
	var actual ScalingHistoryResult
	resp, err := getScalingHistories(apiServerPort, pathVariables, parameters)
	body := MustReadAll(resp.Body)
	FailOnError(fmt.Sprintf("getScalingHistories failed: %d-%s", resp.StatusCode, body), err)
	defer func() { _ = resp.Body.Close() }()
	Expect(resp.StatusCode).WithOffset(1).To(Equal(http.StatusOK), "status code")
	err = json.Unmarshal([]byte(body), &actual)
	Expect(err).WithOffset(1).NotTo(HaveOccurred(), "UnmarshalJson")
	compareScalingHistoryResult(actual, result)
}

func doAttachPolicy(appId string, policyStr []byte, statusCode int, apiServerPort int, httpClient *http.Client) {
	resp, err := attachPolicy(appId, policyStr, apiServerPort, httpClient)
	body := MustReadAll(resp.Body)
	FailOnError(fmt.Sprintf("attachPolicy failed: %d-%s", resp.StatusCode, body), err)
	defer func() { _ = resp.Body.Close() }()
	ExpectWithOffset(1, resp.StatusCode).To(Equal(statusCode), fmt.Sprintf("Got response:%s", body))
}

func MustReadAll(reader io.ReadCloser) string {
	body, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	return string(body)
}

func doDetachPolicy(appId string, statusCode int, msg string, apiServerPort int, httpClient *http.Client) {
	resp, err := detachPolicy(appId, apiServerPort, httpClient)
	FailOnError("detachPolicy failed", err)
	defer func() { _ = resp.Body.Close() }()
	body := MustReadAll(resp.Body)
	Expect(resp.StatusCode).WithOffset(1).To(Equal(statusCode), fmt.Sprintf("response '%s'", body))
	if msg != "" {
		Expect(body).WithOffset(1).To(Equal(msg))
	}
}

func checkApiServerStatus(appId string, statusCode int, apiServerPort int, httpClient *http.Client) {
	By("checking the API Server")
	resp, err := getPolicy(appId, apiServerPort, httpClient)
	FailOnError(fmt.Sprintf("getPolicy failed: %d-%s", resp.StatusCode, MustReadAll(resp.Body)), err)
	defer func() { _ = resp.Body.Close() }()
	Expect(resp.StatusCode).To(Equal(statusCode))
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
