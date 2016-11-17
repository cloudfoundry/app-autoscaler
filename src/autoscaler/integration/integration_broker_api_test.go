package integration_test

import (
	"encoding/base64"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var _ = Describe("Integration_Broker_Api", func() {

	var (
		httpClient         *http.Client
		regPath            = regexp.MustCompile(`^/v2/schedules/.*$`)
		brokerAuth         string
		schedulePolicyJson string = `
			{
		   "instance_min_count":1,
		   "instance_max_count":4,
		   "scaling_rules":[
		      {
		         "metric_type":"MemoryUsage",
		         "stat_window_secs":300,
		         "breach_duration_secs":600,
		         "threshold":30,
		         "operator":"<",
		         "cool_down_secs":300,
		         "adjustment":"-1"
		      },
		      {
		         "metric_type":"MemoryUsage",
		         "stat_window_secs":300,
		         "breach_duration_secs":600,
		         "threshold":90,
		         "operator":">=",
		         "cool_down_secs":300,
		         "adjustment":"+1"
		      }
		   ],
		   "schedules":{
		      "timezone":"Asia/Shanghai",
		      "recurring_schedule":[
		         {
		            "start_time":"10:00",
		            "end_time":"18:00",
		            "days_of_week":[
		               1,
		               2,
		               3
		            ],
		            "instance_min_count":1,
		            "instance_max_count":10,
		            "initial_min_instance_count":5
		         },
		         {
		            "start_date":"2099-06-27",
		            "end_date":"2099-07-23",
		            "start_time":"11:00",
		            "end_time":"19:30",
		            "days_of_month":[
		               5,
		               15,
		               25
		            ],
		            "instance_min_count":3,
		            "instance_max_count":10,
		            "initial_min_instance_count":5
		         },
		         {
		            "start_time":"10:00",
		            "end_time":"18:00",
		            "days_of_week":[
		               4,
		               5,
		               6
		            ],
		            "instance_min_count":1,
		            "instance_max_count":10
		         },
		         {
		            "start_time":"11:00",
		            "end_time":"19:30",
		            "days_of_month":[
		               10,
		               20,
		               30
		            ],
		            "instance_min_count":1,
		            "instance_max_count":10
		         }
		      ],
		      "specific_date":[
		         {
		            "start_date_time":"2099-06-02T10:00",
		            "end_date_time":"2099-06-15T13:59",
		            "instance_min_count":1,
		            "instance_max_count":4,
		            "initial_min_instance_count":2
		         },
		         {
		            "start_date_time":"2099-01-04T20:00",
		            "end_date_time":"2099-02-19T23:15",
		            "instance_min_count":2,
		            "instance_max_count":5,
		            "initial_min_instance_count":3
		         }
		      ]
		   }
		}`
		noSchedulePolicyJson string = `
			{
		   "instance_min_count":1,
		   "instance_max_count":4,
		   "scaling_rules":[
		      {
		         "metric_type":"MemoryUsage",
		         "stat_window_secs":300,
		         "breach_duration_secs":600,
		         "threshold":30,
		         "operator":"<",
		         "cool_down_secs":300,
		         "adjustment":"-1"
		      },
		      {
		         "metric_type":"MemoryUsage",
		         "stat_window_secs":300,
		         "breach_duration_secs":600,
		         "threshold":90,
		         "operator":">=",
		         "cool_down_secs":300,
		         "adjustment":"+1"
		      }
		   ]
		}`
		invalidPolicyJson string = `
			{
		   "instance_min_count":10,
		   "instance_max_count":4		  
		}`
		policyTemplate string = `{ "app_guid": "%s", "parameters": %s }`
	)
	BeforeEach(func() {
		brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
		clearDatabase()
	})
	Describe("Bind Service", func() {
		BeforeEach(func() {
			addServiceInstance(testServiceInstanceId, testOrgId, testSpaceId)
			httpClient = &http.Client{}
			Expect(getNumberOfBinding()).To(Equal(0))
			Expect(getNumberOfPolicyJson()).To(Equal(0))
		})
		Context("Policy with schedules", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})
			It("should return 201, save scaling rules to db and call scheduler", func() {
				policy := fmt.Sprintf(policyTemplate, testAppId, schedulePolicyJson)
				req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports["serviceBroker"], testServiceInstanceId, testBindingId), strings.NewReader(policy))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Basic "+brokerAuth)
				resp, err := httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(201))
				Eventually(getNumberOfBinding).Should(Equal(1))
				Eventually(getNumberOfPolicyJson).Should(Equal(1))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(1))
			})
		})
		Context("Policy without schedules", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))

			})
			It("should return 201, save scaling rules to db and not call scheduler", func() {
				policy := fmt.Sprintf(policyTemplate, testAppId, noSchedulePolicyJson)
				req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports["serviceBroker"], testServiceInstanceId, testBindingId), strings.NewReader(policy))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Basic "+brokerAuth)
				resp, err := httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(201))
				Eventually(getNumberOfBinding).Should(Equal(1))
				Eventually(getNumberOfPolicyJson).Should(Equal(1))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(0))
			})
		})
		Context("Invalid policy", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))

			})
			It("should return 400, save no scaling rule to db and not call scheduler", func() {
				policy := fmt.Sprintf(policyTemplate, testAppId, invalidPolicyJson)
				req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports["serviceBroker"], testServiceInstanceId, testBindingId), strings.NewReader(policy))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Basic "+brokerAuth)
				resp, err := httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(400))
				Eventually(getNumberOfBinding).Should(Equal(0))
				Eventually(getNumberOfPolicyJson).Should(Equal(0))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(0))
			})
		})
		Context("Scheduler returns error", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))

			})
			It("should return 500, save no scaling rule to db and call scheduler", func() {
				policy := fmt.Sprintf(policyTemplate, testAppId, schedulePolicyJson)
				req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports["serviceBroker"], testServiceInstanceId, testBindingId), strings.NewReader(policy))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Basic "+brokerAuth)
				resp, err := httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(500))
				Eventually(getNumberOfBinding).Should(Equal(0))
				Eventually(getNumberOfPolicyJson).Should(Equal(0))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(1))
			})
		})
	})

	Describe("UnBind Service", func() {
		BeforeEach(func() {
			addServiceInstance(testServiceInstanceId, testOrgId, testSpaceId)
			addBinding(testBindingId, testAppId, testServiceInstanceId, time.Now())
			addPolicy(testAppId, "{}", time.Now())
			brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
		})
		Context("Unbind service", func() {
			BeforeEach(func() {
				Expect(getNumberOfBinding()).To(Equal(1))
				Expect(getNumberOfPolicyJson()).To(Equal(1))
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))

			})
			It("should return 200 ,delete binding, policy_json and call scheduler", func() {
				req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports["serviceBroker"], testServiceInstanceId, testBindingId), strings.NewReader(""))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Basic "+brokerAuth)
				resp, err := httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(1))
				Eventually(getNumberOfBinding).Should(Equal(0))
				Eventually(getNumberOfPolicyJson).Should(Equal(0))
				Eventually(getNumberOfServiceInstance).Should(Equal(1))
			})
		})
		Context("Policy does not exist", func() {
			BeforeEach(func() {
				cleanPolicyJsonTable()
				Expect(getNumberOfBinding()).To(Equal(1))
				Expect(getNumberOfPolicyJson()).To(Equal(0))
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))

			})
			It("should return 200 ,delete the binding info and not call scheduler", func() {
				req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports["serviceBroker"], testServiceInstanceId, testBindingId), strings.NewReader(""))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Basic "+brokerAuth)
				resp, err := httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(0))
				Eventually(getNumberOfBinding).Should(Equal(0))
				Eventually(getNumberOfPolicyJson).Should(Equal(0))
				Eventually(getNumberOfServiceInstance).Should(Equal(1))
			})
		})
		Context("Scheduler returns error", func() {
			BeforeEach(func() {
				Expect(getNumberOfBinding()).To(Equal(1))
				Expect(getNumberOfPolicyJson()).To(Equal(1))
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))

			})
			It("should return 500 and not delete the binding info", func() {
				req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:%d/v2/service_instances/%s/service_bindings/%s", components.Ports["serviceBroker"], testServiceInstanceId, testBindingId), strings.NewReader(""))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Basic "+brokerAuth)
				resp, err := httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(500))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(1))
				Eventually(getNumberOfBinding).Should(Equal(1))
				Eventually(getNumberOfPolicyJson).Should(Equal(0))
				Eventually(getNumberOfServiceInstance).Should(Equal(1))
			})
		})
	})
})
