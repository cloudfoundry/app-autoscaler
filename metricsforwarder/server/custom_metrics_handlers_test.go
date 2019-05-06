package server_test

import (
	"autoscaler/metricsforwarder/fakes"
	. "autoscaler/metricsforwarder/server"
	"autoscaler/models"
	"bytes"
	"encoding/json"
	"time"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
	"net/http/httptest"

	cache "github.com/patrickmn/go-cache"
)

var _ = Describe("MetricHandler", func() {

	var (
		handler *CustomMetricsHandler

		credentialCache    cache.Cache
		allowedMetricCache cache.Cache

		allowedMetricTypeSet map[string]struct{}

		policyDB         *fakes.FakePolicyDB
		metricsforwarder *fakes.FakeMetricForwarder

		resp *httptest.ResponseRecorder
		req  *http.Request
		err  error
		body []byte

		vars map[string]string

		credentials models.CustomMetricCredentials
		found       bool

		scalingPolicy *models.ScalingPolicy
	)

	BeforeEach(func() {
		logger := lager.NewLogger("metrichandler-test")
		policyDB = &fakes.FakePolicyDB{}
		metricsforwarder = &fakes.FakeMetricForwarder{}
		credentials = models.CustomMetricCredentials{}
		credentialCache = *cache.New(10*time.Minute, -1)
		allowedMetricCache = *cache.New(10*time.Minute, -1)
		allowedMetricTypeSet = make(map[string]struct{})
		vars = make(map[string]string)
		resp = httptest.NewRecorder()
		handler = NewCustomMetricsHandler(logger, metricsforwarder, policyDB, credentialCache, allowedMetricCache, 10*time.Minute)
		credentialCache.Flush()
		allowedMetricCache.Flush()
	})

	Describe("PublishMetrics", func() {
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodPost, serverUrl+"/v1/apps/an-app-id/metrics", bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
			Expect(err).ToNot(HaveOccurred())
			vars["appid"] = "an-app-id"
			handler.PublishMetrics(resp, req, vars)

		})
		Context("when a valid request to publish custom metrics comes", func() {
			Context("when credentials exists in the cache", func() {
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
					credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
					credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
					credentialCache.Set("an-app-id", credentials, 10*time.Minute)
					allowedMetricTypeSet["queuelength"] = struct{}{}
					allowedMetricCache.Set("allowed_metric_types", allowedMetricTypeSet, 10*time.Minute)
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

			Context("when credentials does not exists in the cache but exist in the database", func() {
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
					policyDB.GetCustomMetricsCredsReturns("$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu", "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G", nil)
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

			Context("when credentials neither exists in the cache nor exist in the database", func() {
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
					credentialCache.Set("an-app-id", credentials, 10*time.Minute)
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
					policyDB.GetCustomMetricsCredsReturns("$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu", "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G", nil)
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
				policyDB.GetCustomMetricsCredsReturns("$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu", "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G", nil)
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
					credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
					credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
					credentialCache.Set("an-app-id", credentials, 10*time.Minute)
					allowedMetricTypeSet["queuelength"] = struct{}{}
					allowedMetricCache.Set("allowed_metric_types", allowedMetricTypeSet, 10*time.Minute)
					customMetrics := []*models.CustomMetric{
						&models.CustomMetric{
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
					credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
					credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
					credentialCache.Set("an-app-id", credentials, 10*time.Minute)
					customMetrics := []*models.CustomMetric{
						&models.CustomMetric{
							Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
						},
					}
					body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
					Expect(err).NotTo(HaveOccurred())
				})

				It("should get the allowedMetrics from database and add it to the cache and returns status code 200", func() {
					Expect(policyDB.GetAppPolicyCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusOK))
					_, found = allowedMetricCache.Get("allowed_metric_types")
					Expect(found).To(Equal(true))
				})

			})

			Context("when allowedMetrics neither exists in the cache nor exist in the database", func() {
				BeforeEach(func() {
					customMetrics := []*models.CustomMetric{
						&models.CustomMetric{
							Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
						},
					}
					credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
					credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
					credentialCache.Set("an-app-id", credentials, 10*time.Minute)
					body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
					Expect(err).NotTo(HaveOccurred())
				})

				It("should search in both cache & database and returns status code 401", func() {
					Expect(policyDB.GetAppPolicyCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})

			})
		})
	})

})
