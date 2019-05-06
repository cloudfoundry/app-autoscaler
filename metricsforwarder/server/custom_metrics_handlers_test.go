package server_test

import (
	"time"
	"autoscaler/metricsforwarder/fakes"
	. "autoscaler/metricsforwarder/server"
	"autoscaler/models"
	"bytes"
	"encoding/json"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/patrickmn/go-cache"

	"net/http"
	"net/http/httptest"
)

var _ = Describe("MetricHandler", func() {

	var (
		handler *CustomMetricsHandler

		credentialCache cache.Cache

		policyDB         *fakes.FakePolicyDB
		metricsforwarder *fakes.FakeMetricForwarder

		resp *httptest.ResponseRecorder
		req  *http.Request
		err  error
		body []byte

		vars map[string]string

		credentials models.CustomMetricCredentials
		found bool
	)

	BeforeEach(func() {
		logger := lager.NewLogger("metrichandler-test")
		policyDB = &fakes.FakePolicyDB{}
		metricsforwarder = &fakes.FakeMetricForwarder{}
		credentials = models.CustomMetricCredentials{}
		credentialCache = *cache.New(10 * time.Minute, -1)
		vars = make(map[string]string)
		resp = httptest.NewRecorder()
		handler = NewCustomMetricsHandler(logger, metricsforwarder, policyDB, credentialCache, 10 * time.Minute)
		credentialCache.Flush()
	})

	Describe("PublishMetrics", func() {
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodPost, serverUrl+"/v1/an-app-id/metrics", bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
			Expect(err).ToNot(HaveOccurred())
			vars["appid"] = "an-app-id"
			handler.PublishMetrics(resp, req, vars)
			
		})
		Context("when a valid request to publish custom metrics comes",func(){
			Context("when a credentials exists in the cache", func() {
				BeforeEach(func() {
					credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
					credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
					credentialCache.Set("an-app-id", credentials, 10 * time.Minute)
					customMetrics := []*models.CustomMetric{
						&models.CustomMetric{
							Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
						},
					}
					body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
					Expect(err).NotTo(HaveOccurred())
				})
	
				It("should get the credentials from cache without searching from database and returns status code 200", func() {
					Expect(policyDB.GetCustomMetricsCredsCallCount()).To(Equal(0))
					Expect(resp.Code).To(Equal(http.StatusOK))
				})
	
			})

			Context("when a credentials does not exists in the cache but exist in the database", func() {
				BeforeEach(func() {
					policyDB.GetCustomMetricsCredsReturns("$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu","$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G",nil)
					customMetrics := []*models.CustomMetric{
						&models.CustomMetric{
							Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
						},
					}
					body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
					Expect(err).NotTo(HaveOccurred())
				})
	
				It("should get the credentials from database and add it to the cache and returns status code 200", func() {
					Expect(policyDB.GetCustomMetricsCredsCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusOK))
					_, found = credentialCache.Get("an-app-id")
					Expect(found).To(Equal(true))
				})
	
			})

			Context("when a credentials neither exists in the cache nor exist in the database", func() {
				BeforeEach(func() {
					customMetrics := []*models.CustomMetric{
						&models.CustomMetric{
							Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
						},
					}
					body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
					Expect(err).NotTo(HaveOccurred())
				})
	
				It("should search in both cache & database and returns status code 401", func() {
					Expect(policyDB.GetCustomMetricsCredsCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusUnauthorized))
				})
	
			})

			Context("when a stale credentials exists in the cache", func() {
				BeforeEach(func() {
					credentials.Username = "some-stale-hashed-username"
					credentials.Password = "some-stale-hashed-password"
					credentialCache.Set("an-app-id", credentials, 10 * time.Minute)
					policyDB.GetCustomMetricsCredsReturns("$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu","$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G",nil)
					customMetrics := []*models.CustomMetric{
						&models.CustomMetric{
							Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
						},
					}
					body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
					Expect(err).NotTo(HaveOccurred())
				})
	
				It("should search in the database and returns status code 200", func() {
					Expect(policyDB.GetCustomMetricsCredsCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusOK))
				})
			})
		})


		Context("when a request to publish custom metrics comes with malformed request body", func() {

			BeforeEach(func() {
				policyDB.GetCustomMetricsCredsReturns("$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu","$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G",nil)
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
