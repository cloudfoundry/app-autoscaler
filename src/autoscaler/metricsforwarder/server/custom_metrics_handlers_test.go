package server_test

import (
	"autoscaler/metricsforwarder/fakes"
	. "autoscaler/metricsforwarder/server"
	"autoscaler/models"
	"bytes"
	"encoding/json"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
	"net/http/httptest"
)

var testCustomMetricsURL = "http://localhost/v1/2b77dd75-a9c2-4743-be97-74bba9cf21b1/metrics"

var _ = Describe("MetricHandler", func() {

	var (
		handler *CustomMetricsHandler

		policyDB         *fakes.FakePolicyDB
		metricsforwarder *fakes.FakeMetricForwarder

		resp *httptest.ResponseRecorder
		req  *http.Request
		err  error
		body []byte
	)

	BeforeEach(func() {
		logger := lager.NewLogger("metrichandler-test")
		policyDB = &fakes.FakePolicyDB{}
		metricsforwarder = &fakes.FakeMetricForwarder{}
		resp = httptest.NewRecorder()
		handler = NewCustomMetricsHandler(logger, metricsforwarder, policyDB)
	})

	Describe("PublishMetrics", func() {
		JustBeforeEach(func() {
			policyDB.ValidateCustomMetricsCredsReturns(true)
			req, err = http.NewRequest(http.MethodPost, testCustomMetricsURL, bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "Basic M2YxZWY2MTJiMThlYTM5YmJlODRjZjUxMzY4MWYwYjc6YWYyNjk1Y2RmZDE0MzA4NThhMWY3MzJhYTI5NTQ2ZTk=")
			Expect(err).ToNot(HaveOccurred())
			handler.PublishMetrics(resp, req, map[string]string{})
		})

		Context("when a request to publish custom metrics comes", func() {

			BeforeEach(func() {
				customMetrics := []*models.CustomMetric{
					&models.CustomMetric{
						Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
					},
				}
				body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns status code 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})

		})

		Context("when a request to publish custom metrics comes with malformed request body", func() {

			BeforeEach(func() {
				body = []byte(`{
					   "instance_index":0,
					   "test" : 
					   "metrics":[
					      {
					         "name":"custom_metric1",
					         "type":"gauge",
					         "value":200,
					         "unit":"unit"
					      }
					   ]
				}`)
			})

			It("returns status code 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})

		})
	})

})
