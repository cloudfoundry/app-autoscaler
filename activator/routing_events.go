package activator

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	routing_api "code.cloudfoundry.org/routing-api"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"code.cloudfoundry.org/lager/v3"
)

// routingAPISubscriber is a RouteEventSubscriber backed by the CF routing-api
// event stream (GET /routing/v1/events). It fetches a fresh client-credentials
// token (scope routing.routes.read) before each subscribe.
type routingAPISubscriber struct {
	logger      lager.Logger
	apiURL      string
	skipTLS     bool
	tokenSource oauth2.TokenSource
}

// NewRoutingAPISubscriber builds a subscriber for the routing-api at apiURL,
// authenticating via UAA client-credentials from creds.
func NewRoutingAPISubscriber(logger lager.Logger, apiURL string, skipTLS bool, creds models.UAACreds) RouteEventSubscriber {
	cfg := &clientcredentials.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		TokenURL:     tokenURL(creds.URL),
	}
	ctx := context.Background()
	if creds.SkipSSLValidation {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, insecureHTTPClient())
	}
	return &routingAPISubscriber{
		logger:      logger.Session("routing-api-subscriber"),
		apiURL:      apiURL,
		skipTLS:     skipTLS,
		tokenSource: cfg.TokenSource(ctx),
	}
}

func (s *routingAPISubscriber) Subscribe() (RouteEventStream, error) {
	token, err := s.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain routing-api token: %w", err)
	}
	client := routing_api.NewClient(s.apiURL, s.skipTLS)
	client.SetToken(token.AccessToken)

	source, err := client.SubscribeToEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to routing-api events: %w", err)
	}
	return &routingAPIStream{source: source}, nil
}

// routingAPIStream adapts routing_api.EventSource to RouteEventStream.
type routingAPIStream struct {
	source routing_api.EventSource
}

func (s *routingAPIStream) Next() (RouteEvent, error) {
	event, err := s.source.Next()
	if err != nil {
		return RouteEvent{}, err
	}
	return RouteEvent{RouteURL: event.Route.Route, Action: event.Action}, nil
}

func (s *routingAPIStream) Close() error {
	return s.source.Close()
}

func tokenURL(uaaURL string) string {
	return fmt.Sprintf("%s/oauth/token", trimTrailingSlash(uaaURL))
}

func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func insecureHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // dev foundations opt-in via skip_ssl_validation
		},
	}
}
