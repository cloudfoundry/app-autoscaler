package plancheck_test

import (
	"autoscaler/api/config"
	"autoscaler/api/plancheck"
	"autoscaler/models"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PlanCheck", func() {
	var (
		quotaConfig      *config.PlanCheckConfig
		validationResult string
		qmc              *plancheck.PlanChecker
		ok               bool
		err              error
		testPolicy       models.ScalingPolicy
		testPlanId       string
	)
	BeforeEach(func() {})
	Context("CheckPlan", func() {
		JustBeforeEach(func() {
			qmc = plancheck.NewPlanChecker(quotaConfig, lagertest.NewTestLogger("Quota"))
			ok, validationResult, err = qmc.CheckPlan(testPolicy, testPlanId)
		})
		Context("when not configured", func() {
			BeforeEach(func() {
				testPolicy = models.ScalingPolicy{
					InstanceMin:  1,
					InstanceMax:  4,
					ScalingRules: nil,
					Schedules:    nil,
				}
				testPlanId = "test-plan"
				quotaConfig = nil
			})
			It("returns -1", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("when configured", func() {
			BeforeEach(func() {
				testPolicy = models.ScalingPolicy{
					InstanceMin:  1,
					InstanceMax:  4,
					ScalingRules: nil,
					Schedules:    nil,
				}
				testPlanId = "test-plan"
				quotaConfig = &config.PlanCheckConfig{
					PlanDefinitions: map[string]config.PlanDefinition{
						"not-checked-plan-id": {
							PlanCheckEnabled:  false,
							SchedulesCount:    0,
							ScalingRulesCount: 0,
							PlanUpdateable: true,
						},
						"small-plan-id": {
							PlanCheckEnabled:  true,
							SchedulesCount:    1,
							ScalingRulesCount: 1,
							PlanUpdateable: false,
						},
						"large-plan-id": {
							PlanCheckEnabled:  true,
							SchedulesCount:    10,
							ScalingRulesCount: 10,
							PlanUpdateable: false,
						},
					},
				}
			})
			Context("when checking a policy with an unknown plan", func() {
				It("errors on unknown plan", func() {
					Expect(err).To(HaveOccurred())
				})
			})
			Context("when checking a plan with too many rules", func() {
				BeforeEach(func() {
					testPlanId = "small-plan-id"
					testPolicy = models.ScalingPolicy{
						ScalingRules: []*models.ScalingRule{
							{},
							{},
						},
					}
				})
				It("fails the check", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(validationResult).NotTo(BeEmpty())
					Expect(ok).To(BeFalse())
				})
			})
			Context("when checking a plan with enough rules allowed", func() {
				BeforeEach(func() {
					testPlanId = "small-plan-id"
					testPolicy = models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 10,
						ScalingRules: []*models.ScalingRule{
							{},
						},
					}
				})
				It("passes the check", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(validationResult).To(BeEmpty())
					Expect(ok).To(BeTrue())
				})
			})
			Context("when checking a plan with too many schedules", func() {
				BeforeEach(func() {
					testPlanId = "small-plan-id"
					testPolicy = models.ScalingPolicy{
						InstanceMin:  1,
						InstanceMax:  10,
						ScalingRules: nil,
						Schedules: &models.ScalingSchedules{
							Timezone: "",
							RecurringSchedules: []*models.RecurringSchedule{
								{},
							},
							SpecificDateSchedules: []*models.SpecificDateSchedule{
								{},
							},
						},
					}
				})
				It("fails the check", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(validationResult).NotTo(BeEmpty())
					Expect(ok).To(BeFalse())
				})
			})
			Context("when checking a plan with enough schedules allowed", func() {
				BeforeEach(func() {
					testPlanId = "small-plan-id"
					testPolicy = models.ScalingPolicy{
						InstanceMin:  1,
						InstanceMax:  10,
						ScalingRules: nil,
						Schedules: &models.ScalingSchedules{
							Timezone: "",
							RecurringSchedules: []*models.RecurringSchedule{
								{},
							},
						},
					}
				})
				It("passes the check", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(validationResult).To(BeEmpty())
					Expect(ok).To(BeTrue())
				})
			})

			Context("IsUpdatable", func() {
				BeforeEach(func() {
					quotaConfig = &config.PlanCheckConfig{
						PlanDefinitions: map[string]config.PlanDefinition{
							"updatable-plan": {
								false,
								0,
								0,
								true,
							},
							"non-updatable-plan": {
								true,
								1,
								1,
								false,
							},
						},
					}
				})
				It("is plan updatable", func() {
					isPlanUpdatable, err:=qmc.IsPlanUpdatable("updatable-plan")
					Expect(isPlanUpdatable).To(Equal(true))
					Expect(err).To(BeNil())
				})
				It("is plan not updatable", func() {
					isPlanUpdatable, err:=qmc.IsPlanUpdatable("non-updatable-plan")
					Expect(isPlanUpdatable).To(Equal(false))
					Expect(err).To(BeNil())
				})
				It("if plan does not exist", func() {
					isPlanUpdatable, err:=qmc.IsPlanUpdatable("non-existent-plan")
					Expect(isPlanUpdatable).To(Equal(false))
					Expect(err.Error()).To(Equal("unknown plan id \"non-existent-plan\""))
				})
			})
		})
	})
})
