package main

import (
	"context"
	"net/http"
	"os"

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
	parker := activator.NewCFParker(logger, cfClient, conf.RouteServiceUPSIName, registry)

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
func watcherRunner(logger lager.Logger, watcher activator.ReadinessWatcher) ifrit.Runner {
	return ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errCh := make(chan error, 1)
		go func() { errCh <- watcher.Run(ctx) }()

		close(ready)

		select {
		case <-signals:
			cancel()
			return nil
		case err := <-errCh:
			// Log and keep the process alive on stream errors; the loop can be
			// restarted. For the PoC we surface the error and exit the runner.
			if err != nil {
				logger.Error("readiness-watcher-stopped", err)
			}
			return err
		}
	})
}
