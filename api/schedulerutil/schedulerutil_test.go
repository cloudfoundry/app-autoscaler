package schedulerutil_test

import (
	"net/http"
	"net/url"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/schedulerutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Schedulerutil", func() {
	const (
		testAppId      = "test-app-id"
		testPolicyGuid = "test-policy-guid"
		testPolicyStr  = `{
			"instance_max_count":4,
			"instance_min_count":1,
			"schedules":{
				"timezone":"Asia/Kolkata",
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
					}
				]
			 }
		}`
	)
	var (
		schedulerUtil   *SchedulerUtil
		schedulerServer *ghttp.Server
		urlPath         *url.URL
		err             error
	)
	BeforeEach(func() {
		schedulerServer = ghttp.NewServer()
		conf := config.Config{
			Scheduler: config.SchedulerConfig{
				SchedulerURL: schedulerServer.URL(),
			},
		}
		logger := lager.NewLogger("schedulerutil")
		schedulerUtil = NewSchedulerUtil(&conf, logger)

		urlPath, _ = routes.SchedulerRoutes().Get(routes.UpdateScheduleRouteName).URLPath("appId", testAppId)

	})

	Context("when scheduler server is not running", func() {
		JustBeforeEach(func() {
			schedulerServer.Close()
		})
		It("should fail", func() {
			err = schedulerUtil.CreateOrUpdateSchedule(testAppId, testPolicyStr, testPolicyGuid)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("CreateOrUpdateSchedule", func() {
		Context("When it Scheduler return 200", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("PUT", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
			})
			It("should succeed", func() {
				err = schedulerUtil.CreateOrUpdateSchedule(testAppId, testPolicyStr, testPolicyGuid)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When it Scheduler return 204", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("PUT", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
			})
			It("should succeed", func() {
				err = schedulerUtil.CreateOrUpdateSchedule(testAppId, testPolicyStr, testPolicyGuid)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When it Scheduler return 400", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("PUT", urlPath.String(), ghttp.RespondWith(http.StatusBadRequest, "error in schedules"))
			})
			It("should succeed", func() {
				err = schedulerUtil.CreateOrUpdateSchedule(testAppId, testPolicyStr, testPolicyGuid)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Failed to create schedules due to validation errors in schedule : error in schedules"))
			})
		})

		Context("When it Scheduler return 500", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("PUT", urlPath.String(), ghttp.RespondWith(http.StatusInternalServerError, "error creating schedules"))
			})
			It("should succeed", func() {
				err = schedulerUtil.CreateOrUpdateSchedule(testAppId, testPolicyStr, testPolicyGuid)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Error occurred in scheduler module during creation/update : error creating schedules"))
			})
		})
	})

	Describe("DeleteSchedule", func() {
		Context("When it Scheduler return 200", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
			})
			It("should succeed", func() {
				err = schedulerUtil.DeleteSchedule(testAppId)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When it Scheduler return 204", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
			})
			It("should succeed", func() {
				err = schedulerUtil.DeleteSchedule(testAppId)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When it Scheduler return 404", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusNotFound, nil))
			})
			It("should succeed", func() {
				err = schedulerUtil.DeleteSchedule(testAppId)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When it Scheduler return 500", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusInternalServerError, "error deleting schedules"))
			})
			It("should succeed", func() {
				err = schedulerUtil.DeleteSchedule(testAppId)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Error occurred in scheduler module during deletion : error deleting schedules"))
			})
		})
	})
})
