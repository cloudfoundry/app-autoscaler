package main

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"

	"github.com/tedsuo/ifrit"
)

func main() {
	conf, logger := startup.Bootstrap("activator", config.LoadConfig)

	activatorServer := server.NewServer(logger.Session("http_server"), conf)
	xm := auth.NewXfccAuthMiddleware(logger, conf.CFServer.XFCC)

	startup.StartService(logger,
		startup.Server("https_server", activatorServer.CreateMtlsServer),
		startup.Server("health_server", activatorServer.CreateHealthServer),
		startup.Server("cf_server", func() (ifrit.Runner, error) { return activatorServer.CreateCFServer(xm) }),
	)
}
