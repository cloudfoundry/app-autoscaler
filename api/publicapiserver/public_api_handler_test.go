package publicapiserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/publicapiserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PublicApiHandler", func() {
	const (
		INVALID_POLICY_STR = `{
			"instance_max_count":4,
			"scaling_rules":[
			{
				"metric_type":"memoryused",
				"threshold":30,
				"operator":"<",
				"adjustment":"-1"
			}]
		}`
		VALID_POLICY_STR = `{
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
		VALID_POLICY_STR_WITH_EXTRA_FIELDS = `{
			"instance_min_count": 1,
			"instance_max_count": 5,
			"scaling_rules": [{
				"metric_type": "memoryused",
				"breach_duration_secs": 300,
				"stats_window_secs": 666,
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
			},
			"is_admin": true
		}`
	)
	var (
		policydb      *fakes.FakePolicyDB
		bindingdb     *fakes.FakeBindingDB
		credentials   *fakes.FakeCredentials
		handler       *PublicApiHandler
		resp          *httptest.ResponseRecorder
		req           *http.Request
		pathVariables map[string]string
	)
	BeforeEach(func() {
		policydb = &fakes.FakePolicyDB{}
		credentials = &fakes.FakeCredentials{}
		bindingdb = nil
		resp = httptest.NewRecorder()

		pathVariables = map[string]string{}
	})
	JustBeforeEach(func() {
		handler = NewPublicApiHandler(lagertest.NewTestLogger("public_api_handler"), conf, policydb, bindingdb, credentials)
	})

	Describe("GetInfo", func() {
		JustBeforeEach(func() {
			handler.GetApiInfo(resp, req, map[string]string{})
		})
		Context("When GetApiInfo is called", func() {
			It("gets the info json", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body.Bytes()).To(Equal(infoBytes))
			})
		})
	})

	Describe("GetHealth", func() {
		JustBeforeEach(func() {
			handler.GetHealth(resp, req, map[string]string{})
		})
		Context("When GetHealth is called", func() {
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body.String()).To(Equal(`{"alive":"true"}`))
			})
		})
	})
	Describe("GetScalingPolicy", func() {
		JustBeforeEach(func() {
			handler.GetScalingPolicy(resp, req, pathVariables)
		})

		Context("When appId is not present", func() {
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"AppId is required"}`))
			})
		})
		Context("When database gives error", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				policydb.GetAppPolicyReturns(nil, fmt.Errorf("Failed to retrieve policy"))
			})
			It("should fail with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error retrieving scaling policy"}`))
			})
		})

		Context("When policy doesn't exist", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				policydb.GetAppPolicyReturns(nil, nil)
			})
			It("should fail with 404", func() {
				Expect(resp.Code).To(Equal(http.StatusNotFound))
				Expect(resp.Body.String()).To(Equal(`{"code":"Not Found","message":"Policy Not Found"}`))
			})
		})

		Context("When policy exist", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				policydb.GetAppPolicyReturns(&models.ScalingPolicy{
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
					Schedules: &models.ScalingSchedules{
						Timezone: "Asia/Kolkata",
						RecurringSchedules: []*models.RecurringSchedule{
							{
								StartTime:             "10:00",
								EndTime:               "18:00",
								DaysOfWeek:            []int{1, 2, 3},
								ScheduledInstanceMin:  1,
								ScheduledInstanceMax:  10,
								ScheduledInstanceInit: 5,
							},
						},
					},
				}, nil)
			})
			It("should succeed", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))

				Expect(strings.TrimSpace(resp.Body.String())).To(Equal(`{"instance_min_count":1,"instance_max_count":5,"scaling_rules":[{"metric_type":"memoryused","breach_duration_secs":300,"threshold":30,"operator":"<","cool_down_secs":300,"adjustment":"-1"}],"schedules":{"timezone":"Asia/Kolkata","recurring_schedule":[{"start_time":"10:00","end_time":"18:00","days_of_week":[1,2,3],"instance_min_count":1,"instance_max_count":10,"initial_min_instance_count":5}]}}`))
			})
		})
	})

	Describe("AttachScalingPolicy", func() {
		JustBeforeEach(func() {
			handler.AttachScalingPolicy(resp, req, pathVariables)
		})

		Context("When appId is not present", func() {
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"AppId is required"}`))
			})
		})
		Context("When the policy is invalid", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(INVALID_POLICY_STR))
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`[{"context":"(root)","description":"instance_min_count is required"}]`))
			})
		})

		Context("When save policy errors", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(VALID_POLICY_STR))
				policydb.SaveAppPolicyReturns(fmt.Errorf("Failed to save policy"))
			})
			It("should fail with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error saving policy"}`))
			})
		})

		Context("When scheduler returns non 200 and non 204 status code", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(VALID_POLICY_STR))
				schedulerStatus = 500
			})
			It("should succeed", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When scheduler returns 200 status code", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(VALID_POLICY_STR))
				schedulerStatus = 200
			})
			It("should succeed", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When providing extra fields", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(VALID_POLICY_STR_WITH_EXTRA_FIELDS))
				schedulerStatus = 200
			})
			It("should succeed and ignore the extra fields", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body).To(MatchJSON(VALID_POLICY_STR))
			})
		})

		Context("When scheduler returns 204 status code", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(VALID_POLICY_STR))
				schedulerStatus = 204
			})
			It("should succeed", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("DetachScalingPolicy", func() {
		JustBeforeEach(func() {
			handler.DetachScalingPolicy(resp, req, pathVariables)
		})

		Context("When appId is not present", func() {
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"AppId is required"}`))
			})
		})

		Context("When delete policy errors", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodDelete, "", nil)
				policydb.DeletePolicyReturns(fmt.Errorf("Failed to save policy"))
			})
			It("should fail with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error deleting policy"}`))
			})
		})

		Context("When scheduler returns non 200 and non 204 status code", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", nil)
				schedulerStatus = 500
			})
			It("should fail with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error deleting schedules"}`))
			})
		})

		Context("When scheduler returns 200 status code", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", nil)
				schedulerStatus = 200
			})
			It("should succeed", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})

			Context("when the service is offered in brokered mode", func() {
				BeforeEach(func() {
					bindingdb = &fakes.FakeBindingDB{}
				})
				Context("but there is no service instance", func() {
					BeforeEach(func() {
						bindingdb.GetServiceInstanceByAppIdReturns(nil, db.ErrDoesNotExist)
					})
					It("should error", func() {
						Expect(resp.Code).To(Equal(http.StatusInternalServerError))
						Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error retrieving service instance"}`))
					})
				})
				Context("and there is a service instance without a default policy", func() {
					BeforeEach(func() {
						bindingdb.GetServiceInstanceByAppIdReturns(&models.ServiceInstance{}, nil)
					})
					It("should still succeed", func() {
						Expect(resp.Code).To(Equal(http.StatusOK))
					})
				})
				Context("and there is a service instance with a default policy", func() {
					BeforeEach(func() {
						bindingdb.GetServiceInstanceByAppIdReturns(&models.ServiceInstance{
							DefaultPolicy:     VALID_POLICY_STR,
							DefaultPolicyGuid: "test-policy-guid",
						}, nil)
					})
					Context("and setting the default policy fails", func() {
						BeforeEach(func() {
							policydb.SaveAppPolicyReturns(fmt.Errorf("failed to save new (default) policy"))
						})
						It("should error", func() {
							Expect(resp.Code).To(Equal(http.StatusInternalServerError))
							Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error attaching the default policy"}`))
						})
					})
					Context("and setting the default policy succeeds", func() {
						It("should succeed and set the default policy", func() {
							Expect(resp.Code).To(Equal(http.StatusOK))
							Expect(policydb.SaveAppPolicyCallCount()).To(Equal(1))
							a, p, g := policydb.SaveAppPolicyArgsForCall(0)
							Expect(a).To(Equal(TEST_APP_ID))
							Expect(p).To(Equal(VALID_POLICY_STR))
							Expect(g).To(Equal("test-policy-guid"))
						})
					})
				})
			})
		})

		Context("When scheduler returns 204 status code", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", nil)
				schedulerStatus = 204
			})
			It("should succeed", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("GetScalingHistories", func() {
		JustBeforeEach(func() {
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
				{
					AppId:        TEST_APP_ID,
					Timestamp:    250,
					ScalingType:  1,
					Status:       1,
					OldInstances: 2,
					NewInstances: 4,
					Reason:       "a reason",
					Message:      "",
					Error:        "",
				},
				{
					AppId:        TEST_APP_ID,
					Timestamp:    200,
					ScalingType:  0,
					Status:       0,
					OldInstances: 2,
					NewInstances: 4,
					Reason:       "a reason",
					Message:      "",
					Error:        "",
				},
				{
					AppId:        TEST_APP_ID,
					Timestamp:    150,
					ScalingType:  1,
					Status:       1,
					OldInstances: 2,
					NewInstances: 4,
					Reason:       "a reason",
					Message:      "",
					Error:        "",
				},
				{
					AppId:        TEST_APP_ID,
					Timestamp:    100,
					ScalingType:  0,
					Status:       0,
					OldInstances: 2,
					NewInstances: 4,
					Reason:       "a reason",
					Message:      "",
					Error:        "",
				},
			}
			handler.GetScalingHistories(resp, req, pathVariables)
		})

		Context("When appId is not present", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"appId is required"}`))
			})
		})

		Context("When start-time is not integer", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "not-integer")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"start-time must be an integer"}`))
			})
		})

		Context("When start-time is not provided", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When end-time is not integer", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "not-integer")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"end-time must be an integer"}`))
			})
		})

		Context("When end-time is not provided", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When order-direction is not provided", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When order-direction is not desc or asc", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "not-asc-desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"order-direction must be DESC or ASC"}`))
			})
		})

		Context("When page is not integer", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "not-integer")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"page must be an integer"}`))
			})
		})

		Context("When page is not provided", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when results-per-page is not integer", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "not-integer")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"results-per-page must be an integer"}`))
			})
		})
		Context("when results-per-page is not provided", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when scaling engine returns 500", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusInternalServerError
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when getting 1st page", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should get full page", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.AppScalingHistoryResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.AppScalingHistoryResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         1,
							PrevUrl:      "",
							NextUrl:      "/v1/apps/test-app-id/scaling_histories/?end-time=300\u0026order-direction=desc\u0026page=2\u0026results-per-page=2\u0026start-time=100",
						},
						Resources: scalingEngineResponse[0:2],
					},
				))

			})
		})
		Context("when getting 2nd page", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "2")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should get full page", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.AppScalingHistoryResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.AppScalingHistoryResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         2,
							PrevUrl:      "/v1/apps/test-app-id/scaling_histories/?end-time=300\u0026order-direction=desc\u0026page=1\u0026results-per-page=2\u0026start-time=100",
							NextUrl:      "/v1/apps/test-app-id/scaling_histories/?end-time=300\u0026order-direction=desc\u0026page=3\u0026results-per-page=2\u0026start-time=100",
						},
						Resources: scalingEngineResponse[2:4],
					},
				))
			})
		})

		Context("when getting 3rd page", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "3")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should get only one record", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.AppScalingHistoryResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.AppScalingHistoryResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         3,
							PrevUrl:      "/v1/apps/test-app-id/scaling_histories/?end-time=300\u0026order-direction=desc\u0026page=2\u0026results-per-page=2\u0026start-time=100",
							NextUrl:      "",
						},
						Resources: scalingEngineResponse[4:5],
					},
				))
			})
		})

		Context("when getting 4th page", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "4")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories/?"+params.Encode(), nil)
			})
			It("should get no records", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.AppScalingHistoryResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.AppScalingHistoryResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         4,
							PrevUrl:      "/v1/apps/test-app-id/scaling_histories/?end-time=300\u0026order-direction=desc\u0026page=3\u0026results-per-page=2\u0026start-time=100",
							NextUrl:      "",
						},
						Resources: []models.AppScalingHistory{},
					},
				))
			})
		})

	})

	Describe("GetInstanceMetricsHistories", func() {
		JustBeforeEach(func() {
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
				{
					AppId:         TEST_APP_ID,
					Timestamp:     110,
					InstanceIndex: 1,
					CollectedAt:   1,
					Name:          TEST_METRIC_TYPE,
					Unit:          TEST_METRIC_UNIT,
					Value:         "250",
				},
				{
					AppId:         TEST_APP_ID,
					Timestamp:     150,
					InstanceIndex: 0,
					CollectedAt:   0,
					Name:          TEST_METRIC_TYPE,
					Unit:          TEST_METRIC_UNIT,
					Value:         "250",
				},
				{
					AppId:         TEST_APP_ID,
					Timestamp:     170,
					InstanceIndex: 1,
					CollectedAt:   1,
					Name:          TEST_METRIC_TYPE,
					Unit:          TEST_METRIC_UNIT,
					Value:         "200",
				},
				{
					AppId:         TEST_APP_ID,
					Timestamp:     120,
					InstanceIndex: 0,
					CollectedAt:   0,
					Name:          TEST_METRIC_TYPE,
					Unit:          TEST_METRIC_UNIT,
					Value:         "200",
				},
			}
			handler.GetInstanceMetricsHistories(resp, req, pathVariables)
		})

		Context("When appId is not present", func() {
			BeforeEach(func() {
				pathVariables["metricType"] = TEST_METRIC_TYPE

				metricsCollectorStatus = http.StatusOK
				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"appId is required"}`))
			})
		})

		Context("When metricType is not present", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Metrictype is required"}`))
			})
		})

		Context("When start-time is not integer", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "not-integer")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"start-time must be an integer"}`))
			})
		})

		Context("When start-time is not provided", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When end-time is not integer", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "not-integer")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"end-time must be an integer"}`))
			})
		})

		Context("When end-time is not provided", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When order-direction is not provided", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When order-direction is not desc or asc", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "not-asc-desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"order-direction must be DESC or ASC"}`))
			})
		})

		Context("When page is not integer", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "not-integer")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"page must be an integer"}`))
			})
		})

		Context("When page is not provided", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when results-per-page is not integer", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "not-integer")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"results-per-page must be an integer"}`))
			})
		})
		Context("when results-per-page is not provided", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when scaling engine returns 500", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusInternalServerError
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when getting 1st page", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should get full page", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.InstanceMetricResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.InstanceMetricResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         1,
							PrevUrl:      "",
							NextUrl:      "/v1/apps/test-app-id/metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=2\u0026results-per-page=2\u0026start-time=100",
						},
						Resources: metricsCollectorResponse[0:2],
					},
				))
			})
		})
		Context("when getting 2nd page", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "2")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should get full page", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.InstanceMetricResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.InstanceMetricResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         2,
							PrevUrl:      "/v1/apps/test-app-id/metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=1\u0026results-per-page=2\u0026start-time=100",
							NextUrl:      "/v1/apps/test-app-id/metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=3\u0026results-per-page=2\u0026start-time=100",
						},
						Resources: metricsCollectorResponse[2:4],
					},
				))

			})
		})

		Context("when getting 3rd page", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "3")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should get only one record", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.InstanceMetricResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.InstanceMetricResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         3,
							PrevUrl:      "/v1/apps/test-app-id/metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=2\u0026results-per-page=2\u0026start-time=100",
							NextUrl:      "",
						},
						Resources: metricsCollectorResponse[4:5],
					},
				))

			})
		})

		Context("when getting 4th page", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "4")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should get no records", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.InstanceMetricResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.InstanceMetricResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         4,
							PrevUrl:      "/v1/apps/test-app-id/metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=3\u0026results-per-page=2\u0026start-time=100",
							NextUrl:      "",
						},
						Resources: []models.AppInstanceMetric{},
					},
				))
			})
		})

	})

	Describe("GetAggregatedMetricsHistories", func() {
		JustBeforeEach(func() {
			eventGeneratorResponse = []models.AppMetric{
				{
					AppId:      TEST_APP_ID,
					Timestamp:  100,
					MetricType: TEST_METRIC_TYPE,
					Unit:       TEST_METRIC_UNIT,
					Value:      "200",
				},
				{
					AppId:      TEST_APP_ID,
					Timestamp:  110,
					MetricType: TEST_METRIC_TYPE,
					Unit:       TEST_METRIC_UNIT,
					Value:      "250",
				},
				{
					AppId:      TEST_APP_ID,
					Timestamp:  150,
					MetricType: TEST_METRIC_TYPE,
					Unit:       TEST_METRIC_UNIT,
					Value:      "250",
				},
				{
					AppId:      TEST_APP_ID,
					Timestamp:  170,
					MetricType: TEST_METRIC_TYPE,
					Unit:       TEST_METRIC_UNIT,
					Value:      "200",
				},
				{
					AppId:      TEST_APP_ID,
					Timestamp:  200,
					MetricType: TEST_METRIC_TYPE,
					Unit:       TEST_METRIC_UNIT,
					Value:      "200",
				},
			}
			handler.GetAggregatedMetricsHistories(resp, req, pathVariables)
		})

		Context("When appId is not present", func() {
			BeforeEach(func() {
				pathVariables["metricType"] = TEST_METRIC_TYPE

				eventGeneratorStatus = http.StatusOK
				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"appId is required"}`))
			})
		})

		Context("When metricType is not present", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Metrictype is required"}`))
			})
		})

		Context("When start-time is not integer", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "not-integer")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"start-time must be an integer"}`))
			})
		})

		Context("When start-time is not provided", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When end-time is not integer", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "not-integer")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"end-time must be an integer"}`))
			})
		})

		Context("When end-time is not provided", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("order-direction", "desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When order-direction is not provided", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When order-direction is not desc or asc", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "not-asc-desc")
				params.Add("page", "1")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"order-direction must be DESC or ASC"}`))
			})
		})

		Context("When page is not integer", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("page", "not-integer")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"page must be an integer"}`))
			})
		})

		Context("When page is not provided", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when results-per-page is not integer", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "not-integer")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"results-per-page must be an integer"}`))
			})
		})
		Context("when results-per-page is not provided", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when scaling engine returns 500", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusInternalServerError
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when getting 1st page", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "1")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should get full page", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.AppMetricResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.AppMetricResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         1,
							PrevUrl:      "",
							NextUrl:      "/v1/apps/test-app-id/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=2\u0026results-per-page=2\u0026start-time=100",
						},
						Resources: eventGeneratorResponse[0:2],
					},
				))

			})
		})
		Context("when getting 2nd page", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "2")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should get full page", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.AppMetricResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.AppMetricResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         2,
							PrevUrl:      "/v1/apps/test-app-id/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=1\u0026results-per-page=2\u0026start-time=100",
							NextUrl:      "/v1/apps/test-app-id/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=3\u0026results-per-page=2\u0026start-time=100",
						},
						Resources: eventGeneratorResponse[2:4],
					},
				))
			})
		})

		Context("when getting 3rd page", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "3")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should get only one record", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.AppMetricResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.AppMetricResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         3,
							PrevUrl:      "/v1/apps/test-app-id/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=2\u0026results-per-page=2\u0026start-time=100",
							NextUrl:      "",
						},
						Resources: eventGeneratorResponse[4:5],
					},
				))
			})
		})

		Context("when getting 4th page", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK
				pathVariables["appId"] = TEST_APP_ID
				pathVariables["metricType"] = TEST_METRIC_TYPE

				params := url.Values{}
				params.Add("start-time", "100")
				params.Add("end-time", "300")
				params.Add("page", "4")
				params.Add("order-direction", "desc")
				params.Add("results-per-page", "2")

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/"+TEST_METRIC_TYPE+"?"+params.Encode(), nil)
			})
			It("should get no records", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var result models.AppMetricResponse
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(
					models.AppMetricResponse{
						PublicApiResponseBase: models.PublicApiResponseBase{
							TotalResults: 5,
							TotalPages:   3,
							Page:         4,
							PrevUrl:      "/v1/apps/test-app-id/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=3\u0026results-per-page=2\u0026start-time=100",
							NextUrl:      "",
						},
						Resources: []models.AppMetric{},
					},
				))
			})
		})

	})
	Describe("CreateCredential", func() {
		var requestBody string
		BeforeEach(func() {
			pathVariables["appId"] = TEST_APP_ID
		})
		JustBeforeEach(func() {
			req, _ = http.NewRequest(http.MethodPut, "/v1/apps/"+TEST_APP_ID+"/credential", strings.NewReader(requestBody))
			req.Header.Set("Content-type", "application/json")
			handler.CreateCredential(resp, req, pathVariables)

		})
		AfterEach(func() {
			requestBody = ""
		})
		Context("When appId is not present", func() {
			BeforeEach(func() {
				delete(pathVariables, "appId")
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"AppId is required"}`))
			})
		})
		Context("When user provide credential", func() {
			Context("When request body is invalid json", func() {
				BeforeEach(func() {
					requestBody = "not-json"
				})
				It("should fail with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Invalid credential format"}`))
				})
			})
			Context("When credential.username is not provided", func() {
				BeforeEach(func() {
					requestBody = `{"password":"password"}`
				})
				It("should fail with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Username and password are both required"}`))
				})
			})
			Context("When credential.password is not provided", func() {
				BeforeEach(func() {
					requestBody = `{"username":"username"}`
				})
				It("should fail with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Username and password are both required"}`))
				})
			})
		})
		Context("When failed to save credential to a credential store", func() {
			BeforeEach(func() {
				credentials.CreateReturns(nil, fmt.Errorf("sql db error"))
			})
			It("should fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error creating credential"}`))
			})
		})
		Context("When successfully save data to a credential store", func() {
			BeforeEach(func() {
				credentials.CreateReturns(nil, nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})
	})
	Describe("DeleteCredential", func() {
		JustBeforeEach(func() {
			handler.DeleteCredential(resp, req, pathVariables)
		})
		BeforeEach(func() {
			pathVariables["appId"] = TEST_APP_ID
			req, _ = http.NewRequest(http.MethodPut, "", nil)
		})
		Context("When appId is not present", func() {
			BeforeEach(func() {
				delete(pathVariables, "appId")
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"AppId is required"}`))
			})
		})
		Context("When failed to delete credential from a credential store", func() {
			BeforeEach(func() {
				credentials.DeleteReturns(fmt.Errorf("sql db error"))
			})
			It("should fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error deleting credential"}`))
			})
		})
		Context("When successfully delete data from a credential store", func() {
			BeforeEach(func() {
				credentials.DeleteReturns(nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

	})
})
