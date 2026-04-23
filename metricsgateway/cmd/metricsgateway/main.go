package main

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/emitter"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"
	"github.com/tedsuo/ifrit"
)

func main() {
	conf, logger := startup.Bootstrap("metricsgateway", config.LoadConfig)

	syslogEmitter, err := emitter.NewSyslogEmitter(logger, conf)
	startup.ExitOnError(err, logger, "failed-to-create-syslog-emitter")

	srv := server.NewServer(logger.Session("http-server"), conf, syslogEmitter)
	xm := auth.NewMultiOrgXfccAuthMiddleware(logger, conf.ValidOrgGuids, "")

	startup.StartService(logger,
		startup.Server("cf_server", func() (ifrit.Runner, error) { return srv.CreateCFServer(xm) }),
		startup.Server("health_server", srv.CreateHealthServer),
	)
}
