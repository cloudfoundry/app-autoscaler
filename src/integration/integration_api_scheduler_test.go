package integration_test

import (
	. "integration"

	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"io/ioutil"
	"net/http"
)

var _ = Describe("Integration_Api_Scheduler", func() {

	var (
		appId     string
		policyStr []byte
	)
	BeforeEach(func() {
		httpClient.Timeout = apiSchedulerHttpRequestTimeout
		fakeScalingEngine = ghttp.NewServer()
		apiServerConfPath = prepareApiServerConfig(components.Ports[APIServer], dbUrl, fmt.Sprintf("http://127.0.0.1:%d", components.Ports[Scheduler]))
		schedulerConfPath = prepareSchedulerConfig(dbUrl, fakeScalingEngine.URL())
		startApiServer()
		startScheduler()
		appId = getRandomId()
		resp, err := detachPolicy(appId)
		resp.Body.Close()
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		fakeScalingEngine.Close()
		stopAll()
	})
	Describe("Create policy", func() {
		Context("Policies with schedules", func() {
			It("creates a policy and associated schedules", func() {
				policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
				resp, err := attachPolicy(appId, policyStr)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()

				By("checking the API Server")
				var expected map[string]interface{}
				err = json.Unmarshal(policyStr, &expected)
				Expect(err).NotTo(HaveOccurred())
				checkResponseContent(getPolicy, appId, http.StatusOK, expected)

				By("checking the Scheduler")
				checkSchedule(getSchedules, appId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})

			})
			It("fails with an invalid policy", func() {
				policyStr = readPolicyFromFile("fakeInvalidPolicy.json")
				resp, err := attachPolicy(appId, policyStr)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()
				By("checking the API Server")
				resp, err = getPolicy(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
				By("checking the Scheduler")
				resp, err = getSchedules(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()

			})
		})

		Context("Policies without schedules", func() {
			It("creates only the policy", func() {
				policyStr = readPolicyFromFile("fakePolicyWithoutSchedule.json")
				resp, err := attachPolicy(appId, policyStr)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()
				By("checking the API Server")
				var expected map[string]interface{}
				err = json.Unmarshal(policyStr, &expected)
				Expect(err).NotTo(HaveOccurred())

				checkResponseContent(getPolicy, appId, http.StatusOK, expected)

				By("checking the Scheduler")
				resp, err = getSchedules(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()

			})
		})
	})
	Describe("Update policy", func() {
		Context("Update policies with schedules", func() {
			BeforeEach(func() {
				//attach a policy first with 4 recurring and 2 specific_date schedules
				policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
				resp, err := attachPolicy(appId, policyStr)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()

			})
			It("updates the policy and schedules", func() {
				//attach another policy with 3 recurring and 1 specific_date schedules
				policyStr = readPolicyFromFile("fakePolicyWithScheduleAnother.json")
				resp, err := attachPolicy(appId, policyStr)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
				By("checking the API Server")
				var expected map[string]interface{}
				err = json.Unmarshal(policyStr, &expected)
				Expect(err).NotTo(HaveOccurred())
				checkResponseContent(getPolicy, appId, http.StatusOK, expected)

				By("checking the Scheduler")
				checkSchedule(getSchedules, appId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})

			})
		})
	})

	Describe("Delete Policies", func() {
		Context("for a non-existing app", func() {
			It("Should return a NOT FOUND (404)", func() {
				resp, err := detachPolicy(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				respBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(respBody)).To(Equal(`{"success":false,"error":{"message":"No policy bound with application","statusCode":404},"result":null}`))
				resp.Body.Close()
			})
		})
		Context("with an existing app", func() {
			BeforeEach(func() {
				//attach a policy first with 4 recurring and 2 specific_date schedules
				policyStr = readPolicyFromFile("fakePolicyWithSchedule.json")
				resp, err := attachPolicy(appId, policyStr)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				resp.Body.Close()

			})
			It("deletes the policy and schedules", func() {
				resp, err := detachPolicy(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()
				By("checking the API Server")
				resp, err = getPolicy(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()

				By("checking the Scheduler")
				resp, err = getSchedules(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()

			})
		})
	})

})
