package activator_test

import (
	"context"
	"errors"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	"code.cloudfoundry.org/lager/v3/lagertest"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeRouteLister is a scripted activator.RouteLister.
type fakeRouteLister struct {
	routes    []cf.Route
	routesErr error
}

func (f *fakeRouteLister) GetAppRoutes(_ context.Context, _ cf.Guid) ([]cf.Route, error) {
	return f.routes, f.routesErr
}

// fakeRegistrar records register/unregister calls.
type fakeRegistrar struct {
	registered   [][]string
	unregistered [][]string
	registerErr  error
}

func (f *fakeRegistrar) Register(uris []string) error {
	if f.registerErr != nil {
		return f.registerErr
	}
	f.registered = append(f.registered, uris)
	return nil
}
func (f *fakeRegistrar) Unregister(uris []string) error {
	f.unregistered = append(f.unregistered, uris)
	return nil
}
func (f *fakeRegistrar) Run(_ <-chan struct{}) {}
func (f *fakeRegistrar) Conn() *nats.Conn      { return nil }

var _ = Describe("CFParker", func() {
	var (
		lister    *fakeRouteLister
		registrar *fakeRegistrar
		registry  activator.Registry
		parker    activator.Parker
	)

	BeforeEach(func() {
		lister = &fakeRouteLister{
			routes: []cf.Route{
				{URL: "https://app-1.example.com"},
				{URL: "https://app-1-alt.example.com"},
			},
		}
		registrar = &fakeRegistrar{}
		registry = activator.NewInMemoryRegistry()
		logger := lagertest.NewTestLogger("parker-test")
		parker = activator.NewCFParker(logger, lister, registrar, registry)
	})

	Describe("Park", func() {
		It("registers the activator for every app route URI and records them", func() {
			Expect(parker.Park(context.Background(), "app-1")).To(Succeed())
			Expect(registrar.registered).To(HaveLen(1))
			Expect(registrar.registered[0]).To(ConsistOf("https://app-1.example.com", "https://app-1-alt.example.com"))
			Expect(registry.IsParked("app-1")).To(BeTrue())
			Expect(registry.RoutesFor("app-1")).To(ConsistOf("https://app-1.example.com", "https://app-1-alt.example.com"))
		})

		It("is a no-op for an app with no routes", func() {
			lister.routes = nil
			Expect(parker.Park(context.Background(), "app-1")).To(Succeed())
			Expect(registrar.registered).To(BeEmpty())
			Expect(registry.IsParked("app-1")).To(BeFalse())
		})

		It("fails when registration fails", func() {
			registrar.registerErr = errors.New("nats boom")
			Expect(parker.Park(context.Background(), "app-1")).To(MatchError(ContainSubstring("nats boom")))
		})

		It("fails when route lookup fails", func() {
			lister.routesErr = errors.New("cf boom")
			Expect(parker.Park(context.Background(), "app-1")).To(MatchError(ContainSubstring("cf boom")))
		})
	})

	Describe("Unpark", func() {
		It("unregisters the recorded routes and clears the registry", func() {
			registry.MarkParked("app-1", []string{"https://app-1.example.com"})
			Expect(parker.Unpark(context.Background(), "app-1")).To(Succeed())
			Expect(registrar.unregistered).To(HaveLen(1))
			Expect(registrar.unregistered[0]).To(ConsistOf("https://app-1.example.com"))
			Expect(registry.IsParked("app-1")).To(BeFalse())
		})
	})
})
