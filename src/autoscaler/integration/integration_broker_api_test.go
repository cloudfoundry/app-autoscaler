package integration_test

import (
	"encoding/base64"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"net/http"
	"regexp"
)

var _ = Describe("Integration_Broker_Api", func() {

	var (
		regPath = regexp.MustCompile(`^/v2/schedules/.*$`)

		serviceInstanceId  string
		bindingId          string
		orgId              string
		spaceId            string
		appId              string
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
		invalidPolicyJson string = `
			{
		   "instance_min_count":10,
		   "instance_max_count":4
		}`
	)
	BeforeEach(func() {

		brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
		serviceInstanceId = getRandomId()
		orgId = getRandomId()
		spaceId = getRandomId()
		bindingId = getRandomId()
		appId = getRandomId()
		//add a service instance
		resp, err := provisionServiceInstance(serviceInstanceId, orgId, spaceId)
		defer resp.Body.Close()
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(201))

		Eventually(func() int {
			return getNumberOfBinding(bindingId, appId, serviceInstanceId)
		}).Should(Equal(0))
		Eventually(func() int {
			return getNumberOfPolicyJson(appId)
		}).Should(Equal(0))

	})
	AfterEach(func() {
		//clean the service instance added in before each
		resp, err := deprovisionServiceInstance(serviceInstanceId)
		defer resp.Body.Close()
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(200))
		Eventually(func() int {
			return getNumberOfServiceInstance(serviceInstanceId, orgId, spaceId)
		}).Should(Equal(0))

	})
	Describe("Bind Service", func() {

		Context("Policy with schedules", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})
			AfterEach(func() {
				//clear the binding
				resp, err := unbindService(bindingId, appId, serviceInstanceId)
				defer resp.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				Eventually(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(0))

			})
			It("creates a binding,, save scaling rules to db and call scheduler", func() {
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
				defer resp.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(201))
				Eventually(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(1))
				Eventually(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(1))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(1))

			})
		})
		Context("Invalid policy", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))

			})
			It("does not create a binding", func() {

				resp, err := bindService(bindingId, appId, serviceInstanceId, invalidPolicyJson)
				defer resp.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(400))
				Eventually(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(0))
				Eventually(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(0))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(0))
			})
		})
		Context("Api-server is down", func() {
			BeforeEach(func() {
				stopApiServer()
				_, err := attachPolicy(appId, schedulePolicyJson)
				Expect(err).To(HaveOccurred())
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))

			})
			It("should return 500, save no scaling rule to db and call scheduler", func() {
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
				defer resp.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(500))
				Eventually(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(0))
				Eventually(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(0))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(0))
			})
		})
		Context("Scheduler returns error", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))

			})
			It("should return 500, save no scaling rule to db and call scheduler", func() {
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
				defer resp.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(500))
				Eventually(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(0))
				Eventually(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(0))
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(1))
			})
		})
	})

	Describe("UnBind Service", func() {
		BeforeEach(func() {
			brokerAuth = base64.StdEncoding.EncodeToString([]byte("username:password"))
			//do a bind first
			scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
			defer resp.Body.Close()
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(201))
			Eventually(func() int {
				return getNumberOfBinding(bindingId, appId, serviceInstanceId)
			}).Should(Equal(1))
			Eventually(func() int {
				return getNumberOfPolicyJson(appId)
			}).Should(Equal(1))
			Consistently(scheduler.ReceivedRequests).Should(HaveLen(1))

		})
		Context("Unbind service", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))

			})
			It("should return 200 ,delete binding, policy_json and call scheduler", func() {
				resp, err := unbindService(bindingId, appId, serviceInstanceId)
				defer resp.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))

				Eventually(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(0))
				Eventually(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(0))
			})
		})
		Context("Policy does not exist", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
				//detach the appId's policy first
				resp, err := detachPolicy(appId)
				defer resp.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				Eventually(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(0))

			})
			It("should return 200 ,delete the binding info and not call scheduler", func() {
				resp, err := unbindService(bindingId, appId, serviceInstanceId)
				defer resp.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))

				Eventually(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(0))
				Eventually(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(0))
			})
		})
		Context("Api-server is down", func() {
			BeforeEach(func() {
				stopApiServer()
				_, err := detachPolicy(appId)
				Expect(err).To(HaveOccurred())
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))

			})
			AfterEach(func() {
				removeBinding(bindingId)
				removePolicy(appId)
				Eventually(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(0))
				Eventually(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(0))
			})
			It("should return 500 and not delete the binding info", func() {
				resp, err := unbindService(bindingId, appId, serviceInstanceId)
				defer resp.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(500))

				Consistently(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(1))
				Consistently(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(1))
			})
		})
		Context("Scheduler returns error", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))

			})
			AfterEach(func() {
				removeBinding(bindingId)
				removePolicy(appId)
				Eventually(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(0))
				Eventually(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(0))
			})
			It("should return 500 and not delete the binding info", func() {
				resp, err := unbindService(bindingId, appId, serviceInstanceId)
				defer resp.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(500))

				Consistently(func() int {
					return getNumberOfBinding(bindingId, appId, serviceInstanceId)
				}).Should(Equal(1))
				Consistently(func() int {
					return getNumberOfPolicyJson(appId)
				}).Should(Equal(0))
			})
		})
	})
})
