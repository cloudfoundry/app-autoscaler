package scalingengine

import (
	"context"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/v3"
)

// httpActivator is an Activator that drives the activator microservice over
// HTTP (mTLS). Park binds the app's routes to the activator route-service
// before the engine scales the app to zero; Unpark reverses it.
type httpActivator struct {
	logger       lager.Logger
	httpClient   *http.Client
	activatorURL string
}

// NewHTTPActivator returns an Activator backed by the activator service at
// activatorURL, using the given mTLS http client.
func NewHTTPActivator(logger lager.Logger, httpClient *http.Client, activatorURL string) Activator {
	return &httpActivator{
		logger:       logger.Session("http-activator"),
		httpClient:   httpClient,
		activatorURL: activatorURL,
	}
}

func (a *httpActivator) Park(ctx context.Context, appId string) error {
	return a.do(ctx, http.MethodPut, routes.ActivatorParkRouteName, appId)
}

func (a *httpActivator) Unpark(ctx context.Context, appId string) error {
	return a.do(ctx, http.MethodDelete, routes.ActivatorUnparkRouteName, appId)
}

func (a *httpActivator) do(ctx context.Context, method, routeName, appId string) error {
	path, err := routes.NewRouter().CreateActivatorRoutes().Get(routeName).URLPath("appid", appId)
	if err != nil {
		return fmt.Errorf("failed to build activator url for %s: %w", appId, err)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.activatorURL+path.Path, nil)
	if err != nil {
		return fmt.Errorf("failed to build activator request for %s: %w", appId, err)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed calling activator for %s: %w", appId, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("activator returned status %d for %s %s", resp.StatusCode, method, appId)
	}
	return nil
}
