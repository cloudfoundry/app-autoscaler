package main

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/schedule"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"
	"code.cloudfoundry.org/clock"
	"github.com/tedsuo/ifrit"
)

func main() {
	conf, logger := startup.Bootstrap("scalingengine", config.LoadConfig)

	clock := clock.NewClock()

	// Database connections
	policyDb := startup.CreatePolicyDB(conf.Db[db.PolicyDb], logger)
	defer func() { _ = policyDb.Closer() }()

	scalingEngineDB := startup.CreateScalingEngineDB(conf.Db[db.ScalingEngineDb], logger)
	defer func() { _ = scalingEngineDB.Closer() }()

	schedulerDB := startup.CreateSchedulerDB(conf.Db[db.SchedulerDb], logger)
	defer func() { _ = schedulerDB.Closer() }()

	// CF Client
	cfClient := startup.CreateAndLoginCFClient(&conf.CF, logger, clock)

	// Business logic
	scalingEngine := scalingengine.NewScalingEngine(logger, cfClient, policyDb.DB, scalingEngineDB.DB, clock, conf.DefaultCoolDownSecs, conf.LockSize)
	synchronizer := schedule.NewActiveScheduleSychronizer(logger, schedulerDB.DB, scalingEngineDB.DB, scalingEngine)

	// Server setup
	srv := server.NewServer(logger.Session("http-server"), conf, policyDb.DB, scalingEngineDB.DB, schedulerDB.DB, scalingEngine, synchronizer)
	xm := auth.NewXfccAuthMiddleware(logger, conf.CFServer.XFCC)

	// Start services
	startup.StartService(logger,
		startup.Server("http_server", srv.CreateMtlsServer),
		startup.Server("health_server", srv.CreateHealthServer),
		startup.Server("cf_server", func() (ifrit.Runner, error) { return srv.CreateCFServer(xm) }),
	)
}
