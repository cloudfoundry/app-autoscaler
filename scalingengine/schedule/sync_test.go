package schedule_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/schedule"

	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"errors"
	"time"
)

const TestSyncInterval = 10 * time.Second

var _ = Describe("Sync", func() {

	var (
		schedulerDB  *fakes.FakeSchedulerDB
		engineDB     *fakes.FakeScalingEngineDB
		engine       *fakes.FakeScalingEngine
		synchronizer ActiveScheduleSychronizer
		buffer       *gbytes.Buffer
	)

	BeforeEach(func() {
		schedulerDB = &fakes.FakeSchedulerDB{}
		engineDB = &fakes.FakeScalingEngineDB{}
		engine = &fakes.FakeScalingEngine{}
		logger := lagertest.NewTestLogger("active-schedule-synchronizer-test")
		buffer = logger.Buffer()
		synchronizer = NewActiveScheduleSychronizer(logger, schedulerDB, engineDB, engine)
	})

	Describe("Sync", func() {

		JustBeforeEach(func() {
			go synchronizer.Sync()
			Eventually(buffer).Should(gbytes.Say("synchronizing-active-schedules"))
		})

		It("synchronizes data between scheduler and scaling engine with the given time interval", func() {
			Eventually(schedulerDB.GetActiveSchedulesCallCount).Should(Equal(1))
			Eventually(engineDB.GetActiveSchedulesCallCount).Should(Equal(1))
		})

		Context("when data are consistent", func() {
			BeforeEach(func() {
				schedulerDB.GetActiveSchedulesReturns(map[string]*models.ActiveSchedule{
					"app-id-1": {
						ScheduleId: "schedule-id-1",
					},
					"app-id-2": {
						ScheduleId: "schedule-id-2",
					},
				}, nil)

				engineDB.GetActiveSchedulesReturns(map[string]string{
					"app-id-1": "schedule-id-1",
					"app-id-2": "schedule-id-2",
				}, nil)
			})

			It("does nothing", func() {
				Consistently(engineDB.SetActiveScheduleCallCount).Should(BeZero())
				Consistently(engineDB.RemoveActiveScheduleCallCount).Should(BeZero())
			})
		})

		Context("there is missing active schedule start", func() {
			BeforeEach(func() {
				schedulerDB.GetActiveSchedulesReturns(map[string]*models.ActiveSchedule{
					"app-id-1": {
						ScheduleId: "schedule-id-1",
					},
					"app-id-2": {
						ScheduleId: "schedule-id-2",
					},
				}, nil)

				engineDB.GetActiveSchedulesReturns(map[string]string{
					"app-id-1": "schedule-id-1",
				}, nil)
			})

			It("set the active schedule", func() {
				Eventually(buffer).Should(gbytes.Say("synchronize-active-schedules-find-missing-active-schedule-start"))
				Eventually(engine.SetActiveScheduleCallCount).Should(Equal(1))
				appId, schedule := engine.SetActiveScheduleArgsForCall(0)
				Expect(appId).To(Equal("app-id-2"))
				Expect(schedule).To(Equal(&models.ActiveSchedule{ScheduleId: "schedule-id-2"}))
			})
		})

		Context("there is new active schedule in schedule DB for an app", func() {
			BeforeEach(func() {
				schedulerDB.GetActiveSchedulesReturns(map[string]*models.ActiveSchedule{
					"app-id-1": {
						ScheduleId: "schedule-id-1",
					},
					"app-id-2": {
						ScheduleId: "schedule-id-2-2",
					},
				}, nil)

				engineDB.GetActiveSchedulesReturns(map[string]string{
					"app-id-1": "schedule-id-1",
					"app-id-2": "schedule-id-2-1",
				}, nil)
			})

			It("set the new active schedule", func() {
				Eventually(buffer).Should(gbytes.Say("synchronize-active-schedules-find-missing-active-schedule-start"))
				Eventually(engine.SetActiveScheduleCallCount).Should(Equal(1))
				appId, schedule := engine.SetActiveScheduleArgsForCall(0)
				Expect(appId).To(Equal("app-id-2"))
				Expect(schedule).To(Equal(&models.ActiveSchedule{ScheduleId: "schedule-id-2-2"}))
			})
		})

		Context("there is missing active schedule end", func() {
			BeforeEach(func() {
				schedulerDB.GetActiveSchedulesReturns(map[string]*models.ActiveSchedule{
					"app-id-1": {
						ScheduleId: "schedule-id-1",
					},
				}, nil)

				engineDB.GetActiveSchedulesReturns(map[string]string{
					"app-id-1": "schedule-id-1",
					"app-id-2": "schedule-id-2",
				}, nil)
			})

			It("set the active schedule", func() {
				Eventually(buffer).Should(gbytes.Say("synchronize-active-schedules-find-missing-active-schedule-end"))
				Eventually(engine.RemoveActiveScheduleCallCount).Should(Equal(1))
				appId, scheduleId := engine.RemoveActiveScheduleArgsForCall(0)
				Expect(appId).To(Equal("app-id-2"))
				Expect(scheduleId).To(Equal("schedule-id-2"))
			})
		})

		Context("when getting active schedule from scheduler db fails", func() {
			BeforeEach(func() {
				schedulerDB.GetActiveSchedulesReturns(nil, errors.New("an error"))
			})

			It("skips the synchronization", func() {
				Eventually(buffer).Should(gbytes.Say("failed-synchronize-active-schedules-get-schedules-from-schedulerDB"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})

		Context("when getting active schedule from engine db fails", func() {
			BeforeEach(func() {
				engineDB.GetActiveSchedulesReturns(nil, errors.New("an error"))
			})

			It("skips the synchronization", func() {
				Eventually(buffer).Should(gbytes.Say("failed-synchronize-active-schedules-get-schedules-from-engineDB"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})
	})

})
