package app_test

import (
	"acceptance"
	. "acceptance/helpers"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AutoScaler recurring schedule policy", func() {
	var (
		initialInstanceCount int
		daysOfMonthOrWeek    Days
		err                  error
		startTime            time.Time
		endTime              time.Time
		policy               string
	)

	BeforeEach(func() {
		instanceName = CreateService(cfg)
		initialInstanceCount = 1
		appToScaleName = CreateTestApp(cfg, "recurring-schedule", initialInstanceCount)
		appToScaleGUID, err = GetAppGuid(cfg, appToScaleName)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(AppAfterEach)

	Context("when scaling by recurring schedule", func() {

		scheduleInitialMinInstanceCount := 3
		scheduleInstanceMinCount := 2
		instanceMinCount := 1
		JustBeforeEach(func() {
			startTime, endTime = getStartAndEndTime(time.UTC, 70*time.Second, time.Duration(interval+120)*time.Second)
			policy = GenerateDynamicAndRecurringSchedulePolicy(instanceMinCount, 4, 50, "UTC", startTime, endTime, daysOfMonthOrWeek, scheduleInstanceMinCount, 5, scheduleInitialMinInstanceCount)
			instanceName = CreatePolicy(cfg, appToScaleName, appToScaleGUID, policy)
			StartApp(appToScaleName, cfg.CfPushTimeoutDuration())
		})

		scaleDown := func() {
			By("setting to initial_min_instance_count")
			jobRunTime := time.Until(startTime.Add(5 * time.Minute))
			WaitForNInstancesRunning(appToScaleGUID, scheduleInitialMinInstanceCount, jobRunTime, "The schedule should initially trigger scaling to initial_min_instance_count %i", scheduleInitialMinInstanceCount)

			By("setting schedule's instance_min_count")
			jobRunTime = time.Until(endTime)
			WaitForNInstancesRunning(appToScaleGUID, scheduleInstanceMinCount, jobRunTime, "The schedule should allow scaling down to instance_min_count %i", scheduleInstanceMinCount)

			By("setting to default instance_min_count")
			WaitForNInstancesRunning(appToScaleGUID, instanceMinCount, time.Until(endTime.Add(time.Duration(interval+60)*time.Second)), "After the schedule ended scaling down to instance_min_count %i should be possible", instanceMinCount)
		}

		Context("with days of month", func() {
			BeforeEach(func() { daysOfMonthOrWeek = DaysOfMonth })
			It("should scale", scaleDown)
		})

		Context("with days of week", func() {
			BeforeEach(func() { daysOfMonthOrWeek = DaysOfWeek })
			It("should scale", Label(acceptance.LabelSmokeTests), scaleDown)
		})
	})

})
