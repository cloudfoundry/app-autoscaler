package activator

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	"code.cloudfoundry.org/lager/v3"
)

// RouteLister is the subset of the CF client the parker needs: listing an app's
// route URLs. Satisfied by cf.CFClient.
type RouteLister interface {
	GetAppRoutes(ctx context.Context, appId cf.Guid) ([]cf.Route, error)
}

// cfParker parks/unparks an app by (de)registering the activator as a backend
// for the app's route URIs on the Gorouter NATS bus. While parked, the activator
// is a live mTLS backend of the route, which keeps the route mapped even though
// the app has zero instances (a plain route with no endpoints is pruned/404'd).
// See docs/design/scale-to-zero.md.
type cfParker struct {
	logger    lager.Logger
	cfClient  RouteLister
	registrar Registrar
	registry  Registry
	mu        sync.Mutex
}

func NewCFParker(logger lager.Logger, cfClient RouteLister, registrar Registrar, registry Registry) Parker {
	return &cfParker{
		logger:    logger.Session("cf-parker"),
		cfClient:  cfClient,
		registrar: registrar,
		registry:  registry,
	}
}

func (p *cfParker) Park(ctx context.Context, appID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	uris, err := p.appRouteURIs(ctx, appID)
	if err != nil {
		return err
	}
	if len(uris) > 0 {
		if err := p.registrar.Register(uris); err != nil {
			return fmt.Errorf("failed registering activator for app %s routes: %w", appID, err)
		}
	}
	var removed []string
	for _, old := range p.registry.RoutesFor(appID) {
		if !slices.Contains(uris, old) {
			removed = append(removed, old)
		}
	}
	if len(removed) > 0 {
		if err := p.registrar.Unregister(removed); err != nil {
			return fmt.Errorf("remove old parked routes: %w", err)
		}
	}
	if len(uris) == 0 {
		p.registry.MarkUnparked(appID)
		return nil
	}
	p.registry.MarkParked(appID, uris)
	p.logger.Info("parked", lager.Data{"appID": appID, "routes": uris})
	return nil
}

func (p *cfParker) Unpark(ctx context.Context, appID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Prefer the URIs recorded at park time (the app's live routes may already
	// differ); fall back to a fresh lookup if the registry has none.
	uris := p.registry.RoutesFor(appID)
	if len(uris) == 0 {
		var err error
		if uris, err = p.appRouteURIs(ctx, appID); err != nil {
			return err
		}
	}
	if len(uris) > 0 {
		if err := p.registrar.Unregister(uris); err != nil {
			return fmt.Errorf("failed unregistering activator for app %s routes: %w", appID, err)
		}
	}
	p.registry.MarkUnparked(appID)
	p.logger.Info("unparked", lager.Data{"appID": appID})
	return nil
}

func (p *cfParker) appRouteURIs(ctx context.Context, appID string) ([]string, error) {
	routeList, err := p.cfClient.GetAppRoutes(ctx, cf.Guid(appID))
	if err != nil {
		return nil, fmt.Errorf("failed getting routes for app %s: %w", appID, err)
	}
	uris := make([]string, 0, len(routeList))
	for _, r := range routeList {
		uris = append(uris, r.URL)
	}
	return uris, nil
}
