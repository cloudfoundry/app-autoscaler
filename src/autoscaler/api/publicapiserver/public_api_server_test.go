package publicapiserver_test

import (
	"autoscaler/models"
	"bytes"

	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PublicApiServer", func() {
	var (
		rsp *http.Response
		err error
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

		Describe("Without AuthorizatioToken", func() {
			Context("when calling scaling_histories endpoint", func() {
				BeforeEach(func() {
					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/scaling_histories"

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})

			Context("when calling instance metrics endpoint", func() {
				BeforeEach(func() {
					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/metric_histories/" + TEST_METRIC_TYPE

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})

			Context("when calling aggregated metrics endpoint", func() {
				BeforeEach(func() {
					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/aggregated_metric_histories/" + TEST_METRIC_TYPE

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})

			Context("when calling get policy endpoint", func() {
				BeforeEach(func() {
					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/policy"

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})

			Context("when calling attach policy endpoint", func() {
				BeforeEach(func() {
					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/policy"

					req, err := http.NewRequest(http.MethodPut, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})

			Context("when calling detach policy endpoint", func() {
				BeforeEach(func() {
					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/policy"

					req, err := http.NewRequest(http.MethodDelete, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
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

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/scaling_histories"

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					req.Header.Add("Authorization", TEST_INVALID_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})

			Context("when calling instance metric endpoint", func() {
				BeforeEach(func() {
					metricsCollectorStatus = http.StatusOK

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/metric_histories/" + TEST_METRIC_TYPE

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					req.Header.Add("Authorization", TEST_INVALID_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})

			Context("when calling aggregated metric endpoint", func() {
				BeforeEach(func() {
					eventGeneratorStatus = http.StatusOK

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/aggregated_metric_histories/" + TEST_METRIC_TYPE

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					req.Header.Add("Authorization", TEST_INVALID_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})

			Context("when calling get policy endpoint", func() {
				BeforeEach(func() {
					schedulerStatus = http.StatusOK

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/policy"

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					req.Header.Add("Authorization", TEST_INVALID_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})

			Context("when calling attach policy endpoint", func() {
				BeforeEach(func() {
					schedulerStatus = http.StatusOK

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/policy"

					req, err := http.NewRequest(http.MethodPut, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					req.Header.Add("Authorization", TEST_INVALID_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
				})
			})

			Context("when calling detach policy endpoint", func() {
				BeforeEach(func() {
					schedulerStatus = http.StatusOK

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/policy"

					req, err := http.NewRequest(http.MethodDelete, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					req.Header.Add("Authorization", TEST_INVALID_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				It("should fail", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
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

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/scaling_histories"

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					req.Header.Add("Authorization", TEST_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				It("should succeed", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				})
			})

			Context("when calling instance metric endpoint", func() {
				BeforeEach(func() {
					metricsCollectorStatus = http.StatusOK

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/metric_histories/" + TEST_METRIC_TYPE

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					req.Header.Add("Authorization", TEST_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				It("should succeed", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				})
			})

			Context("when calling aggregated metric endpoint", func() {
				BeforeEach(func() {
					eventGeneratorStatus = http.StatusOK

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/aggregated_metric_histories/" + TEST_METRIC_TYPE

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					req.Header.Add("Authorization", TEST_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				It("should succeed", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				})
			})

			Context("when calling get policy endpoint", func() {
				JustBeforeEach(func() {
					schedulerStatus = http.StatusOK

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/policy"

					req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())

					req.Header.Add("Authorization", TEST_USER_TOKEN)

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

					rsp, err = httpClient.Do(req)
				})
				Context("when binding is present", func() {
					BeforeEach(func() {
						fakeBindingDB.CheckServiceBindingStub = func(appId string) bool {
							return true
						}
					})
					It("should succeed", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					})
				})
				Context("when binding is not present", func() {
					BeforeEach(func() {
						fakeBindingDB.CheckServiceBindingStub = func(appId string) bool {
							return false
						}
					})
					It("should fail", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusForbidden))
					})
				})

			})

			Context("when calling attach policy endpoint", func() {
				JustBeforeEach(func() {
					schedulerStatus = http.StatusOK

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/policy"

					req, err := http.NewRequest(http.MethodPut, serverUrl.String(), bytes.NewBufferString(`{
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
					}`))
					Expect(err).NotTo(HaveOccurred())

					req.Header.Add("Authorization", TEST_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				Context("when binding is present", func() {
					BeforeEach(func() {
						fakeBindingDB.CheckServiceBindingStub = func(appId string) bool {
							return true
						}
					})
					It("should succeed", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					})
				})
				Context("when binding is not present", func() {
					BeforeEach(func() {
						fakeBindingDB.CheckServiceBindingStub = func(appId string) bool {
							return false
						}
					})
					It("should fail", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusForbidden))
					})
				})
			})

			Context("when calling detach policy endpoint", func() {
				JustBeforeEach(func() {
					schedulerStatus = http.StatusOK

					serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/policy"

					req, err := http.NewRequest(http.MethodDelete, serverUrl.String(), nil)
					Expect(err).NotTo(HaveOccurred())

					req.Header.Add("Authorization", TEST_USER_TOKEN)

					rsp, err = httpClient.Do(req)
				})
				Context("when binding is present", func() {
					BeforeEach(func() {
						fakeBindingDB.CheckServiceBindingStub = func(appId string) bool {
							return true
						}
					})
					It("should succeed", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					})
				})
				Context("when binding is not present", func() {
					BeforeEach(func() {
						fakeBindingDB.CheckServiceBindingStub = func(appId string) bool {
							return false
						}
					})
					It("should fail", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusForbidden))
					})
				})
			})
		})
	})
	Describe("UnProtected Routes", func() {
		Context("when calling info endpoint", func() {
			BeforeEach(func() {
				serverUrl.Path = "/v1/info"
				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				rsp, err = httpClient.Do(req)

			})

			It("should succeed", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		Context("when calling health endpoint", func() {
			BeforeEach(func() {
				serverUrl.Path = "/health"
				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				rsp, err = httpClient.Do(req)

			})

			It("should succeed", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})

	Context("when requesting non existing path", func() {
		BeforeEach(func() {
			serverUrl.Path = "/non-existing-path"

			req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("Authorization", TEST_USER_TOKEN)
			rsp, err = httpClient.Do(req)
		})

		It("should get 404", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})
