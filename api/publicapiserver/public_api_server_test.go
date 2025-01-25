package publicapiserver_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/publicapiserver"
	internalscalinghistory "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/apis/scalinghistory"
	"code.cloudfoundry.org/lager/v3/lagertest"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit/ginkgomon_v2"
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
		scalingHistoryEntry := []internalscalinghistory.HistoryEntry{
			{
				Status:       internalscalinghistory.NewOptHistoryEntryStatus(internalscalinghistory.HistoryEntryStatus0),
				AppID:        internalscalinghistory.NewOptGUID(TEST_APP_ID),
				Timestamp:    internalscalinghistory.NewOptInt(300),
				ScalingType:  internalscalinghistory.NewOptHistoryEntryScalingType(internalscalinghistory.HistoryEntryScalingType0),
				OldInstances: internalscalinghistory.NewOptInt64(2),
				NewInstances: internalscalinghistory.NewOptInt64(4),
				Reason:       internalscalinghistory.NewOptString("a reason"),
			},
		}

		scalingEngineResponse = internalscalinghistory.History{
			TotalResults: internalscalinghistory.NewOptInt64(1),
			TotalPages:   internalscalinghistory.NewOptInt64(1),
			Page:         internalscalinghistory.NewOptInt64(1),
			PrevURL:      internalscalinghistory.OptURI{},
			NextURL:      internalscalinghistory.OptURI{},
			Resources:    scalingHistoryEntry,
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

	})

	AfterEach(func() {
		ginkgomon_v2.Interrupt(serverProcess)
	})

	Describe("CreateMtlsServer", func() {
		JustBeforeEach(func() {
			eventGeneratorResponse = []models.AppMetric{
				{
					AppId:      TEST_APP_ID,
					Timestamp:  100,
					MetricType: TEST_METRIC_TYPE,
					Unit:       TEST_METRIC_UNIT,
					Value:      "200",
				},
			}
			publicApiServer := publicapiserver.NewPublicApiServer(
				lagertest.NewTestLogger("public_apiserver"), conf, fakePolicyDB,
				fakeBindingDB, fakeCredentials, checkBindingFunc, fakeCFClient,
				httpStatusCollector, fakeRateLimiter, fakeBrokerServer)

			httpServer, err := publicApiServer.CreateMtlsServer()
			Expect(err).NotTo(HaveOccurred())
			serverProcess = ginkgomon_v2.Invoke(httpServer)
		})

		Context("when calling health endpoint", func() {
			It("should succeed", func() {
				res := verifyResponse(httpClient, serverUrl, "/health", nil, http.MethodGet, "", http.StatusOK)
				Expect(res).To(ContainSubstring("alive"))
			})
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

			})

			Describe("Without AuthorizatioToken", func() {
				Context("when calling scaling_histories endpoint", func() {
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/scaling_histories",
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

			})

			Describe("Without Client Token", func() {
				BeforeEach(func() {
					fakeCFClient.IsUserSpaceDeveloperReturns(true, nil)
				})

				Context("when calling scaling_histories endpoint", func() {
					BeforeEach(func() {
						scalingEngineStatus = http.StatusOK
					})
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/scaling_histories",
							map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodGet, "", http.StatusUnauthorized)
					})
				})

				Context("when calling aggregated metric endpoint", func() {
					BeforeEach(func() {
						eventGeneratorStatus = http.StatusOK
					})
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE,
							map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodGet, "", http.StatusUnauthorized)
					})
				})

				Context("when calling get policy endpoint", func() {
					BeforeEach(func() {
						schedulerStatus = http.StatusOK
					})
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
							map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodGet, "", http.StatusUnauthorized)
					})

				})

				Context("when calling attach policy endpoint", func() {
					BeforeEach(func() {
						schedulerStatus = http.StatusOK
					})
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
							map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodPut, "", http.StatusUnauthorized)
					})

				})

				Context("when calling detach policy endpoint", func() {
					BeforeEach(func() {
						schedulerStatus = http.StatusOK
					})
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
							map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodDelete, "", http.StatusUnauthorized)
					})

				})
			})

			Describe("With Invalid Client Token", func() {
				BeforeEach(func() {
					fakeCFClient.IsTokenAuthorizedReturns(false, nil)
					fakeCFClient.IsUserSpaceDeveloperReturns(true, nil)
				})

				Context("when calling scaling_histories endpoint", func() {
					BeforeEach(func() {
						scalingEngineStatus = http.StatusOK
					})
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/scaling_histories",
							map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodGet, "", http.StatusUnauthorized)
					})
				})

				Context("when calling aggregated metric endpoint", func() {
					BeforeEach(func() {
						eventGeneratorStatus = http.StatusOK
					})
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE,
							map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodGet, "", http.StatusUnauthorized)
					})
				})

				Context("when calling get policy endpoint", func() {
					BeforeEach(func() {
						schedulerStatus = http.StatusOK
					})
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
							map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodGet, "", http.StatusUnauthorized)
					})

				})

				Context("when calling attach policy endpoint", func() {
					BeforeEach(func() {
						schedulerStatus = http.StatusOK
					})
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
							map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodPut, "", http.StatusUnauthorized)
					})

				})

				Context("when calling detach policy endpoint", func() {
					BeforeEach(func() {
						schedulerStatus = http.StatusOK
					})
					It("should fail with 401", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
							map[string]string{"Authorization": TEST_USER_TOKEN}, http.MethodDelete, "", http.StatusUnauthorized)
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
			})

			Describe("With valid authorization token", func() {
				BeforeEach(func() {
					fakeCFClient.IsTokenAuthorizedReturns(true, nil)
					fakeCFClient.IsUserSpaceDeveloperReturns(true, nil)
				})

				Context("when calling scaling_histories endpoint", func() {
					BeforeEach(func() {
						scalingEngineStatus = http.StatusOK
					})
					It("should succeed", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/scaling_histories",
							map[string]string{"Authorization": TEST_USER_TOKEN, "X-Autoscaler-Token": TEST_CLIENT_TOKEN}, http.MethodGet, "", http.StatusOK)
					})
				})

				Context("when calling aggregated metric endpoint", func() {
					BeforeEach(func() {
						eventGeneratorStatus = http.StatusOK
					})

					It("should succeed", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE,
							map[string]string{"Authorization": TEST_USER_TOKEN, "X-Autoscaler-Token": TEST_CLIENT_TOKEN}, http.MethodGet, "", http.StatusOK)
					})
				})

				Context("when calling get policy endpoint", func() {
					JustBeforeEach(func() {
						schedulerStatus = http.StatusOK
						fakePolicyDB.GetAppPolicyReturns(&models.ScalingPolicy{
							InstanceMax: 5,
							InstanceMin: 1,
							ScalingRules: []*models.ScalingRule{
								{
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
							map[string]string{"Authorization": TEST_USER_TOKEN, "X-Autoscaler-Token": TEST_CLIENT_TOKEN}, http.MethodGet, "", http.StatusOK)
					})

				})

				Context("when calling attach policy endpoint", func() {
					BeforeEach(func() {
						schedulerStatus = http.StatusOK
					})

					It("should succeed", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
							map[string]string{"Authorization": TEST_USER_TOKEN, "X-Autoscaler-Token": TEST_CLIENT_TOKEN}, http.MethodPut, policy, http.StatusOK)
					})

				})

				Context("when calling detach policy endpoint", func() {
					BeforeEach(func() {
						schedulerStatus = http.StatusOK
					})
					It("should succeed", func() {
						verifyResponse(httpClient, serverUrl, "/v1/apps/"+TEST_APP_ID+"/policy",
							map[string]string{"Authorization": TEST_USER_TOKEN, "X-Autoscaler-Token": TEST_CLIENT_TOKEN}, http.MethodPut, policy, http.StatusOK)
					})
				})
			})
		})
	})

	Describe("CreateHealthServer", func() {
		JustBeforeEach(func() {
			publicApiServer := publicapiserver.NewPublicApiServer(
				lagertest.NewTestLogger("public_apiserver"), conf, fakePolicyDB,
				fakeBindingDB, fakeCredentials, checkBindingFunc, fakeCFClient,
				httpStatusCollector, fakeRateLimiter, fakeBrokerServer)

			httpServer, err := publicApiServer.CreateHealthServer()
			Expect(err).NotTo(HaveOccurred())
			serverProcess = ginkgomon_v2.Invoke(httpServer)
		})

		It("should succeed", func() {
			res := verifyResponse(httpClient, healthUrl, "/health", nil, http.MethodGet, "", http.StatusOK)
			Expect(res).To(ContainSubstring("autoscaler_golangapiserver_bindingDB_idle"))
		})
	})

	Describe("CreateCFServer", func() {
		JustBeforeEach(func() {
			eventGeneratorResponse = []models.AppMetric{
				{
					AppId:      TEST_APP_ID,
					Timestamp:  100,
					MetricType: TEST_METRIC_TYPE,
					Unit:       TEST_METRIC_UNIT,
					Value:      "200",
				},
			}
			publicApiServer := publicapiserver.NewPublicApiServer(
				lagertest.NewTestLogger("public_apiserver"), conf, fakePolicyDB,
				fakeBindingDB, fakeCredentials, checkBindingFunc, fakeCFClient,
				httpStatusCollector, fakeRateLimiter, fakeBrokerServer)

			httpServer, err := publicApiServer.CreateCFServer()
			Expect(err).NotTo(HaveOccurred())
			serverProcess = ginkgomon_v2.Invoke(httpServer)
		})

		Context("when calling info endpoint", func() {
			It("should succeed", func() {
				verifyResponse(httpClient, cfServerUrl, "/v1/info", nil, http.MethodGet, "", http.StatusOK)
			})
		})

		Context("when calling health endpoint", func() {
			It("should succeed", func() {
				res := verifyResponse(httpClient, cfServerUrl, "/health", nil, http.MethodGet, "", http.StatusOK)
				Expect(res).To(ContainSubstring("alive"))
			})
		})

		When("calling broker endpoint", func() {
			BeforeEach(func() {
				router := chi.NewRouter()
				router.Get("/v2/catalog", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`Service Broker`))
					Expect(err).NotTo(HaveOccurred())
				})

				fakeBrokerServer.GetRouterReturns(router, nil)
			})

			It("should health, broker and api on the same server", func() {
				res := verifyResponse(httpClient, cfServerUrl, "/v2/catalog", nil, http.MethodGet, "", http.StatusOK)
				Expect(res).To(ContainSubstring("Service Broker"))
			})
		})
	})
})

func verifyResponse(httpClient *http.Client, url *url.URL, path string, headers map[string]string, httpRequestMethod string, httpRequestBody string, expectResponseStatusCode int) string {
	url.Path = path
	var body io.Reader = nil
	if httpRequestBody != "" {
		body = strings.NewReader(httpRequestBody)
	}
	req, err := http.NewRequest(httpRequestMethod, url.String(), body)
	if len(headers) > 0 {
		for headerName, headerValue := range headers {
			req.Header.Set(headerName, headerValue)
		}
	}
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	resp, err := httpClient.Do(req)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
	}
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, resp.StatusCode).To(Equal(expectResponseStatusCode))

	respBody, err := io.ReadAll(resp.Body)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	return string(respBody)
}
