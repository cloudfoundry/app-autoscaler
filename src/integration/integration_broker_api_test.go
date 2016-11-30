package integration_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	. "integration"
	"net/http"
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Integration_Broker_Api", func() {

	var (
		regPath = regexp.MustCompile(`^/v2/schedules/.*$`)

		serviceInstanceId  string
		bindingId          string
		orgId              string
		spaceId            string
		appId              string
		schedulePolicyJson []byte
		invalidPolicyJson  []byte
	)

	BeforeEach(func() {
		httpClient.Timeout = brokerApiHttpRequestTimeout
		fakeScheduler = ghttp.NewServer()
		apiServerConfPath = prepareApiServerConfig(components.Ports[APIServer], dbUrl, fakeScheduler.URL())
		serviceBrokerConfPath = prepareServiceBrokerConfig(components.Ports[ServiceBroker], brokerUserName, brokerPassword, dbUrl, fmt.Sprintf("http://127.0.0.1:%d", components.Ports[APIServer]))

		startApiServer()
		startServiceBroker()
		brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
		serviceInstanceId = getRandomId()
		orgId = getRandomId()
		spaceId = getRandomId()
		bindingId = getRandomId()
		appId = getRandomId()
		//add a service instance
		resp, err := provisionServiceInstance(serviceInstanceId, orgId, spaceId)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusCreated))
		resp.Body.Close()

		schedulePolicyJson = readPolicyFromFile("fakePolicyWithSchedule.json")
		invalidPolicyJson = readPolicyFromFile("fakeInvalidPolicy.json")
	})

	AfterEach(func() {
		//clean the service instance added in before each
		resp, err := deprovisionServiceInstance(serviceInstanceId)
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
				resp, err := unbindService(bindingId, appId, serviceInstanceId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("creates a binding", func() {
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()
				Consistently(fakeScheduler.ReceivedRequests).Should(HaveLen(1))

				By("checking the API Server")
				var expected map[string]interface{}
				err = json.Unmarshal(schedulePolicyJson, &expected)
				Expect(err).NotTo(HaveOccurred())

				checkResponseContent(getPolicy, appId, http.StatusOK, expected)
			})
		})

		Context("Invalid policy", func() {
			BeforeEach(func() {
				fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})

			It("does not create a binding", func() {
				schedulerCount := len(fakeScheduler.ReceivedRequests())
				resp, err := bindService(bindingId, appId, serviceInstanceId, invalidPolicyJson)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()
				Consistently(fakeScheduler.ReceivedRequests).Should(HaveLen(schedulerCount))

				By("checking the API Server")
				resp, err = getPolicy(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})
		})

		Context("ApiServer is down", func() {
			BeforeEach(func() {
				stopApiServer()
				_, err := getPolicy(appId)
				Expect(err).To(HaveOccurred())
				fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))
			})

			It("should return 500", func() {
				schedulerCount := len(fakeScheduler.ReceivedRequests())
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
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
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()
				Consistently(fakeScheduler.ReceivedRequests).Should(HaveLen(schedulerCount + 1))

				By("checking the API Server")
				resp, err = getPolicy(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})
		})
	})

	Describe("Unbind Service", func() {
		BeforeEach(func() {
			brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
			//do a bind first
			fakeScheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			resp.Body.Close()
		})

		BeforeEach(func() {
			fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
		})

		It("should return 200", func() {
			resp, err := unbindService(bindingId, appId, serviceInstanceId)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			resp.Body.Close()

			By("checking the API Server")
			resp, err = getPolicy(appId)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			resp.Body.Close()
		})

		Context("Policy does not exist", func() {
			BeforeEach(func() {
				fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
				//detach the appId's policy first
				resp, err := detachPolicy(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})

			It("should return 200", func() {
				resp, err := unbindService(bindingId, appId, serviceInstanceId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
			})
		})

		Context("APIServer is down", func() {
			BeforeEach(func() {
				stopApiServer()
				_, err := detachPolicy(appId)
				Expect(err).To(HaveOccurred())
				fakeScheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})

			It("should return 500", func() {
				resp, err := unbindService(bindingId, appId, serviceInstanceId)
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
				resp, err := unbindService(bindingId, appId, serviceInstanceId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()

				By("checking the API Server")
				resp, err = getPolicy(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})
		})
	})
})
