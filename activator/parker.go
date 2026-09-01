package activator

import (
	"context"
	"fmt"
	"sync"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	"code.cloudfoundry.org/lager/v3"
)

// RouteBinder is the subset of the CF client the parker needs. It is satisfied
// by cf.CFClient.
type RouteBinder interface {
	GetAppRoutes(ctx context.Context, appId cf.Guid) ([]cf.Route, error)
	GetUserProvidedServiceInstanceGUID(ctx context.Context, name string) (string, error)
	BindRouteService(ctx context.Context, routeGUID, serviceInstanceGUID string) error
	UnbindRouteService(ctx context.Context, routeGUID, serviceInstanceGUID string) error
}

// cfParker binds/unbinds an app's routes to the activator route-service UPSI.
// The UPSI GUID is resolved once (by name) and cached.
type cfParker struct {
	logger     lager.Logger
	cfClient   RouteBinder
	upsiName   string
	registry   Registry
	upsiGUIDMu sync.Mutex
	upsiGUID   string
}

// NewCFParker returns a Parker that binds app routes to the named route-service
// UPSI (e.g. "autoscaler-activator-rs"). Bound routes are recorded in registry.
func NewCFParker(logger lager.Logger, cfClient RouteBinder, upsiName string, registry Registry) Parker {
	return &cfParker{
		logger:   logger.Session("cf-parker"),
		cfClient: cfClient,
		upsiName: upsiName,
		registry: registry,
	}
}

func (p *cfParker) resolveUPSIGUID(ctx context.Context) (string, error) {
	p.upsiGUIDMu.Lock()
	defer p.upsiGUIDMu.Unlock()
	if p.upsiGUID != "" {
		return p.upsiGUID, nil
	}
	guid, err := p.cfClient.GetUserProvidedServiceInstanceGUID(ctx, p.upsiName)
	if err != nil {
		return "", fmt.Errorf("failed resolving route-service UPSI %q: %w", p.upsiName, err)
	}
	p.upsiGUID = guid
	return guid, nil
}

func (p *cfParker) Park(ctx context.Context, appID string) error {
	upsiGUID, err := p.resolveUPSIGUID(ctx)
	if err != nil {
		return err
	}
	routeList, err := p.cfClient.GetAppRoutes(ctx, cf.Guid(appID))
	if err != nil {
		return fmt.Errorf("failed getting routes for app %s: %w", appID, err)
	}
	if len(routeList) == 0 {
		p.logger.Info("no-routes-to-park", lager.Data{"appID": appID})
		return nil
	}

	boundURLs := make([]string, 0, len(routeList))
	for _, r := range routeList {
		if err := p.cfClient.BindRouteService(ctx, r.Guid, upsiGUID); err != nil {
			return fmt.Errorf("failed binding route %s for app %s: %w", r.Guid, appID, err)
		}
		boundURLs = append(boundURLs, r.URL)
	}
	p.registry.MarkParked(appID, boundURLs)
	p.logger.Info("parked", lager.Data{"appID": appID, "routes": boundURLs})
	return nil
}

func (p *cfParker) Unpark(ctx context.Context, appID string) error {
	upsiGUID, err := p.resolveUPSIGUID(ctx)
	if err != nil {
		return err
	}
	routeList, err := p.cfClient.GetAppRoutes(ctx, cf.Guid(appID))
	if err != nil {
		return fmt.Errorf("failed getting routes for app %s: %w", appID, err)
	}
	for _, r := range routeList {
		if err := p.cfClient.UnbindRouteService(ctx, r.Guid, upsiGUID); err != nil {
			return fmt.Errorf("failed unbinding route %s for app %s: %w", r.Guid, appID, err)
		}
	}
	p.registry.MarkUnparked(appID)
	p.logger.Info("unparked", lager.Data{"appID": appID})
	return nil
}
