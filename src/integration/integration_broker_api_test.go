package integration_test

import (
	"encoding/base64"
	"encoding/json"
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
		schedulePolicyJson = []byte(`
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
		}`)
		invalidPolicyJson = []byte(`
			{
		   "instance_min_count":10,
		   "instance_max_count":4
		}`)
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
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(201))
		resp.Body.Close()
	})

	AfterEach(func() {
		//clean the service instance added in before each
		resp, err := deprovisionServiceInstance(serviceInstanceId)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(200))
		resp.Body.Close()
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
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				resp.Body.Close()
			})

			It("creates a binding", func() {
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(201))
				resp.Body.Close()
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(1))

				By("checking the API Server")
				var expected map[string]interface{}
				err = json.Unmarshal(schedulePolicyJson, &expected)
				Expect(err).NotTo(HaveOccurred())

				resp, err = getPolicy(appId)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				var actual map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&actual)
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(Equal(expected))
			})
		})

		Context("Invalid policy", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			})

			It("does not create a binding", func() {
				schedulerCount := len(scheduler.ReceivedRequests())
				resp, err := bindService(bindingId, appId, serviceInstanceId, invalidPolicyJson)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(400))
				resp.Body.Close()
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(schedulerCount))

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
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))
			})

			It("should return 500", func() {
				schedulerCount := len(scheduler.ReceivedRequests())
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(schedulerCount))
			})
		})

		Context("Scheduler returns error", func() {
			BeforeEach(func() {
				scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))
			})

			It("should return 500", func() {
				schedulerCount := len(scheduler.ReceivedRequests())
				resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
				resp.Body.Close()
				Consistently(scheduler.ReceivedRequests).Should(HaveLen(schedulerCount + 1))

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
			scheduler.RouteToHandler("PUT", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
			resp, err := bindService(bindingId, appId, serviceInstanceId, schedulePolicyJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			resp.Body.Close()
		})

		BeforeEach(func() {
			scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
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
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
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
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
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
				scheduler.RouteToHandler("DELETE", regPath, ghttp.RespondWith(http.StatusInternalServerError, "error"))
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
