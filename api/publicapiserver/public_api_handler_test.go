package publicapiserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/publicapiserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("PublicApiHandler", func() {

	var eventGeneratorHandler http.HandlerFunc

	JustBeforeEach(func() {
		eventGeneratorPathMatcher, err := regexp.Compile(`/v1/apps/[A-Za-z0-9\-]+/aggregated_metric_histories/[a-zA-Z0-9_]+`)
		Expect(err).NotTo(HaveOccurred())
		eventGeneratorServer.RouteToHandler(http.MethodGet, eventGeneratorPathMatcher, eventGeneratorHandler)
	})

	BeforeEach(func() {
		eventGeneratorHandler = ghttp.RespondWithJSONEncodedPtr(&eventGeneratorStatus, &eventGeneratorResponse)
	})

	const (
		InvalidPolicyStr = `{
			"instance_max_count":4,
			"scaling_rules":[
			{
				"metric_type":"memoryused",
				"threshold":30,
				"operator":"<",
				"adjustment":"-1"
			}]
		}`
		ValidPolicyStr = `{
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
		ValidPolicyStrWithExtraFields = `{
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
		InvalidCustomMetricsConfigurationStr = `{
		  "configuration": {
			"custom_metrics": {
			  "metric_submission_strategy": {
				"allow_from": "same_app"
			  }
			}
		  },
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
		validCustomMetricsConfigurationStr = `{
		  "configuration": {
			"custom_metrics": {
			  "metric_submission_strategy": {
				"allow_from": "bound_app"
			  }
			}
		  },
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
		bindingdb = &fakes.FakeBindingDB{}
		resp = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/v1/info", nil)
		pathVariables = map[string]string{}
	})

	JustBeforeEach(func() {
		handler = NewPublicApiHandler(lagertest.NewTestLogger("public_api_handler"), conf, policydb, bindingdb, credentials)
	})

	Describe("GetInfo", func() {
		JustBeforeEach(func() {
			handler.GetApiInfo(resp, req, map[string]string{})
		})
		When("GetApiInfo is called", func() {
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
		When("GetHealth is called", func() {
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

		When("appId is not present", func() {
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"AppId is required"}`))
			})
		})
		When("database gives error", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				policydb.GetAppPolicyReturns(nil, fmt.Errorf("Failed to retrieve policy"))
			})
			It("should fail with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error retrieving scaling policy"}`))
			})
		})

		When("policy doesn't exist", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				policydb.GetAppPolicyReturns(nil, nil)
			})
			It("should fail with 404", func() {
				Expect(resp.Code).To(Equal(http.StatusNotFound))
				Expect(resp.Body.String()).To(Equal(`{"code":"Not Found","message":"Policy Not Found"}`))
			})
		})

		When("policy exist", func() {
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
		Context("and custom metric strategy", func() {
			When("custom metric strategy retrieval fails", func() {
				BeforeEach(func() {
					pathVariables["appId"] = TEST_APP_ID
					setupPolicy(policydb)
					bindingdb.GetCustomMetricStrategyByAppIdReturns("", fmt.Errorf("db error"))
				})
				It("should fail with 500", func() {
					Expect(resp.Code).To(Equal(http.StatusInternalServerError))
					Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error retrieving binding policy"}`))
				})
			})
			When("custom metric strategy retrieved successfully", func() {
				BeforeEach(func() {
					pathVariables["appId"] = TEST_APP_ID
					bindingdb.GetCustomMetricStrategyByAppIdReturns("bound_app", nil)
				})
				When("custom metric strategy and policy are present", func() {
					BeforeEach(func() {
						setupPolicy(policydb)
					})
					It("should return combined configuration with 200", func() {
						Expect(resp.Code).To(Equal(http.StatusOK))
						Expect(resp.Body.String()).To(MatchJSON(validCustomMetricsConfigurationStr))
					})
					When("policy is present only", func() {
						BeforeEach(func() {
							setupPolicy(policydb)
							bindingdb.GetCustomMetricStrategyByAppIdReturns("", nil)
						})
						It("should return policy with 200", func() {
							Expect(resp.Code).To(Equal(http.StatusOK))
							Expect(resp.Body.String()).To(MatchJSON(ValidPolicyStr))
						})
					})
				})
			})
		})
	})

	Describe("AttachScalingPolicy", func() {
		JustBeforeEach(func() {
			handler.AttachScalingPolicy(resp, req, pathVariables)
		})

		When("appId is not present", func() {
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"AppId is required"}`))
			})
		})
		When("the policy is invalid", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(InvalidPolicyStr))
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`[{"context":"(root)","description":"instance_min_count is required"}]`))
			})
		})

		When("save policy errors", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(ValidPolicyStr))
				policydb.SaveAppPolicyReturns(fmt.Errorf("Failed to save policy"))
			})
			It("should fail with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error saving policy"}`))
			})
		})

		When("scheduler returns non 200 and non 204 status code", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(ValidPolicyStr))
				schedulerStatus = 500
				msg, err := json.Marshal([]string{"err one", "err two"})
				Expect(err).ToNot(HaveOccurred())
				schedulerErrJson = string(msg)
			})
			It("should fail with valid error", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))

				Expect(resp.Body.String()).To(MatchJSON(`{"code":"Internal Server Error","message":"unable to creation/update schedule: [\"err one\",\"err two\"]"}`))
			})
		})

		When("scheduler returns 200 status code", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(ValidPolicyStr))
				schedulerStatus = 200
			})
			It("should succeed", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body.String()).To(MatchJSON(ValidPolicyStr))
			})
		})

		When("providing extra fields", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(ValidPolicyStrWithExtraFields))
				schedulerStatus = 200
			})
			It("should succeed and ignore the extra fields", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body).To(MatchJSON(ValidPolicyStr))
			})
		})

		When("scheduler returns 204 status code", func() {
			BeforeEach(func() {
				pathVariables["appId"] = TEST_APP_ID
				req, _ = http.NewRequest(http.MethodPut, "", bytes.NewBufferString(ValidPolicyStr))
				schedulerStatus = 204
			})
			It("should succeed", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("Binding Configuration", func() {
			When("reading binding configuration from request fails", func() {
				BeforeEach(func() {
					req = setupRequest("incorrect.json", TEST_APP_ID, pathVariables)
				})
				It("should not succeed and fail with 400", func() {
					Expect(resp.Body.String()).To(ContainSubstring("invalid character"))
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})

			When("invalid configuration is provided with the policy", func() {
				BeforeEach(func() {
					req = setupRequest(InvalidCustomMetricsConfigurationStr, TEST_APP_ID, pathVariables)
					schedulerStatus = 200
				})
				It("should not succeed and fail with 400", func() {
					Expect(resp.Body.String()).To(MatchJSON(`[{"context":"(root).configuration.custom_metrics.metric_submission_strategy.allow_from","description":"configuration.custom_metrics.metric_submission_strategy.allow_from must be one of the following: \"bound_app\""}]`))
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})
		})

		When("save configuration returned error", func() {
			BeforeEach(func() {
				req = setupRequest(validCustomMetricsConfigurationStr, TEST_APP_ID, pathVariables)
				schedulerStatus = 200
				bindingdb.SetOrUpdateCustomMetricStrategyReturns(fmt.Errorf("failed to save custom metrics configuration"))
			})
			It("should not succeed and fail with internal server error", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(MatchJSON(`{"code":"Internal Server Error","message":"failed to save custom metric submission strategy in the database"}`))
			})
		})

		When("valid configuration is provided with the policy", func() {
			BeforeEach(func() {
				req = setupRequest(validCustomMetricsConfigurationStr, TEST_APP_ID, pathVariables)
				schedulerStatus = 200
			})
			It("returns the policy and configuration with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body).To(MatchJSON(validCustomMetricsConfigurationStr))
			})
		})
		When("configuration is removed but only policy is provided", func() {
			BeforeEach(func() {
				req = setupRequest(ValidPolicyStr, TEST_APP_ID, pathVariables)
				schedulerStatus = 200
			})
			It("returns the policy 200", func() {
				Expect(resp.Body).To(MatchJSON(ValidPolicyStr))
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("DetachScalingPolicy", func() {
		BeforeEach(func() {
			req, _ = http.NewRequest(http.MethodDelete, "", nil)
			pathVariables["appId"] = TEST_APP_ID
		})
		JustBeforeEach(func() {
			handler.DetachScalingPolicy(resp, req, pathVariables)
		})

		When("appId is not present", func() {
			BeforeEach(func() {
				delete(pathVariables, "appId")
			})
			It("should fail with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"AppId is required"}`))
			})
		})

		When("delete policy errors", func() {
			BeforeEach(func() {
				policydb.DeletePolicyReturns(fmt.Errorf("Failed to save policy"))
			})
			It("should fail with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error deleting policy"}`))
			})
		})

		When("scheduler returns non 200 and non 204 status code", func() {
			BeforeEach(func() {
				schedulerStatus = 500
			})
			It("should fail with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error deleting schedules"}`))
			})
		})

		When("scheduler returns 200 status code", func() {
			BeforeEach(func() {
				schedulerStatus = 200
			})
			It("should succeed", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})

			When("the service is offered in brokered mode", func() {
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
							DefaultPolicy:     ValidPolicyStr,
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
							c, a, p, g := policydb.SaveAppPolicyArgsForCall(0)
							Expect(c).NotTo(BeNil())
							Expect(a).To(Equal(TEST_APP_ID))
							Expect(p).To(MatchJSON(ValidPolicyStr))
							Expect(g).To(Equal("test-policy-guid"))
						})
					})
				})
			})
		})

		When("scheduler returns 204 status code", func() {
			BeforeEach(func() {
				schedulerStatus = 204
				bindingdb.GetServiceInstanceByAppIdReturns(&models.ServiceInstance{}, nil)
			})
			It("should succeed", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("Custom Metrics Strategy Submission Configuration", func() {
			When("delete configuration in db return errors", func() {
				BeforeEach(func() {
					schedulerStatus = 200
					bindingdb.SetOrUpdateCustomMetricStrategyReturns(fmt.Errorf("failed to delete custom metric submission strategy in the database"))
				})
				It("should not succeed and fail with 500", func() {
					Expect(resp.Code).To(Equal(http.StatusInternalServerError))
					Expect(resp.Body.String()).To(MatchJSON(`{"code":"Internal Server Error","message":"failed to delete custom metric submission strategy in the database"}`))
				})
			})
			When("binding exist for a valid app", func() {
				BeforeEach(func() {
					schedulerStatus = 200
					bindingdb.SetOrUpdateCustomMetricStrategyReturns(nil)
				})
				It("delete the custom metric strategy and returns 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
					Expect(resp.Body.String()).To(Equal(`{}`))
				})
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

		When("CF_INSTANCE_CERT is not set", func() {
			BeforeEach(func() {
				os.Unsetenv("CF_INSTANCE_CERT")
			})

			When("appId is not present", func() {
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

			When("metricType is not present", func() {
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

			When("start-time is not integer", func() {
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

			When("start-time is not provided", func() {
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

			When("end-time is not integer", func() {
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

			When("end-time is not provided", func() {
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

			When("order-direction is not provided", func() {
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

			When("order-direction is not desc or asc", func() {
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

			When("page is not integer", func() {
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

			When("page is not provided", func() {
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

			When("results-per-page is not integer", func() {
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
			When("results-per-page is not provided", func() {
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

			When("scaling engine returns 500", func() {
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

			When("getting 1st page", func() {
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
								NextUrl:      "/v1/apps/" + TEST_APP_ID + "/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=2\u0026results-per-page=2\u0026start-time=100",
							},
							Resources: eventGeneratorResponse[0:2],
						},
					))

				})
			})
			When("getting 2nd page", func() {
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
								PrevUrl:      "/v1/apps/" + TEST_APP_ID + "/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=1\u0026results-per-page=2\u0026start-time=100",
								NextUrl:      "/v1/apps/" + TEST_APP_ID + "/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=3\u0026results-per-page=2\u0026start-time=100",
							},
							Resources: eventGeneratorResponse[2:4],
						},
					))
				})
			})

			When("getting 3rd page", func() {
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
								PrevUrl:      "/v1/apps/" + TEST_APP_ID + "/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=2\u0026results-per-page=2\u0026start-time=100",
								NextUrl:      "",
							},
							Resources: eventGeneratorResponse[4:5],
						},
					))
				})
			})

			When("getting 4th page", func() {
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
								PrevUrl:      "/v1/apps/" + TEST_APP_ID + "/aggregated_metric_histories/test_metric?end-time=300\u0026order-direction=desc\u0026page=3\u0026results-per-page=2\u0026start-time=100",
								NextUrl:      "",
							},
							Resources: []models.AppMetric{},
						},
					))
				})
			})
		})

	})
})

func setupRequest(requestBody, appId string, pathVariables map[string]string) *http.Request {
	pathVariables["appId"] = appId
	req, _ := http.NewRequest(http.MethodPut, "", bytes.NewBufferString(requestBody))
	return req
}
func setupPolicy(policyDb *fakes.FakePolicyDB) {
	policyDb.GetAppPolicyReturns(&models.ScalingPolicy{
		InstanceMax: 5,
		InstanceMin: 1,
		ScalingRules: []*models.ScalingRule{
			{
				MetricType:            "memoryused",
				BreachDurationSeconds: 300,
				CoolDownSeconds:       300,
				Threshold:             30,
				Operator:              ">",
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
}
