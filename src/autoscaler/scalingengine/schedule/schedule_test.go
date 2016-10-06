package schedule_test

import (
	"autoscaler/models"
	"autoscaler/scalingengine/fakes"

	. "autoscaler/scalingengine/schedule"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"errors"
)

var _ = Describe("Schedule", func() {

	var (
		schedules      *AppSchedules
		cfc            *fakes.FakeCfClient
		policyDB       *fakes.FakePolicyDB
		buffer         *gbytes.Buffer
		activeSchedule ActiveSchedule
		err            error
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		policyDB = &fakes.FakePolicyDB{}
		logger := lagertest.NewTestLogger("schedule-test")
		buffer = logger.Buffer()
		schedules = NewAppSchedules(logger, cfc, policyDB)
		activeSchedule = ActiveSchedule{
			ScheduleId:      "a-schedule-id",
			InstanceInitial: 5,
			InstanceMin:     3,
			InstanceMax:     10,
		}
	})

	Describe("SetActiveSchedule", func() {
		JustBeforeEach(func() {
			err = schedules.SetActiveSchedule("an-app-id", activeSchedule)
		})

		Context("when no active schedule exists for the app", func() {
			It("sets the active schedule and initial instances", func() {
				Expect(err).NotTo(HaveOccurred())
				appId, instances := cfc.SetAppInstancesArgsForCall(0)
				Expect(appId).To(Equal("an-app-id"))
				Expect(instances).To(Equal(5))

				schedule, exists := schedules.GetActiveSchedule("an-app-id")
				Expect(exists).To(BeTrue())
				Expect(schedule).To(Equal(activeSchedule))
			})

		})

		Context("when active schedule exists for the app", func() {
			BeforeEach(func() {
				err = schedules.SetActiveSchedule("an-app-id", ActiveSchedule{
					ScheduleId:      "currrent-schedule-id",
					InstanceInitial: 6,
					InstanceMin:     2,
					InstanceMax:     8,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("Sets the new active scheule and logs the info", func() {
				Expect(err).NotTo(HaveOccurred())
				schedule, exists := schedules.GetActiveSchedule("an-app-id")
				Expect(exists).To(BeTrue())
				Expect(schedule).To(Equal(activeSchedule))

				appId, instances := cfc.SetAppInstancesArgsForCall(1)
				Expect(appId).To(Equal("an-app-id"))
				Expect(instances).To(Equal(5))

				Eventually(buffer).Should(gbytes.Say("active schedule exists"))
			})
		})

		Context("when fails to set initial instances number", func() {
			BeforeEach(func() {
				cfc.SetAppInstancesReturns(errors.New("test error"))
			})
			It("should sets the active schedule and return an error", func() {
				Expect(err).To(HaveOccurred())

				schedule, exists := schedules.GetActiveSchedule("an-app-id")
				Expect(exists).To(BeTrue())
				Expect(schedule).To(Equal(activeSchedule))

				Eventually(buffer).Should(gbytes.Say("failed-set-active-schedule-set-instance-initial"))
				Eventually(buffer).Should(gbytes.Say("test error"))
			})
		})
	})

	Describe("RemoveActiveSchedule", func() {
		JustBeforeEach(func() {
			err = schedules.RemoveActiveSchedule("an-app-id", "a-schedule-id")
		})

		Context("when the active schedule exists", func() {
			BeforeEach(func() {
				err = schedules.SetActiveSchedule("an-app-id", activeSchedule)
				Expect(err).NotTo(HaveOccurred())
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{}, nil)
			})

			It("removes the active schedule", func() {
				Expect(err).NotTo(HaveOccurred())
				_, exists := schedules.GetActiveSchedule("an-app-id")
				Expect(exists).To(BeFalse())
			})
		})

		Context("when  active schedule does not exist", func() {
			BeforeEach(func() {
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{}, nil)
			})

			It("logs the information", func() {
				Expect(err).NotTo(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("remove-active-schedule"))
				Eventually(buffer).Should(gbytes.Say("active schedule does not exist"))
			})
		})

		Context("when active schedule id does not match", func() {
			BeforeEach(func() {
				err = schedules.SetActiveSchedule("an-app-id", ActiveSchedule{
					ScheduleId: "current-schedule-id",
				})
				Expect(err).NotTo(HaveOccurred())
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{}, nil)
			})

			It("logs the information and removes the app schedule", func() {
				Expect(err).NotTo(HaveOccurred())
				_, exists := schedules.GetActiveSchedule("an-app-id")
				Expect(exists).To(BeFalse())

				Eventually(buffer).Should(gbytes.Say("remove-active-schedule"))
				Eventually(buffer).Should(gbytes.Say("schedule id does not match"))
			})
		})

		Context("when current instance number is in the range of default policy", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(3, nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 5}, nil)
			})

			It("does not change the instance number", func() {
				Expect(cfc.SetAppInstancesCallCount()).To(Equal(0))
			})
		})

		Context("when current instance number is less than default instance-min-count", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(1, nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 3, InstanceMax: 6}, nil)
			})

			It("changes the instance number to instance-min-count", func() {
				appId, instances := cfc.SetAppInstancesArgsForCall(0)
				Expect(appId).To(Equal("an-app-id"))
				Expect(instances).To(Equal(3))
			})
		})

		Context("when current instance number is greater than default instance-max-count", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(8, nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 3, InstanceMax: 6}, nil)
			})

			It("changes the instance number to instance-max-count", func() {
				appId, instances := cfc.SetAppInstancesArgsForCall(0)
				Expect(appId).To(Equal("an-app-id"))
				Expect(instances).To(Equal(6))
			})
		})

		Context("when fails to get app policy", func() {
			BeforeEach(func() {
				policyDB.GetAppPolicyReturns(nil, errors.New("an error"))
			})

			It("should removes the schedule and error", func() {
				_, exists := schedules.GetActiveSchedule("an-app-id")
				Expect(exists).To(BeFalse())

				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-remove-active-schedule-get-app-policy"))
				Eventually(buffer).Should(gbytes.Say("an error"))

			})
		})

		Context("when fails to get current instance number", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(0, errors.New("an error"))
			})

			It("should removes the schedule and error", func() {
				_, exists := schedules.GetActiveSchedule("an-app-id")
				Expect(exists).To(BeFalse())

				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-remove-active-schedule-get-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))

			})
		})

		Context("when fails to set instance number to default instance limit", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(8, nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 3, InstanceMax: 6}, nil)
				cfc.SetAppInstancesReturns(errors.New("an error"))
			})

			It("should removes the schedule and error", func() {
				_, exists := schedules.GetActiveSchedule("an-app-id")
				Expect(exists).To(BeFalse())

				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-remove-active-schedule-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})
	})
})
