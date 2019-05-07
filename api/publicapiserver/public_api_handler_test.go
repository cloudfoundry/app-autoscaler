package publicapiserver_test

import (
	. "autoscaler/api/publicapiserver"
	"autoscaler/fakes"
	"autoscaler/models"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
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
	)
	var (
		policydb      *fakes.FakePolicyDB
		handler       *PublicApiHandler
		resp          *httptest.ResponseRecorder
		req           *http.Request
		pathVariables map[string]string
	)
	BeforeEach(func() {
		policydb = &fakes.FakePolicyDB{}
		resp = httptest.NewRecorder()

		pathVariables = map[string]string{}
		handler = NewPublicApiHandler(lager.NewLogger("test"), conf, policydb)
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
				Expect(resp.Body.String()).To(Equal(`{"code":"Interal-Server-Error","message":"Error retrieving scaling policy"}`))
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
						&models.ScalingRule{
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
							&models.RecurringSchedule{
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
				Expect(resp.Body.String()).To(Equal(`{"instance_min_count":1,"instance_max_count":5,"scaling_rules":[{"metric_type":"memoryused","breach_duration_secs":300,"threshold":30,"operator":"\u003c","cool_down_secs":300,"adjustment":"-1"}],"schedules":{"timezone":"Asia/Kolkata","recurring_schedule":[{"start_time":"10:00","end_time":"18:00","days_of_week":[1,2,3],"instance_min_count":1,"instance_max_count":10,"initial_min_instance_count":5}]}}`))
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
				Expect(resp.Body.String()).To(Equal(`{"code":"Interal-Server-Error","message":"Error saving policy"}`))
			})
		})

		Context("When scheduler returns non 200 and non 204 status code", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(VALID_POLICY_STR))
				schedulerStatus = 500
			})
			It("should fail with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Interal-Server-Error","message":"Error creating/updating schedules"}`))
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
				Expect(resp.Body.String()).To(Equal(`{"code":"Interal-Server-Error","message":"Error deleting policy"}`))
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
				Expect(resp.Body.String()).To(Equal(`{"code":"Interal-Server-Error","message":"Error deleting schedules"}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":1,"prev_url":"","next_url":"/v1/apps/test-app-id/scaling_histories/?end-time=300\u0026order-direction=DESC\u0026page=2\u0026results-per-page=2\u0026start-time=100","resources":[{"app_id":"test-app-id","error":"","message":"","new_instances":4,"old_instances":2,"reason":"a reason","scaling_type":0,"status":0,"timestamp":300},{"app_id":"test-app-id","error":"","message":"","new_instances":4,"old_instances":2,"reason":"a reason","scaling_type":1,"status":1,"timestamp":250}]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":2,"prev_url":"/v1/apps/test-app-id/scaling_histories/?end-time=300\u0026order-direction=DESC\u0026page=1\u0026results-per-page=2\u0026start-time=100","next_url":"/v1/apps/test-app-id/scaling_histories/?end-time=300\u0026order-direction=DESC\u0026page=3\u0026results-per-page=2\u0026start-time=100","resources":[{"app_id":"test-app-id","error":"","message":"","new_instances":4,"old_instances":2,"reason":"a reason","scaling_type":0,"status":0,"timestamp":200},{"app_id":"test-app-id","error":"","message":"","new_instances":4,"old_instances":2,"reason":"a reason","scaling_type":1,"status":1,"timestamp":150}]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":3,"prev_url":"/v1/apps/test-app-id/scaling_histories/?end-time=300\u0026order-direction=DESC\u0026page=2\u0026results-per-page=2\u0026start-time=100","next_url":"","resources":[{"app_id":"test-app-id","error":"","message":"","new_instances":4,"old_instances":2,"reason":"a reason","scaling_type":0,"status":0,"timestamp":100}]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":4,"prev_url":"/v1/apps/test-app-id/scaling_histories/?end-time=300\u0026order-direction=DESC\u0026page=3\u0026results-per-page=2\u0026start-time=100","next_url":"","resources":[]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":1,"prev_url":"","next_url":"/v1/apps/test-app-id/metric_histories/test_metric?end-time=300\u0026order-direction=DESC\u0026page=2\u0026results-per-page=2\u0026start-time=100","resources":[{"app_id":"test-app-id","collected_at":0,"instance_index":0,"name":"test_metric","timestamp":100,"unit":"test_unit","value":"200"},{"app_id":"test-app-id","collected_at":1,"instance_index":1,"name":"test_metric","timestamp":110,"unit":"test_unit","value":"250"}]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":2,"prev_url":"/v1/apps/test-app-id/metric_histories/test_metric?end-time=300\u0026order-direction=DESC\u0026page=1\u0026results-per-page=2\u0026start-time=100","next_url":"/v1/apps/test-app-id/metric_histories/test_metric?end-time=300\u0026order-direction=DESC\u0026page=3\u0026results-per-page=2\u0026start-time=100","resources":[{"app_id":"test-app-id","collected_at":0,"instance_index":0,"name":"test_metric","timestamp":150,"unit":"test_unit","value":"250"},{"app_id":"test-app-id","collected_at":1,"instance_index":1,"name":"test_metric","timestamp":170,"unit":"test_unit","value":"200"}]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":3,"prev_url":"/v1/apps/test-app-id/metric_histories/test_metric?end-time=300\u0026order-direction=DESC\u0026page=2\u0026results-per-page=2\u0026start-time=100","next_url":"","resources":[{"app_id":"test-app-id","collected_at":0,"instance_index":0,"name":"test_metric","timestamp":120,"unit":"test_unit","value":"200"}]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":4,"prev_url":"/v1/apps/test-app-id/metric_histories/test_metric?end-time=300\u0026order-direction=DESC\u0026page=3\u0026results-per-page=2\u0026start-time=100","next_url":"","resources":[]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":1,"prev_url":"","next_url":"/v1/apps/test-app-id/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=DESC\u0026page=2\u0026results-per-page=2\u0026start-time=100","resources":[{"app_id":"test-app-id","name":"test_metric","timestamp":100,"unit":"test_unit","value":"200"},{"app_id":"test-app-id","name":"test_metric","timestamp":110,"unit":"test_unit","value":"250"}]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":2,"prev_url":"/v1/apps/test-app-id/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=DESC\u0026page=1\u0026results-per-page=2\u0026start-time=100","next_url":"/v1/apps/test-app-id/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=DESC\u0026page=3\u0026results-per-page=2\u0026start-time=100","resources":[{"app_id":"test-app-id","name":"test_metric","timestamp":150,"unit":"test_unit","value":"250"},{"app_id":"test-app-id","name":"test_metric","timestamp":170,"unit":"test_unit","value":"200"}]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":3,"prev_url":"/v1/apps/test-app-id/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=DESC\u0026page=2\u0026results-per-page=2\u0026start-time=100","next_url":"","resources":[{"app_id":"test-app-id","name":"test_metric","timestamp":200,"unit":"test_unit","value":"200"}]}`))
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
				Expect(resp.Body.String()).To(Equal(`{"total_results":5,"total_pages":3,"page":4,"prev_url":"/v1/apps/test-app-id/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=DESC\u0026page=3\u0026results-per-page=2\u0026start-time=100","next_url":"","resources":[]}`))
			})
		})

	})
})
