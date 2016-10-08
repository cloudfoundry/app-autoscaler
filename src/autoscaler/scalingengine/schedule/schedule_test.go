package schedule_test

import (
	"autoscaler/models"
	"autoscaler/scalingengine/fakes"

	. "autoscaler/scalingengine/schedule"

	"code.cloudfoundry.org/lager/lagertest"

	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Schedule", func() {

	var (
		schedules      *AppSchedules
		activeSchedule *ActiveSchedule
		cfc            *fakes.FakeCfClient
		policyDB       *fakes.FakePolicyDB
		buffer         *gbytes.Buffer
		err            error
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		policyDB = &fakes.FakePolicyDB{}
		logger := lagertest.NewTestLogger("schedule-test")
		buffer = logger.Buffer()
		schedules = NewAppSchedules(logger, cfc, policyDB)
		activeSchedule = &ActiveSchedule{
			InstanceMinInitial: 5,
			InstanceMin:        2,
			InstanceMax:        10,
		}
	})

	Describe("SetActiveSchedule", func() {
		JustBeforeEach(func() {
			err = schedules.SetActiveSchedule("an-app-id", activeSchedule)
		})

		Context("when app instance number is greater than InstanceMax in active schedule", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(12, nil)
			})

			It("sets the app instances to be InstanceMax", func() {
				Expect(err).NotTo(HaveOccurred())

				appid, instances := cfc.SetAppInstancesArgsForCall(0)
				Expect(appid).To(Equal("an-app-id"))
				Expect(instances).To(Equal(10))
			})
		})

		Context("when initial min instance is zero (not set)", func() {
			BeforeEach(func() {
				activeSchedule.InstanceMinInitial = 0
			})

			Context("when app instance number is below the InstanceMin in active schedule", func() {
				BeforeEach(func() {
					cfc.GetAppInstancesReturns(1, nil)
				})

				It("sets the app instances to be InstanceMin", func() {
					Expect(err).NotTo(HaveOccurred())

					appid, instances := cfc.SetAppInstancesArgsForCall(0)
					Expect(appid).To(Equal("an-app-id"))
					Expect(instances).To(Equal(2))
				})
			})

			Context("when app instance number is in the range of [InstanceMin, InstanceMax] in active schedule", func() {
				BeforeEach(func() {
					cfc.GetAppInstancesReturns(3, nil)
				})
				It("does not change the instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(cfc.SetAppInstancesCallCount()).To(BeZero())
				})
			})
		})

		Context("when initial min instance is set", func() {
			Context("when app instance number is below the InstanceMinInitial in active schedule", func() {
				BeforeEach(func() {
					cfc.GetAppInstancesReturns(3, nil)
				})

				It("sets the app instances to be InstanceMinInitial", func() {
					Expect(err).NotTo(HaveOccurred())

					appid, instances := cfc.SetAppInstancesArgsForCall(0)
					Expect(appid).To(Equal("an-app-id"))
					Expect(instances).To(Equal(5))
				})
			})

			Context("when app instance number is in the range of [InstanceMinInitial, InstanceMax] in active schedule", func() {
				BeforeEach(func() {
					cfc.GetAppInstancesReturns(6, nil)
				})
				It("does not change the instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(cfc.SetAppInstancesCallCount()).To(BeZero())
				})
			})
		})

		Context("when getting app instances fails", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(0, errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-set-active-schedule-get-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})

		Context("when setting app instances fails", func() {
			BeforeEach(func() {
				cfc.SetAppInstancesReturns(errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-set-active-schedule-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})

	})

	Describe("RemoveActiveSchedule", func() {
		JustBeforeEach(func() {
			err = schedules.RemoveActiveSchedule("an-app-id", "a-schedule-id")
		})

		BeforeEach(func() {
			policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 3, InstanceMax: 6}, nil)
		})

		Context("when app instance number is in the default range [InstanceMin, InstianceMax] in the policy", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(5, nil)
			})

			It("does not change the instance number", func() {
				Expect(cfc.SetAppInstancesCallCount()).To(Equal(0))
			})
		})

		Context("when app instance number is below the default InstanceMin in the policy", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(1, nil)
			})

			It("changes the instance number to InstanceMin", func() {
				appId, instances := cfc.SetAppInstancesArgsForCall(0)
				Expect(appId).To(Equal("an-app-id"))
				Expect(instances).To(Equal(3))
			})
		})

		Context("when app instance number is greater than the default InstanaceMax in the policy", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(8, nil)
			})

			It("changes the instance number to instance-max-count", func() {
				appId, instances := cfc.SetAppInstancesArgsForCall(0)
				Expect(appId).To(Equal("an-app-id"))
				Expect(instances).To(Equal(6))
			})
		})

		Context("when getting app instances fails", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(0, errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-remove-active-schedule-get-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})

		Context("when getting app policy fails", func() {
			BeforeEach(func() {
				policyDB.GetAppPolicyReturns(nil, errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-remove-active-schedule-get-app-policy"))
				Eventually(buffer).Should(gbytes.Say("an error"))

			})
		})

		Context("when setting instance number fails", func() {
			BeforeEach(func() {
				cfc.SetAppInstancesReturns(errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-remove-active-schedule-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})

	})
})
