package scalingengine_test

import (
	"context"
	"errors"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstate"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type stateSource struct {
	changes     appstate.Changes
	err         error
	invalidated []string
}

func (s *stateSource) Changes(context.Context, int64) (appstate.Changes, error) {
	return s.changes, s.err
}
func (s *stateSource) Invalidate(_ context.Context, id string) error {
	s.invalidated = append(s.invalidated, id)
	return nil
}

var _ = Describe("Event-invalidated scaling no-ops", func() {
	var engine ScalingEngine
	var cfc *fakes.FakeCFClient
	var policy *fakes.FakePolicyDB
	var database *fakes.FakeScalingEngineDB
	var cache *appstate.Cache
	var source *stateSource
	var clock *fakeclock.FakeClock
	BeforeEach(func() {
		cfc = &fakes.FakeCFClient{}
		policy = &fakes.FakePolicyDB{}
		database = &fakes.FakeScalingEngineDB{}
		clock = fakeclock.NewFakeClock(time.Now())
		cache = appstate.NewCache(30 * time.Second)
		source = &stateSource{changes: appstate.Changes{CheckedAt: clock.Now()}}
		Expect(cache.Poll(context.Background(), source)).To(Succeed())
		cfc.GetAppAndProcessesReturns(&cf.AppAndProcesses{App: &cf.App{State: "STARTED"}, Processes: cf.Processes{{Instances: 1}}}, nil)
		policy.GetAppPolicyReturns(&models.PolicyDefinition{InstanceMin: 1, InstanceMax: 3}, nil)
		database.CanScaleAppReturns(true, 0, nil)
		engine = NewScalingEngine(lagertest.NewTestLogger("state-cache"), cfc, policy, database, clock, 300, 32, WithAppState(cache, source))
	})
	scale := func(adjustment string) {
		_, err := engine.Scale(context.Background(), "app", &models.Trigger{Adjustment: adjustment})
		Expect(err).NotTo(HaveOccurred())
	}
	It("reads CF only once for repeated scale-ins at minimum", func() {
		for range 20 {
			scale("-1")
		}
		Expect(cfc.GetAppAndProcessesCallCount()).To(Equal(1))
		Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())
	})
	It("does the same at maximum", func() {
		cfc.GetAppAndProcessesReturns(&cf.AppAndProcesses{App: &cf.App{State: "STARTED"}, Processes: cf.Processes{{Instances: 3}}}, nil)
		for range 20 {
			scale("+1")
		}
		Expect(cfc.GetAppAndProcessesCallCount()).To(Equal(1))
		Expect(cfc.ScaleAppWebProcessCallCount()).To(BeZero())
	})
	It("never suppresses opposite-direction writes", func() {
		scale("-1")
		scale("+1")
		Expect(cfc.GetAppAndProcessesCallCount()).To(Equal(2))
		Expect(cfc.ScaleAppWebProcessCallCount()).To(Equal(1))
		Expect(source.invalidated).To(Equal([]string{"app", "app"}))
	})
	It("verifies CF again after a manual scaling event", func() {
		scale("-1")
		source.changes.Items = []appstate.Observation{{AppID: "app", Dirty: true}}
		Expect(cache.Poll(context.Background(), source)).To(Succeed())
		cfc.GetAppAndProcessesReturns(&cf.AppAndProcesses{App: &cf.App{State: "STARTED"}, Processes: cf.Processes{{Instances: 2}}}, nil)
		scale("-1")
		Expect(cfc.ScaleAppWebProcessCallCount()).To(Equal(1))
	})
	It("honours changed policy and active schedule bounds", func() {
		scale("-1")
		policy.GetAppPolicyReturns(&models.PolicyDefinition{InstanceMin: 2, InstanceMax: 3}, nil)
		scale("-1")
		Expect(cfc.ScaleAppWebProcessCallCount()).To(Equal(1))
		database.GetActiveScheduleReturns(&models.ActiveSchedule{InstanceMin: 3, InstanceMax: 4}, nil)
		scale("-1")
		Expect(cfc.ScaleAppWebProcessCallCount()).To(Equal(2))
	})
	It("falls back on expiry or observer failure", func() {
		scale("-1")
		clock.Increment(31 * time.Second)
		scale("-1")
		Expect(cfc.GetAppAndProcessesCallCount()).To(Equal(2))
		source.err = errors.New("offline")
		Expect(cache.Poll(context.Background(), source)).To(HaveOccurred())
		scale("-1")
		Expect(cfc.GetAppAndProcessesCallCount()).To(Equal(3))
	})
	It("always verifies wake triggers even if they would otherwise be no-ops", func() {
		scale("-1")
		_, err := engine.Scale(context.Background(), "app", &models.Trigger{Adjustment: "-1", BypassCooldown: true})
		Expect(err).NotTo(HaveOccurred())
		Expect(cfc.GetAppAndProcessesCallCount()).To(Equal(2))
	})
})
