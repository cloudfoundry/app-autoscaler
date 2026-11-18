package scalingengine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestComputeNewMemoryMB(t *testing.T) {
	for _, tc := range []struct {
		adjustment              string
		current, min, max, want int
		invalid                 bool
	}{
		{"+124", 512, 128, 2048, 636, false},
		{"-128", 512, 128, 2048, 384, false},
		{"+10%", 513, 128, 2048, 565, false},
		{"-50%", 513, 128, 2048, 257, false},
		{"-100%", 513, 128, 2048, 128, false},
		{"+1000000000000000000000000000%", 512, 128, 2048, 2048, false},
		{"-1000000000000000000000000000", 512, 128, 2048, 128, false},
		{"+128", 2048, 128, 2048, 2048, false},
		{"-128", 64, 128, 2048, 128, false},
		{"-1%", 128, 1, 2048, 127, false},
		{"-1%", 1, 1, 2048, 1, false},
		{"-101%", 512, 128, 2048, 0, true},
		{"1", 512, 128, 2048, 0, true},
		{"+0", 512, 128, 2048, 0, true},
		{"+1.5%", 512, 128, 2048, 0, true},
		{"+1", 0, 128, 2048, 0, true},
		{"+1", 512, 2048, 128, 0, true},
	} {
		t.Run(tc.adjustment, func(t *testing.T) {
			got, err := ComputeNewMemoryMB(tc.current, tc.min, tc.max, tc.adjustment)
			if (err != nil) != tc.invalid || got != tc.want {
				t.Fatalf("got (%d, %v), want (%d, invalid=%t)", got, err, tc.want, tc.invalid)
			}
		})
	}
}

var _ = Describe("Vertical scaling and deployment completion", func() {
	var (
		engine   ScalingEngine
		cfc      *fakes.FakeCFClient
		policyDB *fakes.FakePolicyDB
		engineDB *fakes.FakeScalingEngineDB
		clk      *fakeclock.FakeClock
		policy   *models.PolicyDefinition
		trigger  *models.Trigger
		app      *cf.AppAndProcesses
	)
	BeforeEach(func() {
		cfc, policyDB, engineDB = &fakes.FakeCFClient{}, &fakes.FakePolicyDB{}, &fakes.FakeScalingEngineDB{}
		clk = fakeclock.NewFakeClock(time.Now())
		engine = NewScalingEngine(lagertest.NewTestLogger("vertical-scaling"), cfc, policyDB, engineDB, clk, 300, 32)
		rule := &models.ScalingRule{MetricType: models.MetricNameEntitlementUtil, Threshold: 80, Operator: ">=",
			Adjustment: "+128", BreachDurationSeconds: 60, CoolDownSeconds: 120}
		policy = &models.PolicyDefinition{InstanceMin: 1, InstanceMax: 4, MinMemoryMB: 256, MaxMemoryMB: 1024,
			VerticalScalingRules: []*models.ScalingRule{rule}}
		trigger = &models.Trigger{ScalingDimension: models.ScalingDimensionVertical, MetricType: rule.MetricType,
			MetricUnit: "%", Threshold: rule.Threshold, Operator: rule.Operator, Adjustment: rule.Adjustment,
			BreachDurationSeconds: rule.BreachDurationSeconds, CoolDownSeconds: rule.CoolDownSeconds}
		app = &cf.AppAndProcesses{App: &cf.App{State: models.AppStatusStarted},
			Processes: cf.Processes{{Type: cf.ProcessTypeWeb, Instances: 2, MemoryInMb: 512}}}
		cfc.GetAppAndProcessesReturns(app, nil)
		cfc.CreateRollingMemoryDeploymentReturns(&cf.Deployment{GUID: "deployment-guid", StatusValue: "ACTIVE"}, nil)
		policyDB.GetAppPolicyReturns(policy, nil)
		engineDB.CanScaleAppReturns(true, 0, nil)
	})

	It("persists the original request before POST and does not start cooldown at acceptance", func() {
		cfc.CreateRollingMemoryDeploymentStub = func(_ context.Context, id cf.Guid, target int, correlation string) (*cf.Deployment, error) {
			Expect(engineDB.SaveScalingHistoryCallCount()).To(Equal(1))
			h := engineDB.SaveScalingHistoryArgsForCall(0)
			Expect(h.Status).To(Equal(models.ScalingStatusPending))
			Expect(h.CorrelationID).To(Equal(correlation))
			Expect(h.CoolDownSeconds).To(Equal(120))
			Expect(h.OldMemoryMB).To(Equal(512))
			Expect(h.NewMemoryMB).To(Equal(640))
			Expect(h.NewInstances).To(Equal(2))
			Expect(id.String()).To(Equal("app-guid"))
			Expect(target).To(Equal(640))
			return &cf.Deployment{GUID: "deployment-guid"}, nil
		}
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(models.ScalingStatusPending))
		Expect(result.Adjustment).To(Equal(128))
		Expect(result.DeploymentGUID).To(Equal("deployment-guid"))
		Expect(result.CooldownExpiredAt).To(BeZero())
		Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())
		Expect(engineDB.UpdateScalingCooldownExpireTimeCallCount()).To(BeZero())
		Expect(engineDB.SetScalingDeploymentCallCount()).To(Equal(1))
	})

	It("does not POST when durable history cannot be saved", func() {
		engineDB.SaveScalingHistoryReturns(errors.New("database failure"))
		_, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).To(HaveOccurred())
		Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(BeZero())
		Expect(engineDB.SaveScalingHistoryCallCount()).To(Equal(1))
	})

	It("persists the effective default cooldown rather than a future policy value", func() {
		trigger.CoolDownSeconds, policy.VerticalScalingRules[0].CoolDownSeconds = 0, 0
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(models.ScalingStatusPending))
		Expect(engineDB.SaveScalingHistoryArgsForCall(0).CoolDownSeconds).To(Equal(300))
	})

	It("leaves an ambiguous POST outcome open without replaying it", func() {
		cfc.CreateRollingMemoryDeploymentReturns(nil, errors.New("connection reset after write"))
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(models.ScalingStatusUnknown))
		Expect(result.CooldownExpiredAt).To(BeZero())
		Expect(engineDB.MarkScalingRequestUnknownCallCount()).To(Equal(1))
		Expect(engineDB.CompleteScalingHistoryCallCount()).To(BeZero())
		Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(Equal(1))
	})

	It("records definitive rejection without starting cooldown", func() {
		cfc.CreateRollingMemoryDeploymentReturns(nil, &cf.CfError{StatusCode: 422})
		_, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).To(HaveOccurred())
		Expect(engineDB.CompleteScalingHistoryCallCount()).To(Equal(1))
		_, _, status, _, _, _ := engineDB.CompleteScalingHistoryArgsForCall(0)
		Expect(status).To(Equal(models.ScalingStatusFailed))
		Expect(engineDB.UpdateScalingCooldownExpireTimeCallCount()).To(BeZero())
	})

	DescribeTable("blocks mutations safely", func(active bool, open bool, queryErr bool, vertical bool) {
		cfc.HasActiveDeploymentReturns(active, nil)
		engineDB.HasOpenScalingHistoryReturns(open, nil)
		if queryErr {
			cfc.HasActiveDeploymentReturns(false, errors.New("CAPI unavailable"))
		}
		if !vertical {
			trigger.ScalingDimension = ""
		}
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		if queryErr {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal(models.ScalingStatusIgnored))
		}
		Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(BeZero())
		Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())
	},
		Entry("active vertical", true, false, false, true),
		Entry("unresolved vertical", false, true, false, true),
		Entry("failed query vertical", false, false, true, true),
		Entry("active horizontal", true, false, false, false),
		Entry("unresolved horizontal", false, true, false, false),
		Entry("failed query horizontal", false, false, true, false))

	It("returns the persisted app-wide cooldown", func() {
		expire := clk.Now().Add(120 * time.Second).UnixNano()
		engineDB.CanScaleAppReturns(false, expire, nil)
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(models.ScalingStatusIgnored))
		Expect(result.CooldownExpiredAt).To(Equal(expire))
		Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(BeZero())
	})

	It("rejects a stale trigger after a policy change", func() {
		policy.VerticalScalingRules[0].Threshold = 90
		_, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).To(HaveOccurred())
		Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(BeZero())
	})

	It("does not deploy a clamped unchanged target", func() {
		app.Processes[0].MemoryInMb = policy.MaxMemoryMB
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Reason).To(Equal("unchanged_target"))
		Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(BeZero())
	})

	DescribeTable("checks both priorities in both directions", func(priority models.ScalingDimension, vertical, growing bool, expectedReason string) {
		policy.ScalingPriority = priority
		if !growing {
			trigger.Adjustment = "-128"
			policy.VerticalScalingRules[0].Adjustment = "-128"
		}
		if !vertical {
			trigger.ScalingDimension = ""
			if growing {
				trigger.Adjustment = "+1"
			} else {
				trigger.Adjustment = "-1"
			}
		}
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Reason).To(Equal(expectedReason))
		Expect(result.Status).To(Equal(models.ScalingStatusIgnored))
		Expect(result.CooldownExpiredAt).To(BeZero())
		Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(BeZero())
		Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())
	},
		Entry("vertical priority horizontal up", models.ScalingDimensionVertical, false, true, "priority_requires_max_memory"),
		Entry("vertical priority vertical down", models.ScalingDimensionVertical, true, false, "priority_requires_min_instances"),
		Entry("horizontal priority vertical up", models.ScalingDimensionHorizontal, true, true, "priority_requires_max_instances"),
		Entry("horizontal priority horizontal down", models.ScalingDimensionHorizontal, false, false, "priority_requires_min_memory"))

	It("uses active schedule limits for vertical priority", func() {
		policy.ScalingPriority = models.ScalingDimensionHorizontal
		engineDB.GetActiveScheduleReturns(&models.ActiveSchedule{InstanceMin: 1, InstanceMax: 2}, nil)
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(models.ScalingStatusPending))
	})

	DescribeTable("allows the other dimension once its priority boundary is reached", func(priority models.ScalingDimension, vertical, growing bool, memory, instances int) {
		policy.ScalingPriority = priority
		app.Processes[0].MemoryInMb, app.Processes[0].Instances = memory, instances
		if !growing {
			trigger.Adjustment, policy.VerticalScalingRules[0].Adjustment = "-128", "-128"
		}
		if !vertical {
			trigger.ScalingDimension = ""
			if growing {
				trigger.Adjustment = "+1"
			} else {
				trigger.Adjustment = "-1"
			}
		}
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).NotTo(HaveOccurred())
		if vertical {
			Expect(result.Status).To(Equal(models.ScalingStatusPending))
			Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(Equal(1))
		} else {
			Expect(result.Status).To(Equal(models.ScalingStatusSucceeded))
			Expect(cfc.ScaleAppWebProcessCallCount()).To(Equal(1))
		}
	},
		Entry("maximum memory permits horizontal up", models.ScalingDimensionVertical, false, true, 1024, 2),
		Entry("minimum instances permits vertical down", models.ScalingDimensionVertical, true, false, 512, 1),
		Entry("maximum instances permits vertical up", models.ScalingDimensionHorizontal, true, true, 512, 4),
		Entry("minimum memory permits horizontal down", models.ScalingDimensionHorizontal, false, false, 256, 2))

	It("uses the scheduled minimum instances for vertical down", func() {
		policy.ScalingPriority = models.ScalingDimensionVertical
		trigger.Adjustment, policy.VerticalScalingRules[0].Adjustment = "-128", "-128"
		engineDB.GetActiveScheduleReturns(&models.ActiveSchedule{InstanceMin: 2, InstanceMax: 4}, nil)
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(models.ScalingStatusPending))
	})

	It("keeps history correlation recoverable if attaching an accepted deployment fails", func() {
		engineDB.SetScalingDeploymentReturns(errors.New("database connection lost"))
		result, err := engine.Scale(context.Background(), "app-guid", trigger)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(models.ScalingStatusPending))
		Expect(result.DeploymentGUID).To(Equal("deployment-guid"))
		Expect(engineDB.SaveScalingHistoryArgsForCall(0).CorrelationID).NotTo(BeEmpty())
		Expect(engineDB.SaveScalingHistoryCallCount()).To(Equal(1))
		Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(Equal(1))
	})

	It("preserves an ended schedule while its adjustment is deployment-blocked", func() {
		engineDB.GetActiveScheduleReturns(&models.ActiveSchedule{ScheduleId: "schedule", InstanceMin: 3, InstanceMax: 4}, nil)
		cfc.HasActiveDeploymentReturns(true, nil)
		err := engine.RemoveActiveSchedule(context.Background(), "app-guid", "schedule")
		Expect(err).To(HaveOccurred())
		Expect(engineDB.RemoveActiveScheduleCallCount()).To(BeZero())
		Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())
	})

	It("does not commit a deferred schedule, allowing fresh scheduler reconciliation", func() {
		cfc.HasActiveDeploymentReturns(true, nil)
		err := engine.SetActiveSchedule(context.Background(), "app-guid", &models.ActiveSchedule{ScheduleId: "schedule", InstanceMin: 3, InstanceMax: 4})
		Expect(err).To(HaveOccurred())
		Expect(engineDB.SetActiveScheduleCallCount()).To(BeZero())
		Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())
		Expect(engineDB.SaveScalingHistoryArgsForCall(0).Message).To(Equal("active_deployment"))
	})

	Describe("completion", func() {
		var h *models.AppScalingHistory
		BeforeEach(func() {
			h = &models.AppScalingHistory{AppId: "app-guid", ScalingDimension: models.ScalingDimensionVertical,
				CorrelationID: "correlation", DeploymentGUID: "deployment-guid", Status: models.ScalingStatusPending,
				OldMemoryMB: 512, NewMemoryMB: 640, CoolDownSeconds: 120}
			engineDB.GetOpenScalingHistoriesReturns([]*models.AppScalingHistory{h}, nil)
		})
		DescribeTable("processes only recognized terminal outcomes", func(value, reason string, expected models.ScalingStatus, completes bool) {
			cfc.GetDeploymentReturns(&cf.Deployment{GUID: h.DeploymentGUID, StatusValue: value, StatusReason: reason}, nil)
			Expect(CheckDeployments(context.Background(), engine)).NotTo(HaveOccurred())
			if completes {
				Expect(engineDB.CompleteScalingHistoryCallCount()).To(Equal(1))
				_, correlation, status, timestamp, _, _ := engineDB.CompleteScalingHistoryArgsForCall(0)
				Expect(correlation).To(Equal(h.CorrelationID))
				Expect(status).To(Equal(expected))
				Expect(timestamp).To(Equal(clk.Now().UnixNano()))
			} else {
				Expect(engineDB.CompleteScalingHistoryCallCount()).To(BeZero())
			}
			Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(BeZero())
			Expect(engineDB.UpdateScalingCooldownExpireTimeCallCount()).To(BeZero())
		},
			Entry("active paused", "ACTIVE", "PAUSED", models.ScalingStatusPending, false),
			Entry("active canceling", "ACTIVE", "CANCELING", models.ScalingStatusPending, false),
			Entry("deployed", "FINALIZED", "DEPLOYED", models.ScalingStatusSucceeded, true),
			Entry("superseded", "FINALIZED", "SUPERSEDED", models.ScalingStatusSuperseded, true),
			Entry("aborted", "FINALIZED", "CANCELED", models.ScalingStatusAborted, true))

		It("recovers a missing POST response by correlation after restart", func() {
			h.DeploymentGUID, h.Status = "", models.ScalingStatusUnknown
			policyDB.GetAppPolicyReturns(nil, nil) // policy removal must not lose the original cooldown
			cfc.FindDeploymentReturns(&cf.Deployment{GUID: "recovered", StatusValue: "FINALIZED", StatusReason: "DEPLOYED"}, nil)
			Expect(CheckDeployments(context.Background(), engine)).NotTo(HaveOccurred())
			Expect(engineDB.SetScalingDeploymentCallCount()).To(Equal(1))
			Expect(engineDB.CompleteScalingHistoryCallCount()).To(Equal(1))
			Expect(policyDB.GetAppPolicyCallCount()).To(BeZero())
			Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(BeZero())
		})

		It("keeps failed GETs and unfamiliar final states open", func() {
			cfc.GetDeploymentReturns(nil, errors.New("resource missing or temporarily unavailable"))
			Expect(CheckDeployments(context.Background(), engine)).To(HaveOccurred())
			Expect(engineDB.CompleteScalingHistoryCallCount()).To(BeZero())
			cfc.GetDeploymentReturns(&cf.Deployment{StatusValue: "FINALIZED", StatusReason: "NEW_REASON"}, nil)
			Expect(CheckDeployments(context.Background(), engine)).To(HaveOccurred())
			Expect(engineDB.CompleteScalingHistoryCallCount()).To(BeZero())
		})

		It("makes a missing correlation visible without retrying the POST", func() {
			h.DeploymentGUID = ""
			cfc.FindDeploymentReturns(nil, nil)
			Expect(CheckDeployments(context.Background(), engine)).To(HaveOccurred())
			Expect(engineDB.MarkScalingRequestUnknownCallCount()).To(Equal(1))
			Expect(engineDB.CompleteScalingHistoryCallCount()).To(BeZero())
			Expect(cfc.CreateRollingMemoryDeploymentCallCount()).To(BeZero())
		})
	})
})
