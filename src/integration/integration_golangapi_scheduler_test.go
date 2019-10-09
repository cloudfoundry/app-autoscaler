package integration

import (
	"autoscaler/cf"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Integration_GolangApi_Scheduler", func() {
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

		schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir, defaultHttpClientTimeout)
		startScheduler()

		serviceInstanceId = getRandomId()
		orgId = getRandomId()
		spaceId = getRandomId()
		bindingId = getRandomId()
		appId = getRandomId()
		brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
	})

	AfterEach(func() {
		stopScheduler()
	})

	Describe("When offered as a service", func() {

		BeforeEach(func() {
			golangApiServerConfPath = components.PrepareGolangApiServerConfig(dbUrl, components.Ports[GolangAPIServer], components.Ports[GolangServiceBroker],
				fakeCCNOAAUAA.URL(), false, 200, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]), "https://127.0.0.1:8888",
				false, defaultHttpClientTimeout, tmpDir)
			startGolangApiServer()

			resp, err := detachPolicy(appId, components.Ports[GolangAPIServer], httpClientForPublicApi)
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()
		})
		AfterEach(func() {
			stopGolangApiServer()
		})

		Context("Cloud Controller api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
			})
			Context("Create policy", func() {
				It("should error with status code 500", func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[GolangAPIServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusInternalServerError, components.Ports[GolangAPIServer], httpClientForPublicApi)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[GolangAPIServer], httpClientForPublicApi)
				})

				It("should error with status code 500", func() {
					doDetachPolicy(appId, http.StatusInternalServerError, "", components.Ports[GolangAPIServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusInternalServerError, components.Ports[GolangAPIServer], httpClientForPublicApi)
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
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[GolangAPIServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusInternalServerError, components.Ports[GolangAPIServer], httpClientForPublicApi)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[GolangAPIServer], httpClientForPublicApi)
				})

				It("should error with status code 500", func() {
					doDetachPolicy(appId, http.StatusInternalServerError, "", components.Ports[GolangAPIServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusInternalServerError, components.Ports[GolangAPIServer], httpClientForPublicApi)
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
				fakeCCNOAAUAA.RouteToHandler("POST", "/check_token", ghttp.RespondWithJSONEncoded(http.StatusOK,
					struct {
						Scope []string `json:"scope"`
					}{
						[]string{"cloud_controller.read", "cloud_controller.write", "password.write", "openid", "network.admin", "network.write", "uaa.user"},
					}))
				fakeCCNOAAUAA.RouteToHandler("GET", "/userinfo", ghttp.RespondWithJSONEncoded(http.StatusUnauthorized, struct{}{}))
			})
			Context("Create policy", func() {
				It("should error with status code 401", func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, components.Ports[GolangAPIServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusUnauthorized, components.Ports[GolangAPIServer], httpClientForPublicApi)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, components.Ports[GolangAPIServer], httpClientForPublicApi)
				})

				It("should error with status code 401", func() {
					doDetachPolicy(appId, http.StatusUnauthorized, "", components.Ports[GolangAPIServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusUnauthorized, components.Ports[GolangAPIServer], httpClientForPublicApi)
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
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, components.Ports[GolangAPIServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusUnauthorized, components.Ports[GolangAPIServer], httpClientForPublicApi)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, components.Ports[GolangAPIServer], httpClientForPublicApi)
				})

				It("should error with status code 401", func() {
					doDetachPolicy(appId, http.StatusUnauthorized, "", components.Ports[GolangAPIServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusUnauthorized, components.Ports[GolangAPIServer], httpClientForPublicApi)
				})
			})

		})

		Context("Scheduler is down", func() {

			JustBeforeEach(func() {
				stopScheduler()
			})
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, bindingId, appId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})

			Context("Create policy", func() {
				Context("public api", func() {
					It("should create policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
					})
				})

			})

			Context("Delete policy", func() {
				Context("public api", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
					})

					It("should delete policy in API server", func() {
						doDetachPolicy(appId, http.StatusInternalServerError, "", components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[GolangAPIServer], httpClientForPublicApi)
					})
				})

			})

		})

		Describe("Create policy", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, bindingId, appId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			Context("public api", func() {
				Context("Policies with schedules", func() {
					It("creates a policy and associated schedules", func() {
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})).To(BeTrue())
					})

					It("fails with an invalid policy", func() {
						policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

						doAttachPolicy(appId, policyStr, http.StatusBadRequest, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})

				Context("Policies without schedules", func() {
					It("creates only the policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)

					})
				})
			})

		})

		Describe("Update policy", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, bindingId, appId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			Context("public api", func() {
				Context("Update policies with schedules", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
					})

					It("updates the policy and schedules", func() {
						//attach another policy with 3 recurring and 1 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithScheduleAnother.json"))

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})).To(BeTrue())
					})
				})
			})

		})

		Describe("Delete Policies", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, bindingId, appId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})

			Context("public api", func() {
				Context("for a non-existing app", func() {
					It("Should return ok", func() {
						doDetachPolicy(appId, http.StatusOK, `{}`, components.Ports[GolangAPIServer], httpClientForPublicApi)
					})
				})

				Context("with an existing app", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
					})

					It("deletes the policy and schedules", func() {
						doDetachPolicy(appId, http.StatusOK, "", components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})
			})

		})
	})

	Describe("When offered as a built-in experience", func() {
		BeforeEach(func() {
			golangApiServerConfPath = components.PrepareGolangApiServerConfig(dbUrl, components.Ports[GolangAPIServer], components.Ports[GolangServiceBroker],
				fakeCCNOAAUAA.URL(), false, 200, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]), "https://127.0.0.1:8888",
				true, defaultHttpClientTimeout, tmpDir)
			startGolangApiServer()

			resp, err := detachPolicy(appId, components.Ports[GolangAPIServer], httpClientForPublicApi)
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()
		})
		AfterEach(func() {
			stopGolangApiServer()
		})

		Describe("Create policy", func() {
			Context("public api", func() {
				Context("Policies with schedules", func() {
					It("creates a policy and associated schedules", func() {
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})).To(BeTrue())
					})

					It("fails with an invalid policy", func() {
						policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

						doAttachPolicy(appId, policyStr, http.StatusBadRequest, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})

				})

				Context("Policies without schedules", func() {
					It("creates only the policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)

					})
				})
			})

		})

		Describe("Update policy", func() {
			Context("public api", func() {
				Context("Update policies with schedules", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))
						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
					})

					It("updates the policy and schedules", func() {
						//attach another policy with 3 recurring and 1 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithScheduleAnother.json"))

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						Expect(checkSchedule(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})).To(BeTrue())
					})
				})
			})

		})

		Describe("Delete Policies", func() {
			Context("public api", func() {
				Context("for a non-existing app", func() {
					It("Should return OK", func() {
						doDetachPolicy(appId, http.StatusOK, `{}`, components.Ports[GolangAPIServer], httpClientForPublicApi)
					})
				})

				Context("with an existing app", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
					})

					It("deletes the policy and schedules", func() {
						doDetachPolicy(appId, http.StatusOK, "", components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})
			})

		})

	})
})
