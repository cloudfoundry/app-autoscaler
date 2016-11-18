package schedule_test

import (
	"autoscaler/models"
	"autoscaler/scalingengine/fakes"
	. "autoscaler/scalingengine/schedule"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"errors"
	"os"
	"time"
)

const TestSyncInterval time.Duration = 10 * time.Second

var _ = Describe("Sync", func() {

	var (
		schedulerDB  *fakes.FakeSchedulerDB
		engineDB     *fakes.FakeScalingEngineDB
		engine       *fakes.FakeScalingEngine
		fclock       *fakeclock.FakeClock
		synchronizer *ActiveScheduleSychronizer
		buffer       *gbytes.Buffer
		signals      chan os.Signal
		ready        chan struct{}
	)

	BeforeEach(func() {
		schedulerDB = &fakes.FakeSchedulerDB{}
		engineDB = &fakes.FakeScalingEngineDB{}
		engine = &fakes.FakeScalingEngine{}
		logger := lagertest.NewTestLogger("active-schedule-synchronizer-test")
		buffer = logger.Buffer()
		fclock = fakeclock.NewFakeClock(time.Now())
		synchronizer = NewActiveScheduleSychronizer(logger, schedulerDB, engineDB, engine, TestSyncInterval, fclock)
	})

	Describe("Run", func() {
		BeforeEach(func() {
			signals = make(chan os.Signal)
			ready = make(chan struct{})
		})
		AfterEach(func() {
			close(signals)
		})
		JustBeforeEach(func() {
			go synchronizer.Run(signals, ready)
		})

		It("starts synchronization and can be stopped", func() {
			Eventually(buffer).Should(gbytes.Say("started"))
			signals <- os.Interrupt
			Eventually(buffer).Should(gbytes.Say("stopped"))
		})

		It("synchronizes data between scheduler and scaling engine with the given time interval", func() {
			fclock.WaitForWatcherAndIncrement(TestSyncInterval)
			Eventually(schedulerDB.GetActiveSchedulesCallCount).Should(Equal(1))
			Eventually(engineDB.GetActiveSchedulesCallCount).Should(Equal(1))

			fclock.Increment(TestSyncInterval)
			Eventually(schedulerDB.GetActiveSchedulesCallCount).Should(Equal(2))
			Eventually(engineDB.GetActiveSchedulesCallCount).Should(Equal(2))

			signals <- os.Interrupt
		})

		Context("when data are consistent", func() {
			BeforeEach(func() {
				schedulerDB.GetActiveSchedulesReturns(map[string]*models.ActiveSchedule{
					"app-id-1": &models.ActiveSchedule{
						ScheduleId: "schedule-id-1",
					},
					"app-id-2": &models.ActiveSchedule{
						ScheduleId: "schedule-id-2",
					},
				}, nil)

				engineDB.GetActiveSchedulesReturns(map[string]string{
					"app-id-1": "schedule-id-1",
					"app-id-2": "schedule-id-2",
				}, nil)
			})
			AfterEach(func() {
				signals <- os.Interrupt
			})

			It("does nothing", func() {
				fclock.WaitForWatcherAndIncrement(TestSyncInterval)
				Consistently(engineDB.SetActiveScheduleCallCount).Should(BeZero())
				Consistently(engineDB.RemoveActiveScheduleCallCount).Should(BeZero())
			})
		})

		Context("there is missing active schedule start", func() {
			BeforeEach(func() {
				schedulerDB.GetActiveSchedulesReturns(map[string]*models.ActiveSchedule{
					"app-id-1": &models.ActiveSchedule{
						ScheduleId: "schedule-id-1",
					},
					"app-id-2": &models.ActiveSchedule{
						ScheduleId: "schedule-id-2",
					},
				}, nil)

				engineDB.GetActiveSchedulesReturns(map[string]string{
					"app-id-1": "schedule-id-1",
				}, nil)
			})
			AfterEach(func() {
				signals <- os.Interrupt
			})

			It("set the active schedule", func() {
				fclock.WaitForWatcherAndIncrement(TestSyncInterval)
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
					"app-id-1": &models.ActiveSchedule{
						ScheduleId: "schedule-id-1",
					},
					"app-id-2": &models.ActiveSchedule{
						ScheduleId: "schedule-id-2-2",
					},
				}, nil)

				engineDB.GetActiveSchedulesReturns(map[string]string{
					"app-id-1": "schedule-id-1",
					"app-id-2": "schedule-id-2-1",
				}, nil)
			})
			AfterEach(func() {
				signals <- os.Interrupt
			})

			It("set the new active schedule", func() {
				fclock.WaitForWatcherAndIncrement(TestSyncInterval)
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
					"app-id-1": &models.ActiveSchedule{
						ScheduleId: "schedule-id-1",
					},
				}, nil)

				engineDB.GetActiveSchedulesReturns(map[string]string{
					"app-id-1": "schedule-id-1",
					"app-id-2": "schedule-id-2",
				}, nil)
			})
			AfterEach(func() {
				signals <- os.Interrupt
			})

			It("set the active schedule", func() {
				fclock.WaitForWatcherAndIncrement(TestSyncInterval)
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
			AfterEach(func() {
				signals <- os.Interrupt
			})

			It("skips the synchronization", func() {
				fclock.WaitForWatcherAndIncrement(TestSyncInterval)
				Eventually(buffer).Should(gbytes.Say("failed-synchronize-active-schedules-get-schedules-from-schedulerDB"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})

		Context("when getting active schedule from engine db fails", func() {
			BeforeEach(func() {
				engineDB.GetActiveSchedulesReturns(nil, errors.New("an error"))
			})
			AfterEach(func() {
				signals <- os.Interrupt
			})
			It("skips the synchronization", func() {
				fclock.WaitForWatcherAndIncrement(TestSyncInterval)
				Eventually(buffer).Should(gbytes.Say("failed-synchronize-active-schedules-get-schedules-from-engineDB"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})
	})

})
