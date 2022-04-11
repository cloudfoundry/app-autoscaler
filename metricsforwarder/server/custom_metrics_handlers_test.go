package server_test

import (
	"bytes"
	"encoding/json"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"net/http"
	"net/http/httptest"

	"github.com/patrickmn/go-cache"
)

var _ = Describe("MetricHandler", func() {

	var (
		handler *CustomMetricsHandler

		allowedMetricCache cache.Cache

		allowedMetricTypeSet map[string]struct{}

		policyDB         *fakes.FakePolicyDB
		metricsforwarder *fakes.FakeMetricForwarder

		resp *httptest.ResponseRecorder
		req  *http.Request
		err  error
		body []byte

		vars map[string]string

		found bool

		scalingPolicy *models.ScalingPolicy
	)

	BeforeEach(func() {
		logger := lager.NewLogger("metrichandler-test")
		policyDB = &fakes.FakePolicyDB{}
		metricsforwarder = &fakes.FakeMetricForwarder{}
		allowedMetricCache = *cache.New(10*time.Minute, -1)
		allowedMetricTypeSet = make(map[string]struct{})
		vars = make(map[string]string)
		resp = httptest.NewRecorder()
		handler = NewCustomMetricsHandler(logger, metricsforwarder, policyDB, allowedMetricCache)
		allowedMetricCache.Flush()
	})

	Describe("PublishMetrics", func() {
		JustBeforeEach(func() {
			req = CreateRequest(body)
			Expect(err).ToNot(HaveOccurred())
			vars["appid"] = "an-app-id"
			handler.VerifyCredentialsAndPublishMetrics(resp, req, vars)
		})

		Context("when a request to publish custom metrics comes with malformed request body", func() {
			BeforeEach(func() {
				policyDB.GetCredentialReturns(&models.Credential{
					Username: "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu",
					Password: "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G",
				}, nil)
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

			It("returns status code 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Bad-Request",
					Message: "Error unmarshaling custom metrics request body",
				}))
			})

		})

		Context("when a valid request to publish custom metrics comes", func() {
			Context("when allowedMetrics exists in the cache", func() {
				BeforeEach(func() {
					scalingPolicy = &models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 6,
						ScalingRules: []*models.ScalingRule{{
							MetricType:            "queuelength",
							BreachDurationSeconds: 60,
							Threshold:             10,
							Operator:              ">",
							CoolDownSeconds:       60,
							Adjustment:            "+1"}}}
					policyDB.GetAppPolicyReturns(scalingPolicy, nil)
					allowedMetricTypeSet["queuelength"] = struct{}{}
					allowedMetricCache.Set("an-app-id", allowedMetricTypeSet, 10*time.Minute)
					customMetrics := []*models.CustomMetric{
						{
							Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
						},
					}
					body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
					Expect(err).NotTo(HaveOccurred())
				})

				It("should get the allowedMetrics from cache without searching from database and returns status code 200", func() {
					Expect(policyDB.GetAppPolicyCallCount()).To(Equal(0))
					Expect(resp.Code).To(Equal(http.StatusOK))
				})

			})

			Context("when allowedMetrics does not exists in the cache but exist in the database", func() {
				BeforeEach(func() {
					scalingPolicy = &models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 6,
						ScalingRules: []*models.ScalingRule{{
							MetricType:            "queuelength",
							BreachDurationSeconds: 60,
							Threshold:             10,
							Operator:              ">",
							CoolDownSeconds:       60,
							Adjustment:            "+1"}}}
					policyDB.GetAppPolicyReturns(scalingPolicy, nil)
					customMetrics := []*models.CustomMetric{
						{
							Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
						},
					}
					body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
					Expect(err).NotTo(HaveOccurred())
				})

				It("should get the allowedMetrics from database and add it to the cache and returns status code 200", func() {
					Expect(policyDB.GetAppPolicyCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusOK))
					_, found = allowedMetricCache.Get("an-app-id")
					Expect(found).To(Equal(true))
				})

			})

			Context("when allowedMetrics neither exists in the cache nor exist in the database", func() {
				BeforeEach(func() {
					customMetrics := []*models.CustomMetric{
						{
							Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
						},
					}
					body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
					Expect(err).NotTo(HaveOccurred())
				})

				It("should search in both cache & database and returns status code 400", func() {
					Expect(policyDB.GetAppPolicyCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "no policy defined",
					}))
				})
			})
		})

		Context("when a request to publish custom metrics comes with standard metric type", func() {
			BeforeEach(func() {
				policyDB.GetCredentialReturns(&models.Credential{
					Username: "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu",
					Password: "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G",
				}, nil)
				body = []byte(`{
					   "instance_index":0,
					   "metrics":[
					      {
					         "name":"memoryused",
					         "type":"gauge",
					         "value":200,
					         "unit":"unit"
					      }
					   ]
				}`)
				scalingPolicy = &models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 6,
					ScalingRules: []*models.ScalingRule{{
						MetricType:            "queuelength",
						BreachDurationSeconds: 60,
						Threshold:             10,
						Operator:              ">",
						CoolDownSeconds:       60,
						Adjustment:            "+1"}}}
				policyDB.GetAppPolicyReturns(scalingPolicy, nil)
			})

			It("returns status code 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Bad-Request",
					Message: "Custom Metric: memoryused matches with standard metrics name",
				}))
			})

		})

		Context("when a request to publish custom metrics comes with non allowed metric types", func() {
			BeforeEach(func() {
				policyDB.GetCredentialReturns(&models.Credential{
					Username: "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu",
					Password: "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G",
				}, nil)
				body = []byte(`{
					   "instance_index":0,
					   "metrics":[
					      {
					         "name":"wrong_metric_type",
					         "type":"gauge",
					         "value":200,
					         "unit":"unit"
					      }
					   ]
				}`)
				scalingPolicy = &models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 6,
					ScalingRules: []*models.ScalingRule{{
						MetricType:            "queuelength",
						BreachDurationSeconds: 60,
						Threshold:             10,
						Operator:              ">",
						CoolDownSeconds:       60,
						Adjustment:            "+1"}}}
				policyDB.GetAppPolicyReturns(scalingPolicy, nil)
			})

			It("returns status code 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Bad-Request",
					Message: "Custom Metric: wrong_metric_type does not match with metrics defined in policy",
				}))
			})

		})
	})

})

func CreateRequest(body []byte) *http.Request {
	req, err := http.NewRequest(http.MethodPost, serverUrl+"/v1/apps/an-app-id/metrics", bytes.NewReader(body))
	Expect(err).ToNot(HaveOccurred())
	req.Header.Add("Content-Type", "application/json")
	return req
}
