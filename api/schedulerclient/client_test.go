package schedulerclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/schedulerclient"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Scheduler Client", func() {
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
		schedulerUtil   *Client
		schedulerServer *ghttp.Server
		urlPath         *url.URL
		err             error
		policy          *models.ScalingPolicy
	)
	BeforeEach(func() {
		schedulerServer = ghttp.NewServer()
		conf := config.Config{
			Scheduler: config.SchedulerConfig{
				SchedulerURL: schedulerServer.URL(),
			},
		}
		logger := lager.NewLogger("schedulerutil")
		schedulerUtil = New(&conf, logger)

		urlPath, _ = routes.SchedulerRoutes().Get(routes.UpdateScheduleRouteName).URLPath("appId", testAppId)
		err := json.Unmarshal([]byte(testPolicyStr), &policy)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when scheduler server is not running", func() {
		JustBeforeEach(func() {
			schedulerServer.Close()
		})
		It("should fail", func() {
			err = schedulerUtil.CreateOrUpdateSchedule(context.Background(), testAppId, policy, testPolicyGuid)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("CreateOrUpdateSchedule", func() {
		Context("When it Scheduler return 200", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("PUT", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
			})
			It("should succeed", func() {
				err = schedulerUtil.CreateOrUpdateSchedule(context.Background(), testAppId, policy, testPolicyGuid)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When it Scheduler return 204", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("PUT", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
			})
			It("should succeed", func() {
				err = schedulerUtil.CreateOrUpdateSchedule(context.Background(), testAppId, policy, testPolicyGuid)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When Scheduler returns non ok", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("PUT", urlPath.String(), ghttp.RespondWith(http.StatusInternalServerError, "error creating schedules"))
			})
			It("should return err with message", func() {
				err = schedulerUtil.CreateOrUpdateSchedule(context.Background(), testAppId, policy, testPolicyGuid)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("unable to creation/update schedule: error creating schedules"))
			})
		})
	})

	Describe("DeleteSchedule", func() {
		Context("When it Scheduler return 200", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
			})
			It("should succeed", func() {
				err = schedulerUtil.DeleteSchedule(context.Background(), testAppId)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When it Scheduler return 204", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
			})
			It("should succeed", func() {
				err = schedulerUtil.DeleteSchedule(context.Background(), testAppId)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When it Scheduler return 404", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusNotFound, nil))
			})
			It("should succeed", func() {
				err = schedulerUtil.DeleteSchedule(context.Background(), testAppId)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When it Scheduler return 500", func() {
			JustBeforeEach(func() {
				schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusInternalServerError, "error deleting schedules"))
			})
			It("should succeed", func() {
				err = schedulerUtil.DeleteSchedule(context.Background(), testAppId)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("error deleting schedules"))
			})
		})
	})
})
