package activator

import (
	"context"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	"code.cloudfoundry.org/lager/v3"
)

// RouteBinder is the subset of the CF client the parker needs. It is satisfied
// by cf.CFClient.
type RouteBinder interface {
	GetAppRoutes(ctx context.Context, appId cf.Guid) ([]cf.Route, error)
	GetAppSpaceGUID(ctx context.Context, appId cf.Guid) (string, error)
	EnsureRouteServiceInstance(ctx context.Context, name, spaceGUID, routeServiceURL string) (string, error)
	BindRouteService(ctx context.Context, routeGUID, serviceInstanceGUID string) error
	UnbindRouteService(ctx context.Context, routeGUID, serviceInstanceGUID string) error
}

// cfParker binds/unbinds an app's routes to a route-service user-provided
// service instance that lives in the SAME space as the app. The activator
// ensures a per-space UPSI (pointing at its own route) rather than sharing one
// global UPSI across spaces, because cross-space service sharing may be disabled
// on the foundation. See docs/design/scale-to-zero.md.
type cfParker struct {
	logger          lager.Logger
	cfClient        RouteBinder
	upsiName        string
	routeServiceURL string
	registry        Registry
}

// NewCFParker returns a Parker that ensures a route-service UPSI named upsiName
// (with route_service_url = routeServiceURL, the activator's own route) in each
// app's space and binds the app's routes to it.
func NewCFParker(logger lager.Logger, cfClient RouteBinder, upsiName, routeServiceURL string, registry Registry) Parker {
	return &cfParker{
		logger:          logger.Session("cf-parker"),
		cfClient:        cfClient,
		upsiName:        upsiName,
		routeServiceURL: routeServiceURL,
		registry:        registry,
	}
}

func (p *cfParker) Park(ctx context.Context, appID string) error {
	routeList, err := p.cfClient.GetAppRoutes(ctx, cf.Guid(appID))
	if err != nil {
		return fmt.Errorf("failed getting routes for app %s: %w", appID, err)
	}
	if len(routeList) == 0 {
		p.logger.Info("no-routes-to-park", lager.Data{"appID": appID})
		return nil
	}

	upsiGUID, err := p.ensureUPSI(ctx, appID)
	if err != nil {
		return err
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
	routeList, err := p.cfClient.GetAppRoutes(ctx, cf.Guid(appID))
	if err != nil {
		return fmt.Errorf("failed getting routes for app %s: %w", appID, err)
	}
	if len(routeList) == 0 {
		p.registry.MarkUnparked(appID)
		return nil
	}

	upsiGUID, err := p.ensureUPSI(ctx, appID)
	if err != nil {
		return err
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

// ensureUPSI resolves the app's space and ensures the route-service UPSI exists
// there, returning its GUID.
func (p *cfParker) ensureUPSI(ctx context.Context, appID string) (string, error) {
	spaceGUID, err := p.cfClient.GetAppSpaceGUID(ctx, cf.Guid(appID))
	if err != nil {
		return "", fmt.Errorf("failed getting space for app %s: %w", appID, err)
	}
	upsiGUID, err := p.cfClient.EnsureRouteServiceInstance(ctx, p.upsiName, spaceGUID, p.routeServiceURL)
	if err != nil {
		return "", fmt.Errorf("failed ensuring route-service in space %s for app %s: %w", spaceGUID, appID, err)
	}
	return upsiGUID, nil
}
