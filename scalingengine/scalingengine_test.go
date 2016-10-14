package scalingengine_test

import (
	"autoscaler/models"
	"autoscaler/scalingengine/fakes"
	"strconv"
	"time"

	. "autoscaler/scalingengine"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("ScalingEngine", func() {
	var (
		scalingEngine  ScalingEngine
		activeSchedule *ActiveSchedule
		cfc            *fakes.FakeCfClient
		policyDB       *fakes.FakePolicyDB
		historyDB      *fakes.FakeHistoryDB
		clock          *fakeclock.FakeClock

		newInstances int
		trigger      *models.Trigger
		buffer       *gbytes.Buffer
		err          error
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		policyDB = &fakes.FakePolicyDB{}
		historyDB = &fakes.FakeHistoryDB{}
		logger := lagertest.NewTestLogger("schedule-test")
		buffer = logger.Buffer()
		clock = fakeclock.NewFakeClock(time.Now())
		scalingEngine = NewScalingEngine(logger, cfc, policyDB, historyDB, clock)
		activeSchedule = &ActiveSchedule{
			InstanceMinInitial: 5,
			InstanceMin:        2,
			InstanceMax:        10,
		}
	})

	Describe("Scale", func() {
		BeforeEach(func() {
			trigger = &models.Trigger{
				MetricType:            models.MetricNameMemory,
				BreachDurationSeconds: 100,
				CoolDownSeconds:       30,
				Threshold:             222222,
				Operator:              ">",
				Adjustment:            "+1",
			}
		})

		JustBeforeEach(func() {
			newInstances, err = scalingEngine.Scale("an-app-id", trigger)
		})

		Context("when scaling succeeds", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(2, nil)
				historyDB.CanScaleAppReturns(true, nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)

			})

			It("sets the new app instance number and stores the succeeded scaling history", func() {
				Expect(err).NotTo(HaveOccurred())
				id, num := cfc.SetAppInstancesArgsForCall(0)
				Expect(id).To(Equal("an-app-id"))
				Expect(num).To(Equal(3))
				Expect(newInstances).To(Equal(3))

				id, expiredAt := historyDB.UpdateScalingCooldownExpireTimeArgsForCall(0)
				Expect(id).To(Equal("an-app-id"))
				Expect(expiredAt).To(Equal(clock.Now().Add(30 * time.Second).UnixNano()))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 2,
					NewInstances: 3,
					Reason:       "+1 instance(s) because memorybytes > 222222 for 100 seconds",
				}))

			})
		})

		Context("when app is in cooldown period", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(2, nil)
				historyDB.CanScaleAppReturns(false, nil)
			})

			It("ignores the scaling", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfc.SetAppInstancesCallCount()).To(BeZero())

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusIgnored,
					OldInstances: 2,
					NewInstances: 2,
					Reason:       "+1 instance(s) because memorybytes > 222222 for 100 seconds",
					Message:      "app in cooldown period",
				}))

			})
		})

		Context("when app instances not changed", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+20%"
				cfc.GetAppInstancesReturns(2, nil)
				historyDB.CanScaleAppReturns(true, nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)

			})

			It("does not update the app and stores the ignored scaling history", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfc.SetAppInstancesCallCount()).To(BeZero())
				Expect(newInstances).To(Equal(2))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusIgnored,
					OldInstances: 2,
					NewInstances: 2,
					Reason:       "+20% instance(s) because memorybytes > 222222 for 100 seconds",
				}))

			})
		})

		Context("when it exceeds max instances limit", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+2"
				cfc.GetAppInstancesReturns(5, nil)
				historyDB.CanScaleAppReturns(true, nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)

			})

			It("does upate the app instance with max instances and stores the succeeded scaling history", func() {
				Expect(err).NotTo(HaveOccurred())

				id, num := cfc.SetAppInstancesArgsForCall(0)
				Expect(id).To(Equal("an-app-id"))
				Expect(num).To(Equal(6))
				Expect(newInstances).To(Equal(6))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 5,
					NewInstances: 6,
					Reason:       "+2 instance(s) because memorybytes > 222222 for 100 seconds",
					Message:      "limited by max instances 6",
				}))

			})
		})

		Context("when it exceeds min instances limit", func() {
			BeforeEach(func() {
				trigger.Adjustment = "-60%"
				cfc.GetAppInstancesReturns(3, nil)
				historyDB.CanScaleAppReturns(true, nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 2, InstanceMax: 6}, nil)

			})

			It("does upate the app instance with max instances and stores the succeeded scaling history", func() {
				Expect(err).NotTo(HaveOccurred())

				id, num := cfc.SetAppInstancesArgsForCall(0)
				Expect(id).To(Equal("an-app-id"))
				Expect(num).To(Equal(2))
				Expect(newInstances).To(Equal(2))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 3,
					NewInstances: 2,
					Reason:       "-60% instance(s) because memorybytes > 222222 for 100 seconds",
					Message:      "limited by min instances 2",
				}))

			})
		})

		Context("when getting app instances from cloud controller fails", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(-1, errors.New("test error"))
			})

			It("should error and store the failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-app-instances"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: -1,
					NewInstances: -1,
					Reason:       "+1 instance(s) because memorybytes > 222222 for 100 seconds",
					Error:        "failed to get app instances",
				}))

			})
		})

		Context("When checking cooldown fails", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(2, nil)
				historyDB.CanScaleAppReturns(false, errors.New("test error"))
			})
			It("should error and store the failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-check-cooldown"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: -1,
					Reason:       "+1 instance(s) because memorybytes > 222222 for 100 seconds",
					Error:        "failed to check app cooldown setting",
				}))

			})
		})

		Context("when computing new app instances fails", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+a"
				cfc.GetAppInstancesReturns(2, nil)
				historyDB.CanScaleAppReturns(true, nil)
			})

			It("should error and store failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-compute-new-instance"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: -1,
					Reason:       "+a instance(s) because memorybytes > 222222 for 100 seconds",
					Error:        "failed to compute new app instances",
				}))
			})
		})

		Context("when getting policy fails", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(2, nil)
				historyDB.CanScaleAppReturns(true, nil)
				policyDB.GetAppPolicyReturns(nil, errors.New("test error"))
			})

			It("should error and store the failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-app-policy"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: -1,
					Reason:       "+1 instance(s) because memorybytes > 222222 for 100 seconds",
					Error:        "failed to get scaling policy",
				}))

			})
		})

		Context("when set new instances fails", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(2, nil)
				historyDB.CanScaleAppReturns(true, nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
				cfc.SetAppInstancesReturns(errors.New("test error"))
			})

			It("should error and store failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: 3,
					Reason:       "+1 instance(s) because memorybytes > 222222 for 100 seconds",
					Error:        "failed to set app instances",
				}))

			})
		})
	})

	Describe("ComputeNewInstances", func() {
		var adjustment string

		BeforeEach(func() {
			adjustment = ""
		})

		JustBeforeEach(func() {
			instances := 3
			newInstances, err = scalingEngine.ComputeNewInstances(instances, adjustment)
		})

		Context("when adjustment is not valid", func() {
			Context("when adjustment is not a valid percentage", func() {
				BeforeEach(func() {
					adjustment = "10.5a%"
				})

				It("should error", func() {
					Expect(err).To(BeAssignableToTypeOf(&strconv.NumError{}))
				})
			})
			Context("when adjustment is not a valid step", func() {
				BeforeEach(func() {
					adjustment = "#1"
				})

				It("should error", func() {
					Expect(err).To(BeAssignableToTypeOf(&strconv.NumError{}))
				})
			})
		})

		Context("when adjustment is valid", func() {
			Context("when adjustment is by percentage", func() {
				BeforeEach(func() {
					adjustment = "50%"
				})
				It("returns correct new instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(newInstances).To(Equal(5))
				})
			})

			Context("when adjustment is by step", func() {
				BeforeEach(func() {
					adjustment = "-2"
				})
				It("returns correct new instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(newInstances).To(Equal(1))
				})
			})
		})
	})

	Describe("SetActiveSchedule", func() {
		JustBeforeEach(func() {
			err = scalingEngine.SetActiveSchedule("an-app-id", activeSchedule)
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
				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 12,
					NewInstances: 10,
					Reason:       "schedule starts with instance min 2, instance max 10 and instance min initial 5",
					Message:      "limited by max instances 5",
				}))

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

					Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    clock.Now().UnixNano(),
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusSucceeded,
						OldInstances: 1,
						NewInstances: 2,
						Reason:       "schedule starts with instance min 2, instance max 10 and instance min initial 0",
						Message:      "limited by min instances 2",
					}))

				})
			})

			Context("when app instance number is in the range of [InstanceMin, InstanceMax] in active schedule", func() {
				BeforeEach(func() {
					cfc.GetAppInstancesReturns(3, nil)
				})
				It("does not change the instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(cfc.SetAppInstancesCallCount()).To(BeZero())

					Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    clock.Now().UnixNano(),
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusIgnored,
						OldInstances: 3,
						NewInstances: 3,
						Reason:       "schedule starts with instance min 2, instance max 10 and instance min initial 0",
					}))
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

					Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    clock.Now().UnixNano(),
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusSucceeded,
						OldInstances: 3,
						NewInstances: 5,
						Reason:       "schedule starts with instance min 2, instance max 10 and instance min initial 5",
						Message:      "limited by min instances 5",
					}))

				})
			})

			Context("when app instance number is in the range of [InstanceMinInitial, InstanceMax] in active schedule", func() {
				BeforeEach(func() {
					cfc.GetAppInstancesReturns(6, nil)
				})
				It("does not change the instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(cfc.SetAppInstancesCallCount()).To(BeZero())

					Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    clock.Now().UnixNano(),
						ScalingType:  models.ScalingTypeSchedule,
						Status:       models.ScalingStatusIgnored,
						OldInstances: 6,
						NewInstances: 6,
						Reason:       "schedule starts with instance min 2, instance max 10 and instance min initial 5",
					}))

				})
			})
		})

		Context("when getting app instances fails", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(0, errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: -1,
					NewInstances: -1,
					Reason:       "schedule starts with instance min 2, instance max 10 and instance min initial 5",
					Error:        "failed to get app instances",
				}))

			})
		})

		Context("when setting app instances fails", func() {
			BeforeEach(func() {
				cfc.SetAppInstancesReturns(errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: 0,
					NewInstances: 5,
					Reason:       "schedule starts with instance min 2, instance max 10 and instance min initial 5",
					Message:      "limited by min instances 5",
					Error:        "failed to set app instances",
				}))

			})
		})

	})

	Describe("RemoveActiveSchedule", func() {
		JustBeforeEach(func() {
			err = scalingEngine.RemoveActiveSchedule("an-app-id", "a-schedule-id")
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
				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusIgnored,
					OldInstances: 5,
					NewInstances: 5,
					Reason:       "schedule ends",
				}))
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
				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 1,
					NewInstances: 3,
					Reason:       "schedule ends",
					Message:      "limited by min instances 3",
				}))

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

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 8,
					NewInstances: 6,
					Reason:       "schedule ends",
					Message:      "limited by max instances 6",
				}))

			})
		})

		Context("when getting app instances fails", func() {
			BeforeEach(func() {
				cfc.GetAppInstancesReturns(0, errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: -1,
					NewInstances: -1,
					Reason:       "schedule ends",
					Error:        "failed to get app instances",
				}))

			})
		})

		Context("when getting app policy fails", func() {
			BeforeEach(func() {
				policyDB.GetAppPolicyReturns(nil, errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-app-policy"))
				Eventually(buffer).Should(gbytes.Say("an error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: 0,
					NewInstances: -1,
					Reason:       "schedule ends",
					Error:        "failed to get app policy",
				}))

			})
		})

		Context("when setting instance number fails", func() {
			BeforeEach(func() {
				cfc.SetAppInstancesReturns(errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))

				Expect(historyDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: 0,
					NewInstances: 3,
					Reason:       "schedule ends",
					Error:        "failed to set app instances",
					Message:      "limited by min instances 3",
				}))

			})
		})

	})
})
