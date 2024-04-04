package scalingengine_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"

	"errors"
	"strconv"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("ScalingEngine", func() {
	var (
		scalingEngine   ScalingEngine
		activeSchedule  *models.ActiveSchedule
		cfc             *fakes.FakeCFClient
		policyDB        *fakes.FakePolicyDB
		scalingEngineDB *fakes.FakeScalingEngineDB
		clock           *fakeclock.FakeClock

		scalingResult *models.AppScalingResult
		appState      string
		trigger       *models.Trigger
		buffer        *gbytes.Buffer
		err           error
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCFClient{}
		policyDB = &fakes.FakePolicyDB{}
		scalingEngineDB = &fakes.FakeScalingEngineDB{}

		logger := lagertest.NewTestLogger("schedule-test")
		buffer = logger.Buffer()
		clock = fakeclock.NewFakeClock(time.Now())
		scalingEngine = NewScalingEngine(logger, cfc, policyDB, scalingEngineDB, clock, 300, 32)
		appState = models.AppStatusStarted
		activeSchedule = &models.ActiveSchedule{
			ScheduleId:         "a-schedule-id",
			InstanceMinInitial: 5,
			InstanceMin:        2,
			InstanceMax:        10,
		}
	})

	setAppAndProcesses := func(instances int, aState string) {
		cfc.GetAppAndProcessesReturns(&cf.AppAndProcesses{Processes: cf.Processes{{Instances: instances}}, App: &cf.App{State: aState}}, nil)
	}

	Describe("Scale", func() {
		BeforeEach(func() {
			trigger = &models.Trigger{
				MetricType:            "test-metric-type",
				MetricUnit:            "test-unit",
				BreachDurationSeconds: 100,
				CoolDownSeconds:       30,
				Threshold:             80,
				Operator:              ">",
				Adjustment:            "+1",
			}
		})

		JustBeforeEach(func() {
			scalingResult, err = scalingEngine.Scale("an-app-id", trigger)
		})

		Context("when scaling succeeds", func() {
			BeforeEach(func() {
				setAppAndProcesses(2, appState)
				scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)

			})

			It("sets the new app instance number and stores the succeeded scaling history", func() {
				Expect(err).NotTo(HaveOccurred())
				guid, num := cfc.ScaleAppWebProcessArgsForCall(0)
				Expect(guid.String()).To(Equal("an-app-id"))
				Expect(num).To(Equal(3))

				id, expiredAt := scalingEngineDB.UpdateScalingCooldownExpireTimeArgsForCall(0)
				Expect(id).To(Equal("an-app-id"))
				Expect(expiredAt).To(Equal(clock.Now().Add(30 * time.Second).UnixNano()))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 2,
					NewInstances: 3,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
				}))

				Expect(scalingResult.AppId).To(Equal("an-app-id"))
				Expect(scalingResult.Status).To(Equal(models.ScalingStatusSucceeded))
				Expect(scalingResult.Adjustment).To(Equal(1))
				Expect(scalingResult.CooldownExpiredAt).To(Equal(clock.Now().Add(30 * time.Second).UnixNano()))
			})
		})

		Context("When app is not started", func() {
			BeforeEach(func() {
				setAppAndProcesses(2, "test-state")
			})
			It("ignore the scaling and store the ignored scaling history", func() {
				Eventually(buffer).Should(gbytes.Say("check-app-state"))
				Eventually(buffer).Should(gbytes.Say("ignore scaling since app is not started"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusIgnored,
					OldInstances: 2,
					NewInstances: 2,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Message:      "app is not started",
				}))

				Expect(scalingResult.AppId).To(Equal("an-app-id"))
				Expect(scalingResult.Status).To(Equal(models.ScalingStatusIgnored))
				Expect(scalingResult.Adjustment).To(Equal(0))
				Expect(scalingResult.CooldownExpiredAt).To(Equal(int64(0)))
			})
		})

		Context("When app is labeled with app-autoscaler.cloudfoundry.org/disable-autoscaling", func() {
			BeforeEach(func() {
				labelContent := "for test purposes"
				cfc.GetAppAndProcessesReturns(&cf.AppAndProcesses{Processes: cf.Processes{{Instances: 2}}, App: &cf.App{State: appState, Metadata: cf.Metadata{Labels: cf.Labels{DisableAutoscaling: &labelContent}}}}, nil)
			})
			It("ignore the scaling and store the ignored scaling history", func() {
				Eventually(buffer).Should(gbytes.Say("check-app-label"))
				Eventually(buffer).Should(gbytes.Say("ignore scaling since app has the label app-autoscaler.cloudfoundry.org/disable-autoscaling set"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusIgnored,
					OldInstances: 2,
					NewInstances: 2,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Message:      "The application was not scaled as the label \"app-autoscaler.cloudfoundry.org/disable-autoscaling\" was set on the app. The content of the label might give a hint on why the label was set: \"for test purposes\"",
				}))

				Expect(scalingResult.AppId).To(Equal("an-app-id"))
				Expect(scalingResult.Status).To(Equal(models.ScalingStatusIgnored))
				Expect(scalingResult.Adjustment).To(Equal(0))
				Expect(scalingResult.CooldownExpiredAt).To(Equal(int64(0)))
			})
		})

		Context("when app is in cooldown period", func() {
			BeforeEach(func() {
				setAppAndProcesses(2, appState)
				scalingEngineDB.CanScaleAppReturns(false, clock.Now().Add(30*time.Second).UnixNano(), nil)
			})

			It("ignores the scaling", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusIgnored,
					OldInstances: 2,
					NewInstances: 2,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Message:      "app in cooldown period",
				}))

				Expect(scalingResult.AppId).To(Equal("an-app-id"))
				Expect(scalingResult.Status).To(Equal(models.ScalingStatusIgnored))
				Expect(scalingResult.Adjustment).To(Equal(0))
				Expect(scalingResult.CooldownExpiredAt).To(Equal(clock.Now().Add(30 * time.Second).UnixNano()))

			})
		})

		Context("when app instances not changed", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+1"
				setAppAndProcesses(6, appState)
				scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 2, InstanceMax: 6}, nil)

			})

			It("does not update the app and stores the ignored scaling history", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusIgnored,
					OldInstances: 6,
					NewInstances: 6,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Message:      "limited by max instances 6",
				}))

				Expect(scalingResult.AppId).To(Equal("an-app-id"))
				Expect(scalingResult.Status).To(Equal(models.ScalingStatusIgnored))
				Expect(scalingResult.Adjustment).To(Equal(0))
				Expect(scalingResult.CooldownExpiredAt).To(Equal(int64(0)))
			})
		})

		Context("when it exceeds max instances limit in scaling policy", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+2"
				setAppAndProcesses(5, appState)
				scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
			})

			It("updates the app instance with  max instances and stores the succeeded scaling history", func() {
				Expect(err).NotTo(HaveOccurred())

				guid, num := cfc.ScaleAppWebProcessArgsForCall(0)
				Expect(guid.String()).To(Equal("an-app-id"))
				Expect(num).To(Equal(6))

				id, expiredAt := scalingEngineDB.UpdateScalingCooldownExpireTimeArgsForCall(0)
				Expect(id).To(Equal("an-app-id"))
				Expect(expiredAt).To(Equal(clock.Now().Add(30 * time.Second).UnixNano()))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 5,
					NewInstances: 6,
					Reason:       "+2 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Message:      "limited by max instances 6",
				}))

				Expect(scalingResult.AppId).To(Equal("an-app-id"))
				Expect(scalingResult.Status).To(Equal(models.ScalingStatusSucceeded))
				Expect(scalingResult.Adjustment).To(Equal(1))
				Expect(scalingResult.CooldownExpiredAt).To(Equal(clock.Now().Add(30 * time.Second).UnixNano()))
			})
		})

		Context("when current instance equals to the max instances limit in scaling policy", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+2"
				setAppAndProcesses(6, appState)
				scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
			})

			It("updates the app instance with  max instances and stores the ignored scaling history", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())
				Expect(scalingEngineDB.UpdateScalingCooldownExpireTimeCallCount()).To(BeZero())

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusIgnored,
					OldInstances: 6,
					NewInstances: 6,
					Reason:       "+2 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Message:      "limited by max instances 6",
				}))

				Expect(scalingResult.AppId).To(Equal("an-app-id"))
				Expect(scalingResult.Status).To(Equal(models.ScalingStatusIgnored))
				Expect(scalingResult.Adjustment).To(Equal(0))
				Expect(scalingResult.CooldownExpiredAt).To(Equal(int64(0)))
			})
		})

		Context("when it exceeds min instances limit in scaling policy", func() {
			BeforeEach(func() {
				trigger.Adjustment = "-60%"
				setAppAndProcesses(3, appState)
				scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 2, InstanceMax: 6}, nil)

			})

			It("updates the app instance with  min instances and stores the succeeded scaling history", func() {
				Expect(err).NotTo(HaveOccurred())

				guid, num := cfc.ScaleAppWebProcessArgsForCall(0)
				Expect(guid.String()).To(Equal("an-app-id"))
				Expect(num).To(Equal(2))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 3,
					NewInstances: 2,
					Reason:       "-60% instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Message:      "limited by min instances 2",
				}))

				Expect(scalingResult.AppId).To(Equal("an-app-id"))
				Expect(scalingResult.Status).To(Equal(models.ScalingStatusSucceeded))
				Expect(scalingResult.Adjustment).To(Equal(-1))
				Expect(scalingResult.CooldownExpiredAt).To(Equal(clock.Now().Add(30 * time.Second).UnixNano()))
			})
		})

		Context("when there is active schedule", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{
					ScheduleId:  "111111",
					InstanceMin: 3,
					InstanceMax: 7,
				}, nil)
			})

			Context("when it exceeds max instances limit in active schedule", func() {
				BeforeEach(func() {
					trigger.Adjustment = "+2"
					setAppAndProcesses(6, appState)
					scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				})

				It("updates the app instance with  max instances and stores the succeeded scaling history", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(policyDB.GetAppPolicyCallCount()).To(BeZero())

					id, num := cfc.ScaleAppWebProcessArgsForCall(0)
					Expect(id.String()).To(Equal("an-app-id"))
					Expect(num).To(Equal(7))

					Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    clock.Now().UnixNano(),
						ScalingType:  models.ScalingTypeDynamic,
						Status:       models.ScalingStatusSucceeded,
						OldInstances: 6,
						NewInstances: 7,
						Reason:       "+2 instance(s) because test-metric-type > 80test-unit for 100 seconds",
						Message:      "limited by max instances 7",
					}))

					Expect(scalingResult.AppId).To(Equal("an-app-id"))
					Expect(scalingResult.Status).To(Equal(models.ScalingStatusSucceeded))
					Expect(scalingResult.Adjustment).To(Equal(1))
					Expect(scalingResult.CooldownExpiredAt).To(Equal(clock.Now().Add(30 * time.Second).UnixNano()))
				})
			})

			Context("when it exceeds min instances limit in active schedule", func() {
				BeforeEach(func() {
					trigger.Adjustment = "-60%"
					setAppAndProcesses(5, appState)
					scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				})

				It("updates the app instance with min instances and stores the succeeded scaling history", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(policyDB.GetAppPolicyCallCount()).To(BeZero())

					id, num := cfc.ScaleAppWebProcessArgsForCall(0)
					Expect(id.String()).To(Equal("an-app-id"))
					Expect(num).To(Equal(3))

					Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
						AppId:        "an-app-id",
						Timestamp:    clock.Now().UnixNano(),
						ScalingType:  models.ScalingTypeDynamic,
						Status:       models.ScalingStatusSucceeded,
						OldInstances: 5,
						NewInstances: 3,
						Reason:       "-60% instance(s) because test-metric-type > 80test-unit for 100 seconds",
						Message:      "limited by min instances 3",
					}))

					Expect(scalingResult.AppId).To(Equal("an-app-id"))
					Expect(scalingResult.Status).To(Equal(models.ScalingStatusSucceeded))
					Expect(scalingResult.Adjustment).To(Equal(-2))
					Expect(scalingResult.CooldownExpiredAt).To(Equal(clock.Now().Add(30 * time.Second).UnixNano()))
				})
			})
		})

		Context("when getting app info from cloud foundry fails", func() {
			BeforeEach(func() {
				cfc.GetAppAndProcessesReturns(nil, errors.New("test error"))
			})

			It("should error and store the failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-app-info"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: -1,
					NewInstances: -1,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Error:        "failed to get app info: test error",
				}))

				Expect(scalingResult).To(BeNil())

			})
		})

		Context("When checking cooldown fails", func() {
			BeforeEach(func() {
				setAppAndProcesses(2, appState)
				scalingEngineDB.CanScaleAppReturns(false, 0, errors.New("test error"))
			})
			It("should error and store the failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-check-cooldown"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: -1,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Error:        "failed to check app cooldown setting",
				}))

				Expect(scalingResult).To(BeNil())
			})
		})

		Context("when computing new app instances fails", func() {
			BeforeEach(func() {
				trigger.Adjustment = "+a"
				setAppAndProcesses(2, appState)
				scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)

			})

			It("should error and store failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-compute-new-instance"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: -1,
					Reason:       "+a instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Error:        "failed to compute new app instances",
				}))

				Expect(scalingResult).To(BeNil())

			})
		})

		Context("when getting active schedule fails", func() {
			BeforeEach(func() {
				setAppAndProcesses(2, appState)
				scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				scalingEngineDB.GetActiveScheduleReturns(nil, errors.New("test error"))
			})

			It("should error and store the failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-active-schedule"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: -1,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Error:        "failed to get active schedule",
				}))

				Expect(scalingResult).To(BeNil())

			})
		})

		Context("when getting policy fails", func() {
			BeforeEach(func() {
				setAppAndProcesses(2, appState)
				scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				policyDB.GetAppPolicyReturns(nil, errors.New("test error"))
			})

			It("should error and store the failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-app-policy"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: -1,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Error:        "failed to get scaling policy",
				}))

				Expect(scalingResult).To(BeNil())

			})
		})

		Context("when app does not have policy set", func() {
			BeforeEach(func() {
				setAppAndProcesses(2, appState)
				scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				policyDB.GetAppPolicyReturns(nil, nil)
			})

			It("does not update the app and stores the ignored scaling history", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusIgnored,
					OldInstances: 2,
					NewInstances: 2,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Message:      "app does not have policy set",
				}))

				Expect(scalingResult.AppId).To(Equal("an-app-id"))
				Expect(scalingResult.Status).To(Equal(models.ScalingStatusIgnored))
				Expect(scalingResult.Adjustment).To(Equal(0))
				Expect(scalingResult.CooldownExpiredAt).To(Equal(int64(0)))
			})
		})

		Context("when set new instances fails", func() {
			BeforeEach(func() {
				setAppAndProcesses(2, appState)
				scalingEngineDB.CanScaleAppReturns(true, clock.Now().Add(0-30*time.Second).UnixNano(), nil)
				policyDB.GetAppPolicyReturns(&models.ScalingPolicy{InstanceMin: 1, InstanceMax: 6}, nil)
				cfc.ScaleAppWebProcessReturns(errors.New("test error"))
			})

			It("should error and store failed scaling history", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("test error"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: 3,
					Reason:       "+1 instance(s) because test-metric-type > 80test-unit for 100 seconds",
					Error:        "failed to set app instances: test error",
				}))

				Expect(scalingResult).To(BeNil())

			})
		})
	})

	Describe("ComputeNewInstances", func() {
		var adjustment string
		var newInstances int

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
			Context("when adjustment is defined as a positive percentage value", func() {
				BeforeEach(func() {
					adjustment = "10%"
				})
				It("returns correct new instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(newInstances).To(Equal(4))
				})
			})

			Context("when adjustment is defined as a negative  percentage value", func() {
				BeforeEach(func() {
					adjustment = "-10%"
				})
				It("returns correct new instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(newInstances).To(Equal(2))
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

		BeforeEach(func() {
			cfc.GetAppProcessesReturns(cf.Processes{}, nil)
		})

		It("Saves the active schedule to database", func() {
			Expect(err).NotTo(HaveOccurred())
			id, schedule := scalingEngineDB.SetActiveScheduleArgsForCall(0)
			Expect(id).To(Equal("an-app-id"))
			Expect(schedule).To(Equal(activeSchedule))
		})

		Context("when app instance number is greater than InstanceMax in active schedule", func() {
			BeforeEach(func() {
				cfc.GetAppProcessesReturns(cf.Processes{{Instances: 12}}, nil)
			})

			It("sets the app instances to be InstanceMax", func() {
				Expect(err).NotTo(HaveOccurred())

				appid, instances := cfc.ScaleAppWebProcessArgsForCall(0)
				Expect(appid.String()).To(Equal("an-app-id"))
				Expect(instances).To(Equal(10))
				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusSucceeded,
					OldInstances: 12,
					NewInstances: 10,
					Reason:       "schedule starts with instance min 2, instance max 10 and instance min initial 5",
					Message:      "limited by max instances 10",
				}))

			})
		})

		Context("when initial min instance is zero (not set)", func() {
			BeforeEach(func() {
				activeSchedule.InstanceMinInitial = 0
			})

			Context("when app instance number is below the InstanceMin in active schedule", func() {
				BeforeEach(func() {
					cfc.GetAppProcessesReturns(cf.Processes{{Instances: 1}}, nil)
				})

				It("sets the app instances to be InstanceMin", func() {
					Expect(err).NotTo(HaveOccurred())

					appid, instances := cfc.ScaleAppWebProcessArgsForCall(0)
					Expect(appid.String()).To(Equal("an-app-id"))
					Expect(instances).To(Equal(2))

					Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
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
					cfc.GetAppProcessesReturns(cf.Processes{{Instances: 3}}, nil)
				})
				It("does not change the instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())

					Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
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
					cfc.GetAppProcessesReturns(cf.Processes{{Instances: 3}}, nil)
				})

				It("sets the app instances to be InstanceMinInitial", func() {
					Expect(err).NotTo(HaveOccurred())

					appid, instances := cfc.ScaleAppWebProcessArgsForCall(0)
					Expect(appid.String()).To(Equal("an-app-id"))
					Expect(instances).To(Equal(5))

					Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
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
					cfc.GetAppProcessesReturns(cf.Processes{{Instances: 6}}, nil)
				})
				It("does not change the instance number", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())

					Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
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

		Context("when active schedule exists", func() {
			Context("when it is the same active schedule", func() {
				BeforeEach(func() {
					scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-schedule-id"}, nil)
				})

				It("igore the duplicatd request", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(scalingEngineDB.SetActiveScheduleCallCount()).To(BeZero())
					Eventually(buffer).Should(gbytes.Say("set-active-schedule"))
					Eventually(buffer).Should(gbytes.Say("duplicate request to set active schedule"))
				})
			})

			Context("when it is a different active schedule", func() {
				BeforeEach(func() {
					scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-different-schedule-id"}, nil)
				})

				It("rewrites the current active schedule", func() {
					Expect(err).NotTo(HaveOccurred())
					id, schedule := scalingEngineDB.SetActiveScheduleArgsForCall(0)
					Expect(id).To(Equal("an-app-id"))
					Expect(schedule).To(Equal(activeSchedule))
					Eventually(buffer).Should(gbytes.Say("set-active-schedule"))
					Eventually(buffer).Should(gbytes.Say("an active schedule exists in database"))
				})
			})

		})

		Context("when getting current active schedule fails", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(nil, errors.New("an error"))
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-existing-active-schedule-from-database"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})

		Context("when setting active schedule in database fails", func() {
			BeforeEach(func() {
				scalingEngineDB.SetActiveScheduleReturns(errors.New("an error"))
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-set-active-schedule-in-database"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})

		Context("when getting app info from cloud foundry fails", func() {
			BeforeEach(func() {
				cfc.GetAppProcessesReturns(nil, errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-app-info"))
				Eventually(buffer).Should(gbytes.Say("an error"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: -1,
					NewInstances: -1,
					Reason:       "schedule starts with instance min 2, instance max 10 and instance min initial 5",
					Error:        "failed to get app info: an error",
				}))

			})
		})

		Context("when setting app instances fails", func() {
			BeforeEach(func() {
				cfc.ScaleAppWebProcessReturns(errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: 0,
					NewInstances: 5,
					Reason:       "schedule starts with instance min 2, instance max 10 and instance min initial 5",
					Message:      "limited by min instances 5",
					Error:        "failed to set app instances: an error",
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
				scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-schedule-id"}, nil)
				cfc.GetAppProcessesReturns(cf.Processes{{Instances: 5}}, nil)
			})

			It("does not change the instance number", func() {
				Expect(cfc.ScaleAppWebProcessCallCount()).To(Equal(0))
				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
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
				scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-schedule-id"}, nil)
				cfc.GetAppProcessesReturns(cf.Processes{{Instances: 1}}, nil)
			})

			It("changes the instance number to InstanceMin", func() {
				appId, instances := cfc.ScaleAppWebProcessArgsForCall(0)
				Expect(appId.String()).To(Equal("an-app-id"))
				Expect(instances).To(Equal(3))
				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
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
				scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-schedule-id"}, nil)
				cfc.GetAppProcessesReturns(cf.Processes{{Instances: 8}}, nil)
			})

			It("changes the instance number to instance-max-count", func() {
				appId, instances := cfc.ScaleAppWebProcessArgsForCall(0)
				Expect(appId.String()).To(Equal("an-app-id"))
				Expect(instances).To(Equal(6))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
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

		Context("when active schedule does not exist", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(nil, nil)
			})
			It("should not have any error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(scalingEngineDB.RemoveActiveScheduleCallCount()).To(BeZero())
			})
		})

		Context("when there is a different active schedule", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-different-schedule-id"}, nil)
			})
			It("should not have any error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(scalingEngineDB.RemoveActiveScheduleCallCount()).To(BeZero())
			})
		})

		Context("when getting active schedule fails", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(nil, errors.New("an error"))
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-existing-active-schedule-from-database"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})
		})

		Context("when removing active schedule from database fails", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-schedule-id"}, nil)
				scalingEngineDB.RemoveActiveScheduleReturns(errors.New("an error"))
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-remove-active-schedule-from-database"))
				Eventually(buffer).Should(gbytes.Say("an error"))
			})

		})

		Context("when getting app info from cloud foundry fails", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-schedule-id"}, nil)
				cfc.GetAppProcessesReturns(nil, errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: -1,
					NewInstances: -1,
					Reason:       "schedule ends",
					Error:        "failed to get app info: an error",
				}))

			})
		})

		Context("when getting app policy fails", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-schedule-id"}, nil)
				cfc.GetAppProcessesReturns(cf.Processes{{Instances: 2}}, nil)
				policyDB.GetAppPolicyReturns(nil, errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-get-app-policy"))
				Eventually(buffer).Should(gbytes.Say("an error"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: -1,
					Reason:       "schedule ends",
					Error:        "failed to get app policy",
				}))

			})
		})

		Context("when no policy for app in policy db", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-schedule-id"}, nil)
				cfc.GetAppProcessesReturns(cf.Processes{{Instances: 2}}, nil)
				policyDB.GetAppPolicyReturns(nil, nil)
			})

			It("should not have any error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())
				Expect(scalingEngineDB.RemoveActiveScheduleCallCount()).To(Equal(1))
			})
		})

		Context("when setting instance number fails", func() {
			BeforeEach(func() {
				scalingEngineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "a-schedule-id"}, nil)
				cfc.GetAppProcessesReturns(cf.Processes{{Instances: 2}}, nil)
				cfc.ScaleAppWebProcessReturns(errors.New("an error"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Eventually(buffer).Should(gbytes.Say("failed-to-set-app-instances"))
				Eventually(buffer).Should(gbytes.Say("an error"))

				Expect(scalingEngineDB.SaveScalingHistoryArgsForCall(0)).To(Equal(&models.AppScalingHistory{
					AppId:        "an-app-id",
					Timestamp:    clock.Now().UnixNano(),
					ScalingType:  models.ScalingTypeSchedule,
					Status:       models.ScalingStatusFailed,
					OldInstances: 2,
					NewInstances: 3,
					Reason:       "schedule ends",
					Error:        "failed to set app instances: an error",
					Message:      "limited by min instances 3",
				}))

			})
		})

	})
})
