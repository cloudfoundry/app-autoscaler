package integration_legacy

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

var _ = Describe("Integration_legacy_Api_Scheduler", func() {
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
			apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], components.Ports[APIPublicServer], false, 200, fakeCCNOAAUAA.URL(), dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ServiceBrokerInternal]), true, defaultHttpClientTimeout, 30, 30, tmpDir)
			startApiServer()

			resp, err := detachPolicy(appId, components.Ports[APIPublicServer], httpClientForPublicApi)
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
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[APIPublicServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusInternalServerError, components.Ports[APIPublicServer], httpClientForPublicApi)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[APIPublicServer], httpClientForPublicApi)
				})

				It("should error with status code 500", func() {
					doDetachPolicy(appId, http.StatusInternalServerError, "", components.Ports[APIPublicServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusInternalServerError, components.Ports[APIPublicServer], httpClientForPublicApi)
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
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[APIPublicServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusInternalServerError, components.Ports[APIPublicServer], httpClientForPublicApi)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[APIPublicServer], httpClientForPublicApi)
				})

				It("should error with status code 500", func() {
					doDetachPolicy(appId, http.StatusInternalServerError, "", components.Ports[APIPublicServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusInternalServerError, components.Ports[APIPublicServer], httpClientForPublicApi)
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
					By("check public api")
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, components.Ports[APIPublicServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusUnauthorized, components.Ports[APIPublicServer], httpClientForPublicApi)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, components.Ports[APIPublicServer], httpClientForPublicApi)
				})

				It("should error with status code 401", func() {
					doDetachPolicy(appId, http.StatusUnauthorized, "", components.Ports[APIPublicServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusUnauthorized, components.Ports[APIPublicServer], httpClientForPublicApi)
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
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, components.Ports[APIPublicServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusUnauthorized, components.Ports[APIPublicServer], httpClientForPublicApi)
				})
			})
			Context("Delete policy", func() {
				BeforeEach(func() {
					policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
					doAttachPolicy(appId, policyStr, http.StatusUnauthorized, components.Ports[APIPublicServer], httpClientForPublicApi)
				})

				It("should error with status code 401", func() {
					doDetachPolicy(appId, http.StatusUnauthorized, "", components.Ports[APIPublicServer], httpClientForPublicApi)
					checkApiServerStatus(appId, http.StatusUnauthorized, components.Ports[APIPublicServer], httpClientForPublicApi)
				})
			})

		})

		Context("Scheduler is down", func() {

			JustBeforeEach(func() {
				stopScheduler()
			})
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, nil, bindingId, appId, nil, components.Ports[ServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClientForPublicApi)
				startScheduler()
			})

			Context("Create policy", func() {
				Context("internal api", func() {
					It("should not create policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[APIServer], httpClient)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIServer], httpClient)
					})
				})

				Context("public api", func() {
					It("should not create policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIPublicServer], httpClientForPublicApi)
					})
				})

			})

			Context("Delete policy", func() {
				Context("internal api", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIServer], httpClient)
					})

					It("should delete policy in API server", func() {
						doDetachPolicy(appId, http.StatusInternalServerError, "", components.Ports[APIServer], httpClient)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIServer], httpClient)
					})
				})

				Context("public api", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIPublicServer], httpClientForPublicApi)
					})

					It("should delete policy in API server", func() {
						doDetachPolicy(appId, http.StatusInternalServerError, "", components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIPublicServer], httpClientForPublicApi)
					})
				})

			})

		})

		Describe("Create policy", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, nil, bindingId, appId, nil, components.Ports[ServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClientForPublicApi)
			})
			Context("internal api", func() {
				Context("Policies with schedules", func() {
					It("creates a policy and associated schedules", func() {
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIServer], httpClient)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIServer], httpClient)
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})
					})

					It("fails with an invalid policy", func() {
						policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

						doAttachPolicy(appId, policyStr, http.StatusBadRequest, components.Ports[APIServer], httpClient)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIServer], httpClient)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})

				Context("Policies without schedules", func() {
					It("creates only the policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIServer], httpClient)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIServer], httpClient)
						checkSchedulerStatus(appId, http.StatusNotFound)

					})
				})
			})

			Context("public api", func() {
				Context("Policies with schedules", func() {
					It("creates a policy and associated schedules", func() {
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIPublicServer], httpClientForPublicApi)
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})
					})

					It("fails with an invalid policy", func() {
						policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

						doAttachPolicy(appId, policyStr, http.StatusBadRequest, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})

				Context("Policies without schedules", func() {
					It("creates only the policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)

					})
				})
			})

		})

		Describe("Update policy", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, nil, bindingId, appId, nil, components.Ports[ServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClientForPublicApi)
			})
			Context("internal api", func() {
				Context("Update policies with schedules", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIServer], httpClient)
					})

					It("updates the policy and schedules", func() {
						//attach another policy with 3 recurring and 1 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithScheduleAnother.json"))

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[APIServer], httpClient)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIServer], httpClient)
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})
					})
				})
			})

			Context("public api", func() {
				Context("Update policies with schedules", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIPublicServer], httpClientForPublicApi)
					})

					It("updates the policy and schedules", func() {
						//attach another policy with 3 recurring and 1 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithScheduleAnother.json"))

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIPublicServer], httpClientForPublicApi)
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})
					})
				})
			})

		})

		Describe("Delete Policies", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, nil, bindingId, appId, nil, components.Ports[ServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeprovision(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClientForPublicApi)
			})
			Context("internal api", func() {
				Context("for a non-existing app", func() {
					It("Should return a NOT FOUND (404)", func() {
						doDetachPolicy(appId, http.StatusNotFound, `{"error":"No policy bound with application"}`, components.Ports[APIServer], httpClient)
					})
				})

				Context("with an existing app", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIServer], httpClient)
					})

					It("deletes the policy and schedules", func() {
						doDetachPolicy(appId, http.StatusOK, "", components.Ports[APIServer], httpClient)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIServer], httpClient)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})
			})

			Context("public api", func() {
				Context("for a non-existing app", func() {
					It("Should return a NOT FOUND (404)", func() {
						doDetachPolicy(appId, http.StatusNotFound, `{"error":"No policy bound with application"}`, components.Ports[APIPublicServer], httpClientForPublicApi)
					})
				})

				Context("with an existing app", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIPublicServer], httpClientForPublicApi)
					})

					It("deletes the policy and schedules", func() {
						doDetachPolicy(appId, http.StatusOK, "", components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})
			})

		})
	})

	Describe("When offered as a built-in experience", func() {
		BeforeEach(func() {
			apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], components.Ports[APIPublicServer], false, 200, fakeCCNOAAUAA.URL(), dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ServiceBrokerInternal]), false, defaultHttpClientTimeout, 30, 30, tmpDir)
			startApiServer()

			resp, err := detachPolicy(appId, components.Ports[APIServer], httpClient)
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
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIServer], httpClient)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIServer], httpClient)
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})
					})

					It("fails with an invalid policy", func() {
						policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

						doAttachPolicy(appId, policyStr, http.StatusBadRequest, components.Ports[APIServer], httpClient)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIServer], httpClient)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})

				})

				Context("Policies without schedules", func() {
					It("creates only the policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIServer], httpClient)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIServer], httpClient)
						checkSchedulerStatus(appId, http.StatusNotFound)

					})
				})
			})

			Context("public api", func() {
				Context("Policies with schedules", func() {
					It("creates a policy and associated schedules", func() {
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIPublicServer], httpClientForPublicApi)
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})
					})

					It("fails with an invalid policy", func() {
						policyStr = readPolicyFromFile("fakeInvalidPolicy.json")

						doAttachPolicy(appId, policyStr, http.StatusBadRequest, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})

				})

				Context("Policies without schedules", func() {
					It("creates only the policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIPublicServer], httpClientForPublicApi)
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
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))
						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIServer], httpClient)
					})

					It("updates the policy and schedules", func() {
						//attach another policy with 3 recurring and 1 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithScheduleAnother.json"))

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[APIServer], httpClient)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIServer], httpClient)
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})
					})
				})
			})

			Context("public api", func() {
				Context("Update policies with schedules", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))
						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIPublicServer], httpClientForPublicApi)
					})

					It("updates the policy and schedules", func() {
						//attach another policy with 3 recurring and 1 specific_date schedules
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithScheduleAnother.json"))

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[APIPublicServer], httpClientForPublicApi)
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})
					})
				})
			})

		})

		Describe("Delete Policies", func() {
			Context("internal api", func() {
				Context("for a non-existing app", func() {
					It("Should return a NOT FOUND (404)", func() {
						doDetachPolicy(appId, http.StatusNotFound, `{"error":"No policy bound with application"}`, components.Ports[APIServer], httpClient)
					})
				})

				Context("with an existing app", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")

						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIServer], httpClient)
					})

					It("deletes the policy and schedules", func() {
						doDetachPolicy(appId, http.StatusOK, "", components.Ports[APIServer], httpClient)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIServer], httpClient)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})
			})

			Context("public api", func() {
				Context("for a non-existing app", func() {
					It("Should return a NOT FOUND (404)", func() {
						doDetachPolicy(appId, http.StatusNotFound, `{"error":"No policy bound with application"}`, components.Ports[APIPublicServer], httpClientForPublicApi)
					})
				})

				Context("with an existing app", func() {
					BeforeEach(func() {
						//attach a policy first with 4 recurring and 2 specific_date schedules
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusCreated, components.Ports[APIPublicServer], httpClientForPublicApi)
					})

					It("deletes the policy and schedules", func() {
						doDetachPolicy(appId, http.StatusOK, "", components.Ports[APIPublicServer], httpClientForPublicApi)
						checkApiServerStatus(appId, http.StatusNotFound, components.Ports[APIPublicServer], httpClientForPublicApi)
						checkSchedulerStatus(appId, http.StatusNotFound)
					})
				})
			})

		})

	})
})
