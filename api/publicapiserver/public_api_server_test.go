package publicapiserver_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"autoscaler/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PublicApiServer", func() {
	var (
		policy = `{
			"instance_min_count": 1,
			"instance_max_count": 5,
			"scaling_rules": [{
				"metric_type": "memoryused",
				"breach_duration_secs": 300,
				"threshold": 30,
				"operator": ">",
				"cool_down_secs": 300,
				"adjustment": "-1"
			}],
			"schedules": {
				"timezone": "Asia/Kolkata",
				"recurring_schedule": [{
					"start_time": "10:00",
					"end_time": "18:00",
					"days_of_week": [1, 2, 3],
					"instance_min_count": 1,
					"instance_max_count": 10,
					"initial_min_instance_count": 5
				}]
			}
		}`
	)

	BeforeEach(func() {

		scalingEngineResponse = []models.AppScalingHistory{
			{
				AppId:        TEST_APP_ID,
				Timestamp:    300,
				ScalingType:  0,
				Status:       0,
				OldInstances: 2,
				NewInstances: 4,
				Reason:       "a reason",
				Message:      "",
				Error:        "",
			},
		}

		metricsCollectorResponse = []models.AppInstanceMetric{
			{
				AppId:         TEST_APP_ID,
				Timestamp:     100,
				InstanceIndex: 0,
				CollectedAt:   0,
				Name:          TEST_METRIC_TYPE,
				Unit:          TEST_METRIC_UNIT,
				Value:         "200",
			},
		}

		eventGeneratorResponse = []models.AppMetric{
			{
				AppId:      TEST_APP_ID,
				Timestamp:  100,
				MetricType: TEST_METRIC_TYPE,
				Unit:       TEST_METRIC_UNIT,
				Value:      "200",
			},
		}
	})

	Describe("Protected Routes", func() {

		Describe("Exceed rate limit", func() {
			BeforeEach(func() {
				fakeRateLimiter.ExceedsLimitReturns(true)
			})
			AfterEach(func() {
				fakeRateLimiter.ExceedsLimitReturns(false)
			})

			Context("when calling scaling_histories endpoint", func() {
				It("should fail with 429", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/scaling_histories",
						nil, http.MethodGet, "", http.StatusTooManyRequests)
				})
			})

			Context("when calling instance metrics endpoint", func() {
				It("should fail with 429", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE,
						nil, http.MethodGet, "", http.StatusTooManyRequests)
				})
			})

			Context("when calling aggregated metrics endpoint", func() {
				It("should fail with 429", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE,
						nil, http.MethodGet, "", http.StatusTooManyRequests)
				})
			})

			Context("when calling get policy endpoint", func() {
				It("should fail with 429", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						nil, http.MethodGet, "", http.StatusTooManyRequests)
				})
			})

			Context("when calling attach policy endpoint", func() {
				It("should fail with 429", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						nil, http.MethodPut, "", http.StatusTooManyRequests)
				})
			})

			Context("when calling detach policy endpoint", func() {
				It("should fail with 429", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						nil, http.MethodDelete, "", http.StatusTooManyRequests)
				})

			})

			Context("when calling create credential endpoint", func() {
				It("should fail with 429", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/credential",
						nil, http.MethodPut, "", http.StatusTooManyRequests)
				})

			})

			Context("when calling delete credential endpoint", func() {
				It("should fail with 429", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/credential",
						nil, http.MethodDelete, "", http.StatusTooManyRequests)
				})

			})

		})

		Describe("Without AuthorizatioToken", func() {
			Context("when calling scaling_histories endpoint", func() {
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/scaling_histories",
						nil, http.MethodGet, "", http.StatusUnauthorized)
				})
			})

			Context("when calling instance metrics endpoint", func() {
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE,
						nil, http.MethodGet, "", http.StatusUnauthorized)
				})
			})

			Context("when calling aggregated metrics endpoint", func() {
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE,
						nil, http.MethodGet, "", http.StatusUnauthorized)
				})
			})

			Context("when calling get policy endpoint", func() {
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						nil, http.MethodGet, "", http.StatusUnauthorized)
				})
			})

			Context("when calling attach policy endpoint", func() {
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						nil, http.MethodPut, "", http.StatusUnauthorized)
				})
			})

			Context("when calling detach policy endpoint", func() {
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						nil, http.MethodDelete, "", http.StatusUnauthorized)
				})

			})

			Context("when calling create credential endpoint", func() {
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/credential",
						nil, http.MethodPut, "", http.StatusUnauthorized)
				})

			})

			Context("when calling delete credential endpoint", func() {
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/credential",
						nil, http.MethodDelete, "", http.StatusUnauthorized)
				})

			})

		})

		Describe("With Invalid Authorization Token", func() {
			BeforeEach(func() {
				fakeCFClient.IsUserSpaceDeveloperReturns(false, nil)
			})

			Context("when calling scaling_histories endpoint", func() {
				BeforeEach(func() {
					scalingEngineStatus = http.StatusOK
				})
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/scaling_histories",
						map[string]string{"Authorization": TEST_INVALID_USER_TOKEN}, http.MethodGet, "", http.StatusUnauthorized)
				})
			})

			Context("when calling instance metric endpoint", func() {
				BeforeEach(func() {
					metricsCollectorStatus = http.StatusOK
				})
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE,
						map[string]string{"Authorization": TEST_INVALID_USER_TOKEN}, http.MethodGet, "", http.StatusUnauthorized)
				})
			})

			Context("when calling aggregated metric endpoint", func() {
				BeforeEach(func() {
					eventGeneratorStatus = http.StatusOK
				})
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE,
						map[string]string{"Authorization": TEST_INVALID_USER_TOKEN}, http.MethodGet, "", http.StatusUnauthorized)
				})
			})

			Context("when calling get policy endpoint", func() {
				BeforeEach(func() {
					schedulerStatus = http.StatusOK
				})
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						map[string]string{"Authorization": TEST_INVALID_USER_TOKEN}, http.MethodGet, "", http.StatusUnauthorized)
				})

			})

			Context("when calling attach policy endpoint", func() {
				BeforeEach(func() {
					schedulerStatus = http.StatusOK
				})
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						map[string]string{"Authorization": TEST_INVALID_USER_TOKEN}, http.MethodPut, "", http.StatusUnauthorized)
				})

			})

			Context("when calling detach policy endpoint", func() {
				BeforeEach(func() {
					schedulerStatus = http.StatusOK
				})
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						map[string]string{"Authorization": TEST_INVALID_USER_TOKEN}, http.MethodDelete, "", http.StatusUnauthorized)
				})

			})
			Context("when calling create credential endpoint", func() {
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/credential",
						map[string]string{"Authorization": TEST_INVALID_USER_TOKEN}, http.MethodPut, "", http.StatusUnauthorized)
				})

			})
			Context("when calling delete credential endpoint", func() {
				It("should fail with 401", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/credential",
						map[string]string{"Authorization": TEST_INVALID_USER_TOKEN}, http.MethodDelete, "", http.StatusUnauthorized)
				})

			})
		})

		Describe("With valid authorization token", func() {
			BeforeEach(func() {
				fakeCFClient.IsUserSpaceDeveloperReturns(true, nil)
			})

			Context("when calling scaling_histories endpoint", func() {
				BeforeEach(func() {
					scalingEngineStatus = http.StatusOK
				})
				It("should succeed", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/scaling_histories",
						map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodGet, "", http.StatusOK)
				})
			})

			Context("when calling instance metric endpoint", func() {
				BeforeEach(func() {
					metricsCollectorStatus = http.StatusOK
				})
				It("should succeed", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE,
						map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodGet, "", http.StatusOK)
				})
			})

			Context("when calling aggregated metric endpoint", func() {
				BeforeEach(func() {
					eventGeneratorStatus = http.StatusOK
				})
				It("should succeed", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE,
						map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodGet, "", http.StatusOK)
				})
			})

			Context("when calling get policy endpoint", func() {
				JustBeforeEach(func() {
					schedulerStatus = http.StatusOK
					fakePolicyDB.GetAppPolicyReturns(&models.ScalingPolicy{
						InstanceMax: 5,
						InstanceMin: 1,
						ScalingRules: []*models.ScalingRule{
							&models.ScalingRule{
								MetricType:            "memoryused",
								BreachDurationSeconds: 300,
								CoolDownSeconds:       300,
								Threshold:             30,
								Operator:              "<",
								Adjustment:            "-1",
							}},
					}, nil)

				})
				It("should succeed", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodGet, "", http.StatusOK)
				})

			})

			Context("when calling attach policy endpoint", func() {
				BeforeEach(func() {
					schedulerStatus = http.StatusOK

				})
				It("should succeed", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodPut, policy, http.StatusOK)
				})

			})

			Context("when calling detach policy endpoint", func() {
				BeforeEach(func() {
					schedulerStatus = http.StatusOK
				})
				It("should succeed", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
						map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodPut, policy, http.StatusOK)
				})
			})

			Context("when calling create credential endpoint", func() {
				It("should succeed", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/credential",
						map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodPut, "", http.StatusOK)
				})

			})
			Context("when calling delete credential endpoint", func() {
				It("should succeed", func() {
					verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/credential",
						map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodDelete, "", http.StatusOK)
				})

			})
		})
	})
	Describe("UnProtected Routes", func() {
		Context("when calling info endpoint", func() {
			It("should succeed", func() {
				verifyResponse(httpClient, serverUrl, "/v1/info", nil, http.MethodGet, "", http.StatusOK)
			})
		})
		Context("when calling health endpoint", func() {
			It("should succeed", func() {
				verifyResponse(httpClient, serverUrl, "/health", nil, http.MethodGet, "", http.StatusOK)
			})
		})
	})

	Context("when requesting non existing path", func() {
		It("should get 404", func() {
			verifyResponse(httpClient, serverUrl, "/non-existing-path", nil, http.MethodGet, "", http.StatusNotFound)
		})
	})
})

func verifyResponse(httpClient *http.Client, serverUrl *url.URL, path string, headers map[string]string, httpRequestMethod string, httpRequestBody string, expectResponseStatusCode int) {
	serverUrl.Path = path
	var body io.Reader = nil
	if httpRequestBody != "" {
		body = strings.NewReader(httpRequestBody)
	}
	req, err := http.NewRequest(httpRequestMethod, serverUrl.String(), body)
	if headers != nil && len(headers) > 0 {
		for headerName, headerValue := range headers {
			req.Header.Set(headerName, headerValue)
		}
	}
	Expect(err).NotTo(HaveOccurred())
	rsp, err := httpClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	Expect(rsp.StatusCode).To(Equal(expectResponseStatusCode))

}
