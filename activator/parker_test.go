package activator_test

import (
	"context"
	"errors"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeBinder is a scripted activator.RouteBinder.
type fakeBinder struct {
	upsiGUID  string
	upsiErr   error
	routes    []cf.Route
	routesErr error
	spaceGUID string
	spaceErr  error
	shared    []string // space GUIDs shared into
	shareErr  error
	bound     []string // route GUIDs bound
	unbound   []string // route GUIDs unbound
	bindErr   error
}

func (f *fakeBinder) GetAppRoutes(_ context.Context, _ cf.Guid) ([]cf.Route, error) {
	return f.routes, f.routesErr
}
func (f *fakeBinder) GetAppSpaceGUID(_ context.Context, _ cf.Guid) (string, error) {
	if f.spaceGUID == "" && f.spaceErr == nil {
		return "app-space", nil
	}
	return f.spaceGUID, f.spaceErr
}
func (f *fakeBinder) GetUserProvidedServiceInstanceGUID(_ context.Context, _ string) (string, error) {
	return f.upsiGUID, f.upsiErr
}
func (f *fakeBinder) ShareServiceInstanceWithSpace(_ context.Context, _, spaceGUID string) error {
	if f.shareErr != nil {
		return f.shareErr
	}
	f.shared = append(f.shared, spaceGUID)
	return nil
}
func (f *fakeBinder) BindRouteService(_ context.Context, routeGUID, _ string) error {
	if f.bindErr != nil {
		return f.bindErr
	}
	f.bound = append(f.bound, routeGUID)
	return nil
}
func (f *fakeBinder) UnbindRouteService(_ context.Context, routeGUID, _ string) error {
	f.unbound = append(f.unbound, routeGUID)
	return nil
}

var _ = Describe("CFParker", func() {
	var (
		binder   *fakeBinder
		registry activator.Registry
		parker   activator.Parker
	)

	BeforeEach(func() {
		binder = &fakeBinder{
			upsiGUID: "upsi-guid",
			routes: []cf.Route{
				{Guid: "route-1", URL: "https://app-1.example.com"},
				{Guid: "route-2", URL: "https://app-1-alt.example.com"},
			},
		}
		registry = activator.NewInMemoryRegistry()
		logger := lagertest.NewTestLogger("parker-test")
		parker = activator.NewCFParker(logger, binder, "autoscaler-activator-rs", registry)
	})

	Describe("Park", func() {
		It("binds every app route to the route-service UPSI and records them", func() {
			Expect(parker.Park(context.Background(), "app-1")).To(Succeed())
			Expect(binder.shared).To(ConsistOf("app-space"))
			Expect(binder.bound).To(ConsistOf("route-1", "route-2"))
			Expect(registry.IsParked("app-1")).To(BeTrue())
			Expect(registry.RoutesFor("app-1")).To(ConsistOf("https://app-1.example.com", "https://app-1-alt.example.com"))
		})

		It("is a no-op for an app with no routes", func() {
			binder.routes = nil
			Expect(parker.Park(context.Background(), "app-1")).To(Succeed())
			Expect(binder.bound).To(BeEmpty())
			Expect(registry.IsParked("app-1")).To(BeFalse())
		})

		It("fails when binding fails", func() {
			binder.bindErr = errors.New("bind boom")
			Expect(parker.Park(context.Background(), "app-1")).To(MatchError(ContainSubstring("bind boom")))
		})

		It("fails when the UPSI cannot be resolved", func() {
			binder.upsiErr = errors.New("no upsi")
			Expect(parker.Park(context.Background(), "app-1")).To(MatchError(ContainSubstring("no upsi")))
		})
	})

	Describe("Unpark", func() {
		It("unbinds every app route and clears the registry", func() {
			registry.MarkParked("app-1", []string{"https://app-1.example.com"})
			Expect(parker.Unpark(context.Background(), "app-1")).To(Succeed())
			Expect(binder.unbound).To(ConsistOf("route-1", "route-2"))
			Expect(registry.IsParked("app-1")).To(BeFalse())
		})
	})
})
