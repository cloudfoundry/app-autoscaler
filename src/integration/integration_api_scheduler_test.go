package integration_test

import (
	"autoscaler/cf"
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "integration"
	"io/ioutil"
	"net/http"
	"strings"
)

var _ = Describe("Integration_Api_Scheduler", func() {
	var (
		appId             string
		policyStr         []byte
		initInstanceCount int = 2
	)

	BeforeEach(func() {
		startFakeCCNOAAUAA(initInstanceCount)
		initializeHttpClient("api.crt", "api.key", "autoscaler-ca.crt", apiSchedulerHttpRequestTimeout)
		initializeHttpClientForPublicApi("api_public.crt", "api_public.key", "autoscaler-ca.crt", apiMetricsCollectorHttpRequestTimeout)

		schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir, strings.Split(consulRunner.Address(), ":")[1])
		schedulerProcess = startScheduler()

		apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], components.Ports[APIPublicServer], fakeCCNOAAUAA.URL(), dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), tmpDir)
		startApiServer()
		appId = getRandomId()
		resp, err := detachPolicy(appId, INTERNAL)
		Expect(err).NotTo(HaveOccurred())
		resp.Body.Close()
	})

	AfterEach(func() {
		stopApiServer()
		stopScheduler(schedulerProcess)
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
					AuthEndpoint:    fakeCCNOAAUAA.URL(),
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
					AuthEndpoint:    fakeCCNOAAUAA.URL(),
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
			stopScheduler(schedulerProcess)
		})

		AfterEach(func() {
			schedulerProcess = startScheduler()
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
		Context("internal api", func() {
			Context("Policies with schedules", func() {
				It("creates a policy and associated schedules", func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

					doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
					checkApiServerContent(appId, policyStr, http.StatusOK, INTERNAL)
					checkSchedulerContent(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2}, INTERNAL)
				})

				It("fails with an invalid policy", func() {
					policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

					doAttachPolicy(appId, policyStr, http.StatusBadRequest, INTERNAL)
					checkApiServerStatus(appId, http.StatusNotFound, INTERNAL)
					checkSchedulerStatus(appId, http.StatusNotFound, INTERNAL)
				})

			})

			Context("Policies without schedules", func() {
				It("creates only the policy", func() {
					policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

					doAttachPolicy(appId, policyStr, http.StatusCreated, INTERNAL)
					checkApiServerContent(appId, policyStr, http.StatusOK, INTERNAL)
					checkSchedulerStatus(appId, http.StatusNotFound, INTERNAL)

				})
			})
		})

		Context("public api", func() {
			Context("Policies with schedules", func() {
				It("creates a policy and associated schedules", func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

					doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
					checkApiServerContent(appId, policyStr, http.StatusOK, PUBLIC)
					checkSchedulerContent(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2}, PUBLIC)
				})

				It("fails with an invalid policy", func() {
					policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

					doAttachPolicy(appId, policyStr, http.StatusBadRequest, PUBLIC)
					checkApiServerStatus(appId, http.StatusNotFound, PUBLIC)
					checkSchedulerStatus(appId, http.StatusNotFound, PUBLIC)
				})

			})

			Context("Policies without schedules", func() {
				It("creates only the policy", func() {
					policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

					doAttachPolicy(appId, policyStr, http.StatusCreated, PUBLIC)
					checkApiServerContent(appId, policyStr, http.StatusOK, PUBLIC)
					checkSchedulerStatus(appId, http.StatusNotFound, PUBLIC)

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
					checkSchedulerContent(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1}, INTERNAL)
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
					checkSchedulerContent(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1}, PUBLIC)
				})
			})
		})

	})

	Describe("Delete Policies", func() {
		Context("internal api", func() {
			Context("for a non-existing app", func() {
				It("Should return a NOT FOUND (404)", func() {
					doDetachPolicy(appId, http.StatusNotFound, `{"success":false,"error":{"message":"No policy bound with application","statusCode":404},"result":null}`, INTERNAL)
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
					checkSchedulerStatus(appId, http.StatusNotFound, INTERNAL)
				})
			})
		})

		Context("public api", func() {
			Context("for a non-existing app", func() {
				It("Should return a NOT FOUND (404)", func() {
					doDetachPolicy(appId, http.StatusNotFound, `{"success":false,"error":{"message":"No policy bound with application","statusCode":404},"result":null}`, PUBLIC)
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
					checkSchedulerStatus(appId, http.StatusNotFound, PUBLIC)
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
func checkSchedulerStatus(appId string, statusCode int, apiType APIType) {
	By("checking the Scheduler")
	resp, err := getSchedules(appId, apiType)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(statusCode))
	resp.Body.Close()
}
func checkSchedulerContent(appId string, statusCode int, expectedScheduleNumMap map[string]int, apiType APIType) {
	By("checking the Scheduler")
	checkSchedule(getSchedules, appId, statusCode, expectedScheduleNumMap, apiType)
}
