package integration_test

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Integration_GolangApi_Scheduler", func() {
	var (
		appId             string
		app2Id            string
		app3Id            string
		policyStr         []byte
		initInstanceCount = 2
		serviceInstanceId string
		bindingId         string
		binding2Id        string
		binding3Id        string
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
		binding2Id = getRandomId()
		binding3Id = getRandomId()
		appId = getRandomId()
		app2Id = getRandomId()
		app3Id = getRandomId()
		brokerAuth = base64.StdEncoding.EncodeToString([]byte("broker_username:broker_password"))
	})

	AfterEach(func() {
		stopScheduler()
	})

	Describe("When offered as a service", func() {

		BeforeEach(func() {
			golangApiServerConfPath = components.PrepareGolangApiServerConfig(
				dbUrl,
				components.Ports[GolangAPIServer],
				components.Ports[GolangServiceBroker],
				fakeCCNOAAUAA.URL(),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]),
				"https://127.0.0.1:8888",
				false,
				tmpDir)
			startGolangApiServer()

			resp, err := detachPolicy(appId, components.Ports[GolangAPIServer], httpClientForPublicApi)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

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
				fakeCCNOAAUAA.RouteToHandler("GET", rolesRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
					struct {
						Pagination struct {
							Total int `json:"total_results"`
						} `json:"pagination"`
					}{}))
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
				provisionAndBind(serviceInstanceId, orgId, spaceId, nil, bindingId, appId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
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
				provisionAndBind(serviceInstanceId, orgId, spaceId, nil, bindingId, appId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeProvision(bindingId, appId, serviceInstanceId, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			Context("public api", func() {
				Context("Policies with schedules", func() {
					It("creates a policy and associated schedules", func() {
						policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

						doAttachPolicy(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkApiServerContent(appId, policyStr, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})
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

		Describe("creating and binding a service instance without a default policy", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, nil, bindingId, appId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeProvision(bindingId, appId, serviceInstanceId, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			Context("and then setting a default policy", func() {
				var (
					newDefaultPolicy []byte
					err              error
					resp             *http.Response
				)

				BeforeEach(func() {
					newDefaultPolicy = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))
					resp, err = updateServiceInstance(serviceInstanceId, newDefaultPolicy, components.Ports[GolangServiceBroker], httpClientForPublicApi)
					Expect(err).NotTo(HaveOccurred())
					defer func() { _ = resp.Body.Close() }()
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

				})

				It("creates a policy and associated schedules", func() {
					checkApiServerContent(appId, newDefaultPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
					assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})
				})
			})
		})

		Describe("creating a service instance with a default policy", func() {
			var (
				defaultPolicy []byte
				err           error
				resp          *http.Response
			)
			JustBeforeEach(func() {
				resp, err = provisionServiceInstance(serviceInstanceId, orgId, spaceId, defaultPolicy, components.Ports[GolangServiceBroker], httpClientForPublicApi)
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() { _ = resp.Body.Close() })

			Context("with an invalid default policy", func() {
				BeforeEach(func() {
					defaultPolicy = readPolicyFromFile("fakeInvalidPolicy.json")
				})

				It("fails", func() {
					Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				})
			})

			Context("when binding to it", func() {
				var secondPolicy []byte
				BeforeEach(func() {
					defaultPolicy = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))
					secondPolicy = setPolicyRecurringDate(readPolicyFromFile("fakeMinimalScalingRulePolicy.json"))
				})

				JustBeforeEach(func() {
					resp, err = bindService(bindingId, appId, serviceInstanceId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
					Expect(err).NotTo(HaveOccurred(), "Error: %s", err)
					Expect(resp.StatusCode).To(Equal(http.StatusCreated), ResponseMessage(resp))
					defer func() { _ = resp.Body.Close() }()

					resp, err = bindService(binding2Id, app2Id, serviceInstanceId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
					defer func() { _ = resp.Body.Close() }()
					Expect(err).NotTo(HaveOccurred(), "Error: %s", err)
					Expect(resp.StatusCode).To(Equal(http.StatusCreated), ResponseMessage(resp))

					// app with explicit policy
					resp, err = bindService(binding3Id, app3Id, serviceInstanceId, secondPolicy, components.Ports[GolangServiceBroker], httpClientForPublicApi)
					defer func() { _ = resp.Body.Close() }()
					Expect(err).NotTo(HaveOccurred(), "Error: %s", err)
					Expect(resp.StatusCode).To(Equal(http.StatusCreated), ResponseMessage(resp))
				})

				AfterEach(func() {
					resp, err = unbindService(binding2Id, app2Id, serviceInstanceId, components.Ports[GolangServiceBroker], httpClientForPublicApi)
					Expect(err).NotTo(HaveOccurred(), "Error: %s", err)
					defer func() { _ = resp.Body.Close() }()
					Expect(resp.StatusCode).To(Equal(http.StatusOK), ResponseMessage(resp))

					resp, err = unbindService(binding3Id, app3Id, serviceInstanceId, components.Ports[GolangServiceBroker], httpClientForPublicApi)
					Expect(err).NotTo(HaveOccurred(), "Error: %s", err)
					defer func() { _ = resp.Body.Close() }()
					Expect(resp.StatusCode).To(Equal(http.StatusOK), ResponseMessage(resp))
					unbindAndDeProvision(bindingId, appId, serviceInstanceId, components.Ports[GolangServiceBroker], httpClientForPublicApi)
				})

				It("creates a policy and associated schedules", func() {
					By("setting the default policy on apps without an explicit one")
					checkApiServerContent(appId, defaultPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
					assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})

					checkApiServerContent(app2Id, defaultPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
					assertScheduleContents(app2Id, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})

					By("setting the provided explicit policy")
					checkApiServerContent(app3Id, secondPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
					assertScheduleContents(app3Id, http.StatusOK, map[string]int{"recurring_schedule": 2, "specific_date": 1})
				})

				Context("and then updating some apps' policies explicitly", func() {
					JustBeforeEach(func() {
						doAttachPolicy(app2Id, secondPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						doDetachPolicy(app3Id, http.StatusOK, "", components.Ports[GolangAPIServer], httpClientForPublicApi)
					})

					It("changes the apps' policies and schedules", func() {
						By("leaving the unchanged app alone")
						checkApiServerContent(appId, defaultPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})

						By("setting the new explicit policy")
						checkApiServerContent(app2Id, secondPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						assertScheduleContents(app2Id, http.StatusOK, map[string]int{"recurring_schedule": 2, "specific_date": 1})

						By("reverting to the default policy as there is no longer an explicit one")
						checkApiServerContent(app3Id, defaultPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
						checkSchedulerStatus(app3Id, http.StatusOK)
					})

					var newDefaultPolicy []byte
					Context("and then changing the default policy", func() {
						JustBeforeEach(func() {
							resp, err = updateServiceInstance(serviceInstanceId, newDefaultPolicy, components.Ports[GolangServiceBroker], httpClientForPublicApi)
							Expect(err).NotTo(HaveOccurred())
							Expect(resp.StatusCode).To(Equal(http.StatusOK))
						})
						Context("to a new one", func() {
							BeforeEach(func() {
								newDefaultPolicy = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithScheduleAnother.json"))
							})
							It("is reflected in the apps' policies and schedules", func() {
								By("ensuring that the app with the default policy gets the new default app policy")
								checkApiServerContent(appId, newDefaultPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
								assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})

								By("ensuring that the app with the specifically set policy is not modified")
								checkApiServerContent(app2Id, secondPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
								assertScheduleContents(app2Id, http.StatusOK, map[string]int{"recurring_schedule": 2, "specific_date": 1})

								By("ensuring that the app without a policy gets the new default app policy")
								checkApiServerContent(app3Id, newDefaultPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
								assertScheduleContents(app3Id, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})
							})
						})
						Context("by removing it", func() {
							BeforeEach(func() {
								newDefaultPolicy = []byte("{\t\n}\r\n")
							})
							It("is reflected in the apps' policies and schedules", func() {
								checkApiServerStatus(appId, http.StatusNotFound, components.Ports[GolangAPIServer], httpClientForPublicApi)
								checkSchedulerStatus(appId, http.StatusNotFound)
								checkApiServerContent(app2Id, secondPolicy, http.StatusOK, components.Ports[GolangAPIServer], httpClientForPublicApi)
								assertScheduleContents(app2Id, http.StatusOK, map[string]int{"recurring_schedule": 2, "specific_date": 1})
							})
						})
					})

				})
			})
		})

		Describe("Update policy", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, nil, bindingId, appId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeProvision(bindingId, appId, serviceInstanceId, components.Ports[GolangServiceBroker], httpClientForPublicApi)
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
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})
					})
				})
			})

		})

		Describe("Delete Policies", func() {
			BeforeEach(func() {
				provisionAndBind(serviceInstanceId, orgId, spaceId, nil, bindingId, appId, nil, components.Ports[GolangServiceBroker], httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeProvision(bindingId, appId, serviceInstanceId, components.Ports[GolangServiceBroker], httpClientForPublicApi)
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
			golangApiServerConfPath = components.PrepareGolangApiServerConfig(
				dbUrl,
				components.Ports[GolangAPIServer],
				components.Ports[GolangServiceBroker],
				fakeCCNOAAUAA.URL(),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]),
				"https://127.0.0.1:8888",
				true,
				tmpDir)
			startGolangApiServer()

			resp, err := detachPolicy(appId, components.Ports[GolangAPIServer], httpClientForPublicApi)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()
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
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})
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
						assertScheduleContents(appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})
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

func ResponseMessage(resp *http.Response) string {
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred(), "Error: %s", err)
	return fmt.Sprintf("Error retrieved status %d - '%s'", resp.StatusCode, body)
}
