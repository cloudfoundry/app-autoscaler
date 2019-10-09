package integration_legacy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Integration_legacy_Broker_Api", func() {

	var (
		regPath = regexp.MustCompile(`^/v1/apps/.*/schedules`)

		serviceInstanceId            string
		bindingId                    string
		orgId                        string
		spaceId                      string
		appId                        string
		schedulePolicyJson           []byte
		invalidSchemaPolicyJson      []byte
		invalidDataPolicyJson        []byte
		minimalScalingRulePolicyJson []byte
	)

	BeforeEach(func() {
		initializeHttpClient("servicebroker.crt", "servicebroker.key", "autoscaler-ca.crt", brokerApiHttpRequestTimeout)
		fakeScheduler = ghttp.NewServer()
		apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], components.Ports[APIPublicServer], false, 200, "", dbUrl, fakeScheduler.URL(), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ServiceBrokerInternal]), true, defaultHttpClientTimeout, 30, 30, tmpDir)
		serviceBrokerConfPath = components.PrepareServiceBrokerConfig(components.Ports[ServiceBroker], components.Ports[ServiceBrokerInternal], brokerUserName, brokerPassword, false, dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[APIServer]), brokerApiHttpRequestTimeout, tmpDir)

		startApiServer()
		startServiceBroker()
		brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
		serviceInstanceId = getRandomId()
		orgId = getRandomId()
		spaceId = getRandomId()
		bindingId = getRandomId()
		appId = getRandomId()
		//add a service instance
		resp, err := provisionServiceInstance(serviceInstanceId, orgId, spaceId, nil, components.Ports[ServiceBroker], httpClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusCreated))
		resp.Body.Close()

		schedulePolicyJson = readPolicyFromFile("fakePolicyWithSchedule.json")
		invalidSchemaPolicyJson = readPolicyFromFile("fakeInvalidPolicy.json")
		invalidDataPolicyJson = readPolicyFromFile("fakeInvalidDataPolicy.json")
		minimalScalingRulePolicyJson = readPolicyFromFile("fakeMinimalScalingRulePolicy.json")
	})

	AfterEach(func() {
		//clean the service instance added in before each
		resp, err := deprovisionServiceInstance(serviceInstanceId, components.Ports[ServiceBroker], httpClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		resp.Body.Close()
		fakeScheduler.Close()
		stopAll()
	})

	Describe("Bind Service", func() {
		Context("Policy with schedules", func() {
			BeforeEach(func() {
				fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
				fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})

			AfterEach(func() {
				//clear the binding
				resp, err := unbindService(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("creates a binding", func() {
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()
				Consistently(fakeScheduler.ReceivedRequests).Should(HaveLen(1))

				By("checking the API Server")
				var expected map[string]interface{}
				err = json.Unmarshal(schedulePolicyJson, &expected)
				Expect(err).NotTo(HaveOccurred())
				// If custom metrics not enabled, credentials should not be created
				By("checking the credential table content")
				Expect(getCredentialsCount(appId)).To(Equal(0))

				checkResponseContent(getPolicy, appId, http.StatusOK, expected, components.Ports[APIServer], httpClient)
			})
		})

		Context("Policy with minimal Scaling Rules", func() {
			BeforeEach(func() {
				fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
				fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})

			AfterEach(func() {
				//clear the binding
				resp, err := unbindService(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("creates a binding", func() {
				schedulerRequestCount := len(fakeScheduler.ReceivedRequests())
				resp, err := bindService(bindingId, appId, serviceInstanceId, minimalScalingRulePolicyJson, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()
				Consistently(fakeScheduler.ReceivedRequests).Should(HaveLen(schedulerRequestCount + 1))

				By("checking the API Server")
				var expected map[string]interface{}
				err = json.Unmarshal(minimalScalingRulePolicyJson, &expected)
				Expect(err).NotTo(HaveOccurred())

				// If custom metrics not enabled, credentials should not be created
				By("checking the credential table content")
				Expect(getCredentialsCount(appId)).To(Equal(0))

				checkResponseContent(getPolicy, appId, http.StatusOK, expected, components.Ports[APIServer], httpClient)
			})
		})

		Context("Invalid policy Schema", func() {
			BeforeEach(func() {
				fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})

			It("does not create a binding", func() {
				schedulerCount := len(fakeScheduler.ReceivedRequests())
				resp, err := bindService(bindingId, appId, serviceInstanceId, invalidSchemaPolicyJson, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				respBody, err := ioutil.ReadAll(resp.Body)
				Expect(string(respBody)).To(Equal(`{"description":[{"property":"instance","message":"is not any of [subschema 0],[subschema 1]","schema":"/policySchema","instance":{"instance_min_count":10,"instance_max_count":4},"name":"anyOf","argument":["[subschema 0]","[subschema 1]"],"stack":"instance is not any of [subschema 0],[subschema 1]"}]}`))
				resp.Body.Close()
				Consistently(fakeScheduler.ReceivedRequests).Should(HaveLen(schedulerCount))

				By("checking the API Server")
				resp, err = getPolicy(appId, components.Ports[APIServer], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})
		})

		Context("Invalid policy Data", func() {
			BeforeEach(func() {
				fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})

			It("does not create a binding", func() {
				schedulerCount := len(fakeScheduler.ReceivedRequests())
				resp, err := bindService(bindingId, appId, serviceInstanceId, invalidDataPolicyJson, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				respBody, err := ioutil.ReadAll(resp.Body)
				Expect(string(respBody)).To(Equal(`{"description":[{"property":"instance.scaling_rules[0].cool_down_secs","message":"must have a minimum value of 30","schema":{"type":"integer","minimum":30,"maximum":3600},"instance":-300,"name":"minimum","argument":30,"stack":"instance.scaling_rules[0].cool_down_secs must have a minimum value of 30"}]}`))
				resp.Body.Close()
				Consistently(fakeScheduler.ReceivedRequests).Should(HaveLen(schedulerCount))

				By("checking the API Server")
				resp, err = getPolicy(appId, components.Ports[APIServer], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})
		})

		Context("ApiServer is down", func() {
			BeforeEach(func() {
				stopApiServer()
				_, err := getPolicy(appId, components.Ports[APIServer], httpClient)
				Expect(err).To(HaveOccurred())
				fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))
			})

			It("should return 500", func() {
				schedulerCount := len(fakeScheduler.ReceivedRequests())
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()
				Consistently(fakeScheduler.ReceivedRequests).Should(HaveLen(schedulerCount))
			})
		})

		Context("Scheduler returns error", func() {
			BeforeEach(func() {
				fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))
			})

			It("should return 500", func() {
				schedulerCount := len(fakeScheduler.ReceivedRequests())
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()
				Consistently(fakeScheduler.ReceivedRequests).Should(HaveLen(schedulerCount + 1))

				By("checking the API Server")
				resp, err = getPolicy(appId, components.Ports[APIServer], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})
		})

		Context("Policy with schedules where Custom Metrics feature Enabled", func() {
			BeforeEach(func() {
				fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
				fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})

			AfterEach(func() {
				//clear the binding
				resp, err := unbindService(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			JustBeforeEach(func() {
				// Restarting servicebroker after enabling Custom Metrics feature
				stopServiceBroker()
				serviceBrokerConfPath = components.PrepareServiceBrokerConfig(components.Ports[ServiceBroker], components.Ports[ServiceBrokerInternal], brokerUserName, brokerPassword, true, dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[APIServer]), brokerApiHttpRequestTimeout, tmpDir)
				startServiceBroker()
			})

			It("creates a binding", func() {
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()
				Consistently(fakeScheduler.ReceivedRequests).Should(HaveLen(1))

				By("checking the API Server")
				var expected map[string]interface{}
				err = json.Unmarshal(schedulePolicyJson, &expected)
				Expect(err).NotTo(HaveOccurred())

				// If custom metrics enabled, credentials should be created
				By("checking the credential table content")
				Expect(getCredentialsCount(appId)).To(Equal(1))

				checkResponseContent(getPolicy, appId, http.StatusOK, expected, components.Ports[APIServer], httpClient)
			})
		})
	})

	Describe("Unbind Service", func() {
		BeforeEach(func() {
			brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
			//do a bind first
			fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson, components.Ports[ServiceBroker], httpClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			resp.Body.Close()
		})

		BeforeEach(func() {
			fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
		})

		It("should return 200", func() {
			resp, err := unbindService(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			resp.Body.Close()

			By("checking the API Server")
			resp, err = getPolicy(appId, components.Ports[APIServer], httpClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			resp.Body.Close()
		})

		Context("Policy does not exist", func() {
			BeforeEach(func() {
				fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
				//detach the appId's policy first
				resp, err := detachPolicy(appId, components.Ports[APIServer], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should return 200", func() {
				resp, err := unbindService(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})
		})

		Context("APIServer is down", func() {
			BeforeEach(func() {
				stopApiServer()
				_, err := detachPolicy(appId, components.Ports[APIServer], httpClient)
				Expect(err).To(HaveOccurred())
				fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})

			It("should return 500", func() {
				resp, err := unbindService(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()
			})
		})

		Context("Scheduler returns error", func() {
			BeforeEach(func() {
				fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))
			})

			It("should return 500 and not delete the binding info", func() {
				resp, err := unbindService(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()

				By("checking the API Server")
				resp, err = getPolicy(appId, components.Ports[APIServer], httpClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})
		})
	})

	Describe("Unbind Service with Custom Metrics enabled", func() {
		BeforeEach(func() {
			// Restarting servicebroker after enabling Custom Metrics feature
			stopServiceBroker()
			serviceBrokerConfPath = components.PrepareServiceBrokerConfig(components.Ports[ServiceBroker], components.Ports[ServiceBrokerInternal], brokerUserName, brokerPassword, true, dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[APIServer]), brokerApiHttpRequestTimeout, tmpDir)
			startServiceBroker()

			brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
			//do a bind first
			fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson, components.Ports[ServiceBroker], httpClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			resp.Body.Close()

			By("checking the credential table content")
			Expect(getCredentialsCount(appId)).To(Equal(1))
		})

		BeforeEach(func() {
			fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
		})

		It("should return 200", func() {
			resp, err := unbindService(bindingId, appId, serviceInstanceId, components.Ports[ServiceBroker], httpClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			resp.Body.Close()

			By("checking the API Server")
			resp, err = getPolicy(appId, components.Ports[APIServer], httpClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			resp.Body.Close()

			By("checking the credential table content")
			Expect(getCredentialsCount(appId)).To(Equal(0))
		})
	})
})
