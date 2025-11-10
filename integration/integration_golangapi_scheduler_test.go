package integration_test

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		tmpDir            string

		err       error
		brokerUrl *url.URL
	)

	BeforeEach(func() {
		tmpDir, err = os.MkdirTemp("", "autoscaler")
		Expect(err).NotTo(HaveOccurred())

		startFakeCCNOAAUAA(initInstanceCount)
		httpClient = testhelpers.NewApiClient()
		httpClientForPublicApi = testhelpers.NewPublicApiClient()

		schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir, defaultHttpClientTimeout)
		startScheduler()

		serviceInstanceId = getRandomIdRef("serviceInstId")
		orgId = getRandomIdRef("orgId")
		spaceId = getRandomIdRef("spaceId")
		bindingId = getRandomIdRef("bindingId")
		binding2Id = getRandomIdRef("binding2Id")
		binding3Id = getRandomIdRef("binding3Id")
		appId = uuid.NewString()
		app2Id = uuid.NewString()
		app3Id = uuid.NewString()
		brokerAuth = base64.StdEncoding.EncodeToString([]byte("broker_username:broker_password"))

		brokerUrl, err = url.Parse(fmt.Sprintf("https://127.0.0.1:%d", components.Ports[GolangServiceBroker]))
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
		stopScheduler()
	})

	When("offered as a service", func() {
		BeforeEach(func() {
			golangApiServerConfPath := components.PrepareGolangApiServerConfig(
				dbUrl,
				fakeCCNOAAUAA.URL(),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
				fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]),
				tmpDir)

			startGolangApiServer(golangApiServerConfPath)

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
				fakeCCNOAAUAA.Add().Info(fakeCCNOAAUAA.URL())
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
				fakeCCNOAAUAA.Add().Info(fakeCCNOAAUAA.URL()).Introspect(testUserScope).UserInfo(http.StatusUnauthorized, "ERR")
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
				fakeCCNOAAUAA.Add().Roles(http.StatusOK)
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
				provisionAndBind(brokerUrl, serviceInstanceId, orgId, spaceId, bindingId, appId, httpClientForPublicApi)
			})

			Context("Create policy", func() {
				Context("public api", func() {
					It("should create policy", func() {
						policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
						doAttachPolicy(appId, policyStr, http.StatusInternalServerError, components.Ports[GolangAPIServer], httpClientForPublicApi)
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
				provisionAndBind(brokerUrl, serviceInstanceId, orgId, spaceId, bindingId, appId, httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeProvision(brokerUrl, bindingId, appId, serviceInstanceId, httpClientForPublicApi)
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
				provisionAndBind(brokerUrl, serviceInstanceId, orgId, spaceId, bindingId, appId, httpClientForPublicApi)
			})

			AfterEach(func() {
				unbindAndDeProvision(brokerUrl, bindingId, appId, serviceInstanceId, httpClientForPublicApi)
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
				resp, err = provisionServiceInstance(brokerUrl, serviceInstanceId, orgId, spaceId, defaultPolicy, httpClientForPublicApi)
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
					resp, err = unbindService(brokerUrl, binding2Id, app2Id, serviceInstanceId, httpClientForPublicApi)
					Expect(err).NotTo(HaveOccurred(), "Error: %s", err)
					defer func() { _ = resp.Body.Close() }()
					Expect(resp.StatusCode).To(Equal(http.StatusOK), ResponseMessage(resp))

					resp, err = unbindService(brokerUrl, binding3Id, app3Id, serviceInstanceId, httpClientForPublicApi)
					Expect(err).NotTo(HaveOccurred(), "Error: %s", err)
					defer func() { _ = resp.Body.Close() }()
					Expect(resp.StatusCode).To(Equal(http.StatusOK), ResponseMessage(resp))
					unbindAndDeProvision(brokerUrl, bindingId, appId, serviceInstanceId, httpClientForPublicApi)
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
							It("is reflected in the apps policies and schedules", func() {
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
				provisionAndBind(brokerUrl, serviceInstanceId, orgId, spaceId, bindingId, appId, httpClientForPublicApi)
			})

			AfterEach(func() {
				unbindAndDeProvision(brokerUrl, bindingId, appId, serviceInstanceId, httpClientForPublicApi)
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
				provisionAndBind(brokerUrl, serviceInstanceId, orgId, spaceId, bindingId, appId, httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeProvision(brokerUrl, bindingId, appId, serviceInstanceId, httpClientForPublicApi)
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

	When("connection to scheduler through the gorouter", func() {
		BeforeEach(func() {
			golangConf := DefaultGolangAPITestConfig()

			golangConf.Scheduler.SchedulerURL = fmt.Sprintf("https://127.0.0.1:%d", components.Ports[GoRouterProxy])
			golangConf.EventGenerator.EventGeneratorUrl = fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator])
			golangConf.ScalingEngine.ScalingEngineUrl = fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine])
			golangConf.CF.API = fakeCCNOAAUAA.URL()

			certTmpDir := os.TempDir()

			cfInstanceCertFileContent, cfInstanceKeyContent, err := testhelpers.GenerateClientCertWithCA("some-org-guid", "some-space-guid", "../test-certs/gorouter-ca.crt", "../test-certs/gorouter-ca.key")
			Expect(err).ToNot(HaveOccurred())

			certFile, err := configutil.MaterializeContentInFile(certTmpDir, "cf.crt", string(cfInstanceCertFileContent))
			Expect(err).NotTo(HaveOccurred())
			os.Setenv("CF_INSTANCE_CERT", certFile)
			os.Setenv("CF_INSTANCE_CA_CERT", certFile)

			keyFile, err := configutil.MaterializeContentInFile(certTmpDir, "cf.key", string(cfInstanceKeyContent))
			Expect(err).NotTo(HaveOccurred())
			os.Setenv("CF_INSTANCE_KEY", keyFile)

			os.Setenv("VCAP_APPLICATION", `{}`)

			golangConfJson, err := configutil.ToJSON(golangConf)

			Expect(err).NotTo(HaveOccurred())

			vcapServicesJson := testhelpers.GetVcapServices("apiserver-config", golangConfJson)
			os.Setenv("VCAP_SERVICES", vcapServicesJson)

			os.Setenv("PORT", strconv.Itoa(components.Ports[GolangAPICFServer]))

			startGolangApiCFServer()
			startGoRouterProxyTo(components.Ports[SchedulerCFServer])
		})

		AfterEach(func() {
			stopGolangApiServer()
			stopGoRouterProxy()
			os.Unsetenv("VCAP_APPLICATION")
			os.Unsetenv("CF_INSTANCE_KEY")
			os.Unsetenv("CF_INSTANCE_CA_KEY")
			os.Unsetenv("CF_INSTANCE_CERT")
		})

		Describe("Create policy", func() {
			BeforeEach(func() {
				provisionAndBind(brokerUrl, serviceInstanceId, orgId, spaceId, bindingId, appId, httpClientForPublicApi)
			})
			AfterEach(func() {
				unbindAndDeProvision(brokerUrl, bindingId, appId, serviceInstanceId, httpClientForPublicApi)
			})

			When("Policies with schedules", func() {
				It("creates a policy and associated schedules", func() {
					policyStr = setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json"))

					// change to golangCFAPIServer ?
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
		})
	})
})

func ResponseMessage(resp *http.Response) string {
	body, err := io.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred(), "Error: %s", err)
	return fmt.Sprintf("Error retrieved status %d - '%s'", resp.StatusCode, body)
}
