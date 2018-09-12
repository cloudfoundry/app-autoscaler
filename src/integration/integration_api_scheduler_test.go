package integration

import (
	"autoscaler/cf"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Integration_Api_Scheduler", func() {
	var (
		appId             string
		policyStr         []byte
		initInstanceCount int = 2
		serviceInstanceId string
		bindingId         string
		orgId             string
		spaceId           string
	)

	BeforeEach(func() {
		startFakeCCNOAAUAA(initInstanceCount)
		initializeHttpClient("api.crt", "api.key", "autoscaler-ca.crt", apiSchedulerHttpRequestTimeout)
		initializeHttpClientForPublicApi("api_public.crt", "api_public.key", "autoscaler-ca.crt", apiMetricsCollectorHttpRequestTimeout)

		schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir, strings.Split(consulRunner.Address(), ":")[1])
		startScheduler()

		serviceBrokerConfPath = components.PrepareServiceBrokerConfig(components.Ports[ServiceBroker], components.Ports[ServiceBrokerInternal], brokerUserName, brokerPassword, false, dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[APIServer]), brokerApiHttpRequestTimeout, tmpDir)
		startServiceBroker()

		serviceInstanceId = getRandomId()
		orgId = getRandomId()
		spaceId = getRandomId()
		bindingId = getRandomId()
		appId = getRandomId()
		brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
	})

	AfterEach(func() {
		stopServiceBroker()
		stopScheduler()
	})

	Describe("When offered as a service", func() {

		BeforeEach(func() {
			apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], components.Ports[APIPublicServer], false, 200, fakeCCNOAAUAA.URL(), dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ServiceBrokerInternal]), true, tmpDir)
			startApiServer()

			resp, err := detachPolicy(appId, INTERNAL)
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()
		})
		AfterEach(func() {
			stopApiServer()
		})

		Context("Cloud Controller api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
			})
			Context("Create policy", func() {
				It("should error with status code 500", func() {
					By("check public api")
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, PUBLIC)
					checkApiServerStatus(appId, http.StatusInternalServerError, PUBLIC)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, PUBLIC)
				})

				It("should error with status code 500", func() {
					doDetachPolicy(appId, http.StatusInternalServerError, "", PUBLIC)
					checkApiServerStatus(appId, http.StatusInternalServerError, PUBLIC)
				})
			})

		})

		Context("UAA api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
					cf.Endpoints{
						TokenEndpoint:   fakeCCNOAAUAA.URL(),
						DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
					}))
			})
			Context("Create policy", func() {
				It("should error with status code 500", func() {
					By("check public api")
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, PUBLIC)
					checkApiServerStatus(appId, http.StatusInternalServerError, PUBLIC)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, PUBLIC)
				})

				It("should error with status code 500", func() {
					doDetachPolicy(appId, http.StatusInternalServerError, "", PUBLIC)
					checkApiServerStatus(appId, http.StatusInternalServerError, PUBLIC)
				})
			})

		})

		Context("UAA api returns 401", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
					cf.Endpoints{
						TokenEndpoint:   fakeCCNOAAUAA.URL(),
						DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
					}))
				fakeCCNOAAUAA.RouteToHandler("GET", "/userinfo", ghttp.RespondWithJSONEncoded(http.StatusUnauthorized, struct{}{}))
			})
			Context("Create policy", func() {
				It("should error with status code 401", func() {
					By("check public api")
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, PUBLIC)
					checkApiServerStatus(appId, http.StatusUnauthorized, PUBLIC)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, PUBLIC)
				})

				It("should error with status code 401", func() {
					doDetachPolicy(appId, http.StatusUnauthorized, "", PUBLIC)
					checkApiServerStatus(appId, http.StatusUnauthorized, PUBLIC)
				})
			})

		})

		Context("Check permission not passed", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.RouteToHandler("GET", checkUserSpaceRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
					struct {
						TotalResults int `json:"total_results"`
					}{
						0,
					}))
			})
			Context("Create policy", func() {
				It("should error with status code 401", func() {
					By("check public api")
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, PUBLIC)
					checkApiServerStatus(appId, http.StatusUnauthorized, PUBLIC)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, PUBLIC)
				})

				It("should error with status code 401", func() {
					doDetachPolicy(appId, http.StatusUnauthorized, "", PUBLIC)
					checkApiServerStatus(appId, http.StatusUnauthorized, PUBLIC)
				})
			})

		})

		Context("Scheduler is down", func() {

			JustBeforeEach(func() {
				stopScheduler()
			})
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, bindingId, appId, nil)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId)
				startScheduler()
			})

			Context("Create policy", func() {
				Context("internal api", func() {
					It("should not create policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusInternalServerError, INTERNAL)
						checkApiServerStatus(appId, http.StatusNotFound, INTERNAL)
					})
				})

				Context("public api", func() {
					It("should not create policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusInternalServerError, PUBLIC)
						checkApiServerStatus(appId, http.StatusNotFound, PUBLIC)
					})
				})

			})

			Context("Delete policy", func() {
				Context("internal api", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
					})

					It("should delete policy in API server", func() {
						doDetachPolicy(appId, http.StatusInternalServerError, "", INTERNAL)
						checkApiServerStatus(appId, http.StatusNotFound, INTERNAL)
					})
				})

				Context("public api", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
					})

					It("should delete policy in API server", func() {
						doDetachPolicy(appId, http.StatusInternalServerError, "", PUBLIC)
						checkApiServerStatus(appId, http.StatusNotFound, PUBLIC)
					})
				})

			})

		})

		Describe("Create policy", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, bindingId, appId, nil)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId)
			})
			Context("internal api", func() {
				Context("Policies with schedules", func() {
					It("creates a policy and associated schedules", func() {
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
						checkApiServerContent(appId, policyStr, http.StatusOK, INTERNAL)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})).To(BeTrue())
					})

					It("fails with an invalid policy", func() {
						policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

						doAttachPolicy(appId, policyStr, http.StatusBadRequest, INTERNAL)
						checkApiServerStatus(appId, http.StatusNotFound, INTERNAL)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})

				Context("Policies without schedules", func() {
					It("creates only the policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
						checkApiServerContent(appId, policyStr, http.StatusOK, INTERNAL)
						checkSchedulerStatus(appId, http.StatusNotFound)

					})
				})
			})

			Context("public api", func() {
				Context("Policies with schedules", func() {
					It("creates a policy and associated schedules", func() {
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
						checkApiServerContent(appId, policyStr, http.StatusOK, PUBLIC)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})).To(BeTrue())
					})

					It("fails with an invalid policy", func() {
						policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

						doAttachPolicy(appId, policyStr, http.StatusBadRequest, PUBLIC)
						checkApiServerStatus(appId, http.StatusNotFound, PUBLIC)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})

				Context("Policies without schedules", func() {
					It("creates only the policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
						checkApiServerContent(appId, policyStr, http.StatusOK, PUBLIC)
						checkSchedulerStatus(appId, http.StatusNotFound)

					})
				})
			})

		})

		Describe("Update policy", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, bindingId, appId, nil)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId)
			})
			Context("internal api", func() {
				Context("Update policies with schedules", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
					})

					It("updates the policy and schedules", func() {
						//attach another policy with 3 recurring and 1 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithScheduleAnother.json")

						doAttachPolicy(appId, policyStr, http.StatusOK, INTERNAL)
						checkApiServerContent(appId, policyStr, http.StatusOK, INTERNAL)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})).To(BeTrue())
					})
				})
			})

			Context("public api", func() {
				Context("Update policies with schedules", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
					})

					It("updates the policy and schedules", func() {
						//attach another policy with 3 recurring and 1 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithScheduleAnother.json")

						doAttachPolicy(appId, policyStr, http.StatusOK, PUBLIC)
						checkApiServerContent(appId, policyStr, http.StatusOK, PUBLIC)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})).To(BeTrue())
					})
				})
			})

		})

		Describe("Delete Policies", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, bindingId, appId, nil)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId)
			})
			Context("internal api", func() {
				Context("for a non-existing app", func() {
					It("Should return a NOT FOUND (404)", func() {
						doDetachPolicy(appId, http.StatusNotFound, `{"error":"No policy bound with application"}`, INTERNAL)
					})
				})

				Context("with an existing app", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
					})

					It("deletes the policy and schedules", func() {
						doDetachPolicy(appId, http.StatusOK, "", INTERNAL)
						checkApiServerStatus(appId, http.StatusNotFound, INTERNAL)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})
			})

			Context("public api", func() {
				Context("for a non-existing app", func() {
					It("Should return a NOT FOUND (404)", func() {
						doDetachPolicy(appId, http.StatusNotFound, `{"error":"No policy bound with application"}`, PUBLIC)
					})
				})

				Context("with an existing app", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
					})

					It("deletes the policy and schedules", func() {
						doDetachPolicy(appId, http.StatusOK, "", PUBLIC)
						checkApiServerStatus(appId, http.StatusNotFound, PUBLIC)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})
			})

		})
	})

	Describe("When offered as a built-in experience", func() {
		BeforeEach(func() {
			apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], components.Ports[APIPublicServer], false, 200, fakeCCNOAAUAA.URL(), dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ServiceBrokerInternal]), false, tmpDir)
			startApiServer()

			resp, err := detachPolicy(appId, INTERNAL)
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()
		})
		AfterEach(func() {
			stopApiServer()
		})

		Describe("Create policy", func() {
			Context("internal api", func() {
				Context("Policies with schedules", func() {
					It("creates a policy and associated schedules", func() {
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
						checkApiServerContent(appId, policyStr, http.StatusOK, INTERNAL)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})).To(BeTrue())
					})

					It("fails with an invalid policy", func() {
						policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

						doAttachPolicy(appId, policyStr, http.StatusBadRequest, INTERNAL)
						checkApiServerStatus(appId, http.StatusNotFound, INTERNAL)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})

				})

				Context("Policies without schedules", func() {
					It("creates only the policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
						checkApiServerContent(appId, policyStr, http.StatusOK, INTERNAL)
						checkSchedulerStatus(appId, http.StatusNotFound)

					})
				})
			})

			Context("public api", func() {
				Context("Policies with schedules", func() {
					It("creates a policy and associated schedules", func() {
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
						checkApiServerContent(appId, policyStr, http.StatusOK, PUBLIC)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})).To(BeTrue())
					})

					It("fails with an invalid policy", func() {
						policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

						doAttachPolicy(appId, policyStr, http.StatusBadRequest, PUBLIC)
						checkApiServerStatus(appId, http.StatusNotFound, PUBLIC)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})

				})

				Context("Policies without schedules", func() {
					It("creates only the policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
						checkApiServerContent(appId, policyStr, http.StatusOK, PUBLIC)
						checkSchedulerStatus(appId, http.StatusNotFound)

					})
				})
			})

		})

		Describe("Update policy", func() {
			Context("internal api", func() {
				Context("Update policies with schedules", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
					})

					It("updates the policy and schedules", func() {
						//attach another policy with 3 recurring and 1 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithScheduleAnother.json")

						doAttachPolicy(appId, policyStr, http.StatusOK, INTERNAL)
						checkApiServerContent(appId, policyStr, http.StatusOK, INTERNAL)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})).To(BeTrue())
					})
				})
			})

			Context("public api", func() {
				Context("Update policies with schedules", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
					})

					It("updates the policy and schedules", func() {
						//attach another policy with 3 recurring and 1 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithScheduleAnother.json")

						doAttachPolicy(appId, policyStr, http.StatusOK, PUBLIC)
						checkApiServerContent(appId, policyStr, http.StatusOK, PUBLIC)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})).To(BeTrue())
					})
				})
			})

		})

		Describe("Delete Policies", func() {
			Context("internal api", func() {
				Context("for a non-existing app", func() {
					It("Should return a NOT FOUND (404)", func() {
						doDetachPolicy(appId, http.StatusNotFound, `{"error":"No policy bound with application"}`, INTERNAL)
					})
				})

				Context("with an existing app", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
					})

					It("deletes the policy and schedules", func() {
						doDetachPolicy(appId, http.StatusOK, "", INTERNAL)
						checkApiServerStatus(appId, http.StatusNotFound, INTERNAL)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})
			})

			Context("public api", func() {
				Context("for a non-existing app", func() {
					It("Should return a NOT FOUND (404)", func() {
						doDetachPolicy(appId, http.StatusNotFound, `{"error":"No policy bound with application"}`, PUBLIC)
					})
				})

				Context("with an existing app", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
					})

					It("deletes the policy and schedules", func() {
						doDetachPolicy(appId, http.StatusOK, "", PUBLIC)
						checkApiServerStatus(appId, http.StatusNotFound, PUBLIC)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})
			})

		})

	})
})

func doAttachPolicy(appId string, policyStr []byte, statusCode int, apiType APIType) {
	resp, err := attachPolicy(appId, policyStr, apiType)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(statusCode))
	resp.Body.Close()

}
func doDetachPolicy(appId string, statusCode int, msg string, apiType APIType) {
	resp, err := detachPolicy(appId, apiType)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(statusCode))
	if msg != "" {
		respBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(respBody)).To(Equal(msg))
	}
	resp.Body.Close()
}
func checkApiServerStatus(appId string, statusCode int, apiType APIType) {
	By("checking the API Server")
	resp, err := getPolicy(appId, apiType)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(statusCode))
	resp.Body.Close()
}
func checkApiServerContent(appId string, policyStr []byte, statusCode int, apiType APIType) {
	By("checking the API Server")
	var expected map[string]interface{}
	err := json.Unmarshal(policyStr, &expected)
	Expect(err).NotTo(HaveOccurred())
	checkResponseContent(getPolicy, appId, statusCode, expected, apiType)
}
func checkSchedulerStatus(appId string, statusCode int) {
	By("checking the Scheduler")
	resp, err := getSchedules(appId)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(statusCode))
	resp.Body.Close()
}
