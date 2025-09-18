package main

import (
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/sync"
	"github.com/google/uuid"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit/grouper"
)

func main() {
	conf, logger := startup.Bootstrap("operator", config.LoadConfig)

	prClock := clock.NewClock()

	// Database connections using startup factories
	appMetricsDB := startup.CreateAppMetricDB(conf.Db[db.AppMetricsDb], logger)
	defer func() { _ = appMetricsDB.Closer() }()

	scalingEngineDB := startup.CreateScalingEngineDB(conf.Db[db.ScalingEngineDb], logger)
	defer func() { _ = scalingEngineDB.Closer() }()

	policyDb := startup.CreatePolicyDB(conf.Db[db.PolicyDb], logger)
	defer func() { _ = policyDb.Closer() }()

	// CF Client
	cfClient := startup.CreateAndLoginCFClient(&conf.CF, logger, prClock)

	// HTTP clients
	scalingEngineHttpclient, err := helpers.CreateHTTPSClient(&conf.ScalingEngine.TLSClientCerts, helpers.DefaultClientConfig(), logger.Session("scaling_client"))
	startup.ExitOnError(err, logger, "failed to create http client for scalingengine", lager.Data{"scalingengineTLS": conf.ScalingEngine.TLSClientCerts})

	schedulerHttpclient, err := helpers.CreateHTTPSClient(&conf.Scheduler.TLSClientCerts, helpers.DefaultClientConfig(), logger.Session("scheduler_client"))
	startup.ExitOnError(err, logger, "failed to create http client for scheduler", lager.Data{"schedulerTLS": conf.Scheduler.TLSClientCerts})

	// Operator components
	loggerSessionName := "appmetrics-dbpruner"
	appMetricsDBPruner := operator.NewAppMetricsDbPruner(appMetricsDB.DB, conf.AppMetricsDb.CutoffDuration, prClock, logger.Session(loggerSessionName))
	appMetricsDBOperatorRunner := operator.NewOperatorRunner(appMetricsDBPruner, conf.AppMetricsDb.RefreshInterval, prClock, logger.Session(loggerSessionName))

	loggerSessionName = "scalingengine-dbpruner"
	scalingEngineDBPruner := operator.NewScalingEngineDbPruner(scalingEngineDB.DB, conf.ScalingEngineDb.CutoffDuration, prClock, logger.Session(loggerSessionName))
	scalingEngineDBOperatorRunner := operator.NewOperatorRunner(scalingEngineDBPruner, conf.ScalingEngineDb.RefreshInterval, prClock, logger.Session(loggerSessionName))

	loggerSessionName = "scalingengine-sync"
	scalingEngineSync := operator.NewScheduleSynchronizer(scalingEngineHttpclient, conf.ScalingEngine.URL, prClock, logger.Session(loggerSessionName))
	scalingEngineSyncRunner := operator.NewOperatorRunner(scalingEngineSync, conf.ScalingEngine.SyncInterval, prClock, logger.Session(loggerSessionName))

	loggerSessionName = "scheduler-sync"
	schedulerSync := operator.NewScheduleSynchronizer(schedulerHttpclient, conf.Scheduler.URL, prClock, logger.Session(loggerSessionName))
	schedulerSyncRunner := operator.NewOperatorRunner(schedulerSync, conf.Scheduler.SyncInterval, prClock, logger.Session(loggerSessionName))

	loggerSessionName = "application-sync"
	applicationSync := operator.NewApplicationSynchronizer(cfClient.GetCtxClient(), policyDb.DB, logger.Session(loggerSessionName))
	applicationSyncRunner := operator.NewOperatorRunner(applicationSync, conf.AppSyncer.SyncInterval, prClock, logger.Session(loggerSessionName))

	// Service members
	members := grouper.Members{
		{Name: "appmetrics-dbpruner", Runner: appMetricsDBOperatorRunner},
		{Name: "scalingEngine-dbpruner", Runner: scalingEngineDBOperatorRunner},
		{Name: "scalingEngine-sync", Runner: scalingEngineSyncRunner},
		{Name: "scheduler-sync", Runner: schedulerSyncRunner},
		{Name: "application-sync", Runner: applicationSyncRunner},
	}

	// Database lock
	guid := uuid.NewString()
	const lockTableName = "operator_lock"
	lockDB := startup.CreateLockDB(conf.Db[db.LockDb], lockTableName, logger)
	defer func() { _ = lockDB.Closer() }()

	prdl := sync.NewDatabaseLock(logger)
	dbLockMaintainer := prdl.InitDBLockRunner(conf.DBLock.LockRetryInterval, conf.DBLock.LockTTL, guid, lockDB.DB, func() {
		// Empty callback for lock acquisition - no special action needed when lock is obtained
		// The operator services will start normally once the lock is acquired
	}, func() {
		os.Exit(1)
	})

	members = append(
		grouper.Members{{Name: "db-lock-maintainer", Runner: dbLockMaintainer}},
		members...,
	)

	// Health server
	gatherer := createPrometheusRegistry(policyDb.DB, appMetricsDB.DB, scalingEngineDB.DB, logger)
	healthRouter, err := healthendpoint.NewHealthRouter(conf.Health, []healthendpoint.Checker{}, logger, gatherer, time.Now)
	startup.ExitOnError(err, logger, "failed to create health router")

	healthServer, err := helpers.NewHTTPServer(logger, conf.Health.ServerConfig, healthRouter)
	startup.ExitOnError(err, logger, "failed to create health server")

	members = append(
		grouper.Members{{Name: "health_server", Runner: healthServer}},
		members...,
	)

	// Start services
	err = startup.StartServices(logger, members)
	if err != nil {
		os.Exit(1)
	}
}

func createPrometheusRegistry(policyDB db.PolicyDB, appMetricsDB db.AppMetricDB, scalingEngineDB db.ScalingEngineDB, logger lager.Logger) *prometheus.Registry {
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "operator", "policyDB", policyDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "operator", "appMetricsDB", appMetricsDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "operator", "scalingEngineDB", scalingEngineDB),
	}, true, logger.Session("operator-prometheus"))
	return promRegistry
}
