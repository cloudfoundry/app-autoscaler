package activator_test

import (
	"context"
	"errors"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstate"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type observationSource struct {
	page appstate.Changes
	err  error
}

func (s *observationSource) Changes(context.Context, int64) (appstate.Changes, error) {
	return s.page, s.err
}
func (*observationSource) Invalidate(context.Context, string) error { return nil }

var _ = Describe("Observed parking reconciliation", func() {
	var r *activator.Reconciler
	var source *observationSource
	var cfc *fakes.FakeCFClient
	var policies *fakes.FakePolicyDB
	var registry activator.Registry
	var registrar *fakeRegistrar
	var app *cf.AppAndProcesses
	BeforeEach(func() {
		source = &observationSource{page: appstate.Changes{CheckedAt: time.Now(), Cursor: 1, Items: []appstate.Observation{{AppID: "app", Managed: true, Instances: 0}}}}
		cfc = &fakes.FakeCFClient{}
		policies = &fakes.FakePolicyDB{}
		app = &cf.AppAndProcesses{App: &cf.App{State: "STARTED"}, Processes: cf.Processes{{Instances: 0}}}
		cfc.GetAppAndProcessesReturns(app, nil)
		policies.GetAppPolicyReturns(&models.PolicyDefinition{InstanceMin: 0}, nil)
		registry = activator.NewInMemoryRegistry()
		registrar = &fakeRegistrar{}
		parker := activator.NewCFParker(lagertest.NewTestLogger("reconcile"), &fakeRouteLister{routes: []cf.Route{{URL: "app.example.com"}}}, registrar, registry)
		r = &activator.Reconciler{Source: source, CF: cfc, Policies: policies, Parker: parker, Registry: registry, MaxAge: 30 * time.Second}
	})
	It("parks a manually zero-scaled app and recovers after registry loss", func() {
		Expect(r.Poll(context.Background())).To(Succeed())
		Expect(registry.IsParked("app")).To(BeTrue())
		registry.MarkUnparked("app")
		Expect(r.Poll(context.Background())).To(Succeed())
		Expect(registry.IsParked("app")).To(BeTrue())
	})
	It("does not unpark merely because desired instances increase", func() {
		Expect(r.Poll(context.Background())).To(Succeed())
		app.Processes[0].Instances = 2
		Expect(r.Poll(context.Background())).To(Succeed())
		Expect(registry.IsParked("app")).To(BeTrue())
		Expect(registrar.unregistered).To(BeEmpty())
	})
	It("does not replay a stale zero observation over a manual scale-up", func() {
		app.Processes[0].Instances = 2
		Expect(r.Poll(context.Background())).To(Succeed())
		Expect(registry.IsParked("app")).To(BeFalse())
	})
	It("unparks stopped apps and does not interpret zero as a stop", func() {
		Expect(r.Poll(context.Background())).To(Succeed())
		app.App.State = "STOPPED"
		Expect(r.Poll(context.Background())).To(Succeed())
		Expect(registry.IsParked("app")).To(BeFalse())
	})
	It("cleans up on unbinding or deletion", func() {
		Expect(r.Poll(context.Background())).To(Succeed())
		source.page.Items[0].Managed = false
		Expect(r.Poll(context.Background())).To(Succeed())
		Expect(registry.IsParked("app")).To(BeFalse())
	})
	It("respects disabled autoscaling and a changed minimum", func() {
		label := "disabled"
		app.App.DisableAutoscaling = &label
		Expect(r.Poll(context.Background())).To(Succeed())
		Expect(registry.IsParked("app")).To(BeFalse())
		app.App.DisableAutoscaling = nil
		Expect(r.Poll(context.Background())).To(Succeed())
		policies.GetAppPolicyReturns(&models.PolicyDefinition{InstanceMin: 1}, nil)
		Expect(r.Poll(context.Background())).To(Succeed())
		Expect(registry.IsParked("app")).To(BeFalse())
	})
	It("does not act on dirty observations, reader lag or transport failure", func() {
		source.page.Items[0].Dirty = true
		Expect(r.Poll(context.Background())).To(Succeed())
		Expect(registry.IsParked("app")).To(BeFalse())
		source.page.CheckedAt = time.Now().Add(-time.Minute)
		Expect(r.Poll(context.Background())).To(HaveOccurred())
		source.err = errors.New("offline")
		Expect(r.Poll(context.Background())).To(HaveOccurred())
	})
})
