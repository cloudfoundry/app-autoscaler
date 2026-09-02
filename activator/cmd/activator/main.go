package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"

	"code.cloudfoundry.org/lager/v3"
	"github.com/tedsuo/ifrit"
)

func main() {
	conf, logger := startup.Bootstrap("activator", config.LoadConfig)

	// CF client for route-service bind/unbind.
	cfClient := startup.CreateAndLoginCFClient(&conf.CF, logger)

	registry := activator.NewInMemoryRegistry()
	parker := activator.NewCFParker(logger, cfClient, conf.RouteServiceUPSIName, conf.RouteServiceURL, registry)

	// Activator -> scaling engine (wake) client.
	engineHTTPClient, err := helpers.CreateHTTPSClient(&conf.ScalingEngine.TLSClientCerts, helpers.DefaultClientConfig(), logger.Session("scaling_engine_client"))
	startup.ExitOnError(err, logger, "failed to create scaling engine http client")
	engineClient := activator.NewScalingEngineClient(logger, engineHTTPClient, conf.ScalingEngine.ScalingEngineURL)

	// routing-api event subscriber (readiness signal) + Loop B watcher.
	subscriber := activator.NewRoutingAPISubscriber(logger, conf.RoutingAPI.URL, conf.RoutingAPI.SkipSSLValidation, conf.RoutingAPI.UAACreds)
	watcher := activator.NewReadinessWatcher(logger, subscriber, registry, parker)

	// Client used to forward held requests to the woken app (via Gorouter).
	forwardClient := &http.Client{}

	handler := activator.NewHandler(logger, parker, engineClient, watcher, registry, forwardClient, conf.ReadinessTimeout)

	activatorServer := server.NewServer(logger.Session("http_server"), conf, handler)
	xm := auth.NewXfccAuthMiddleware(logger, conf.CFServer.XFCC)

	startup.StartService(logger,
		startup.Server("readiness_watcher", func() (ifrit.Runner, error) { return watcherRunner(logger, watcher), nil }),
		startup.Server("https_server", activatorServer.CreateMtlsServer),
		startup.Server("health_server", activatorServer.CreateHealthServer),
		startup.Server("cf_server", func() (ifrit.Runner, error) { return activatorServer.CreateCFServer(xm) }),
	)
}

// watcherRunner runs the ReadinessWatcher's event loop (Loop B) until signaled.
// A failure to reach the routing-api event stream must NOT crash the activator:
// the health, park/unpark, and route-service surfaces stay useful, and the
// watcher reconnects with backoff.
func watcherRunner(logger lager.Logger, watcher activator.ReadinessWatcher) ifrit.Runner {
	return ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			const retryInterval = 15 * time.Second
			for {
				if err := watcher.Run(ctx); err != nil && ctx.Err() == nil {
					logger.Error("readiness-watcher-stopped-retrying", err, lager.Data{"retryIn": retryInterval.String()})
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(retryInterval):
				}
			}
		}()

		close(ready)
		<-signals
		cancel()
		return nil
	})
}
