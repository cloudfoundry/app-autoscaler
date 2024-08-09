package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/sync"
	"github.com/google/uuid"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	var path string
	flag.StringVar(&path, "c", "", "config file")
	flag.Parse()
	if path == "" {
		fmt.Fprintln(os.Stderr, "missing config file")
		os.Exit(1)
	}

	configFile, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to open config file '%s' : %s\n", path, err.Error())
		os.Exit(1)
	}

	var conf *config.Config
	conf, err = config.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to read config file '%s' : %s\n", path, err.Error())
		os.Exit(1)
	}
	configFile.Close()

	err = conf.Validate()
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to validate configuration : %s\n", err.Error())
		os.Exit(1)
	}

	helpers.SetupOpenTelemetry()

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "operator")
	prClock := clock.NewClock()

	appMetricsDB, err := sqldb.NewAppMetricSQLDB(conf.AppMetricsDB.DB, logger.Session("appmetrics-db"))
	if err != nil {
		logger.Error("failed to connect appmetrics db", err, lager.Data{"dbConfig": conf.AppMetricsDB.DB})
		os.Exit(1)
	}
	defer appMetricsDB.Close()

	scalingEngineDB, err := sqldb.NewScalingEngineSQLDB(conf.ScalingEngineDB.DB, logger.Session("scalingengine-db"))
	if err != nil {
		logger.Error("failed to connect scalingengine db", err, lager.Data{"dbConfig": conf.ScalingEngineDB.DB})
		os.Exit(1)
	}
	defer scalingEngineDB.Close()

	cfClient := cf.NewCFClient(&conf.CF, logger.Session("cf"), prClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"API": conf.CF.API})
		os.Exit(1)
	}

	policyDb := sqldb.CreatePolicyDb(conf.AppSyncer.DB, logger)
	defer func() { _ = policyDb.Close() }()

	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "operator", "policyDB", policyDb),

		healthendpoint.NewDatabaseStatusCollector("autoscaler", "operator", "appMetricsDB", appMetricsDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "operator", "scalingEngineDB", scalingEngineDB),
	}, true, logger.Session("operator-prometheus"))

	scalingEngineHttpclient, err := helpers.CreateHTTPClient(&conf.ScalingEngine.TLSClientCerts, helpers.DefaultClientConfig(), logger.Session("scaling_client"))
	if err != nil {
		logger.Error("failed to create http client for scalingengine", err, lager.Data{"scalingengineTLS": conf.ScalingEngine.TLSClientCerts})
		os.Exit(1)
	}
	schedulerHttpclient, err := helpers.CreateHTTPClient(&conf.Scheduler.TLSClientCerts, helpers.DefaultClientConfig(), logger.Session("scheduler_client"))
	if err != nil {
		logger.Error("failed to create http client for scheduler", err, lager.Data{"schedulerTLS": conf.Scheduler.TLSClientCerts})
		os.Exit(1)
	}

	loggerSessionName := "appmetrics-dbpruner"
	appMetricsDBPruner := operator.NewAppMetricsDbPruner(appMetricsDB, conf.AppMetricsDB.CutoffDuration, prClock, logger.Session(loggerSessionName))
	appMetricsDBOperatorRunner := operator.NewOperatorRunner(appMetricsDBPruner, conf.AppMetricsDB.RefreshInterval, prClock, logger.Session(loggerSessionName))

	loggerSessionName = "scalingengine-dbpruner"
	scalingEngineDBPruner := operator.NewScalingEngineDbPruner(scalingEngineDB, conf.ScalingEngineDB.CutoffDuration, prClock, logger.Session(loggerSessionName))
	scalingEngineDBOperatorRunner := operator.NewOperatorRunner(scalingEngineDBPruner, conf.ScalingEngineDB.RefreshInterval, prClock, logger.Session(loggerSessionName))
	loggerSessionName = "scalingengine-sync"
	scalingEngineSync := operator.NewScheduleSynchronizer(scalingEngineHttpclient, conf.ScalingEngine.URL, prClock, logger.Session(loggerSessionName))
	scalingEngineSyncRunner := operator.NewOperatorRunner(scalingEngineSync, conf.ScalingEngine.SyncInterval, prClock, logger.Session(loggerSessionName))

	loggerSessionName = "scheduler-sync"
	schedulerSync := operator.NewScheduleSynchronizer(schedulerHttpclient, conf.Scheduler.URL, prClock, logger.Session(loggerSessionName))
	schedulerSyncRunner := operator.NewOperatorRunner(schedulerSync, conf.Scheduler.SyncInterval, prClock, logger.Session(loggerSessionName))

	loggerSessionName = "application-sync"
	applicationSync := operator.NewApplicationSynchronizer(cfClient.GetCtxClient(), policyDb, logger.Session(loggerSessionName))
	applicationSyncRunner := operator.NewOperatorRunner(applicationSync, conf.AppSyncer.SyncInterval, prClock, logger.Session(loggerSessionName))

	members := grouper.Members{
		{"appmetrics-dbpruner", appMetricsDBOperatorRunner},
		{"scalingEngine-dbpruner", scalingEngineDBOperatorRunner},
		{"scalingEngine-sync", scalingEngineSyncRunner},
		{"scheduler-sync", schedulerSyncRunner},
		{"application-sync", applicationSyncRunner},
	}

	guid := uuid.NewString()
	const lockTableName = "operator_lock"
	var lockDB db.LockDB
	lockDB, err = sqldb.NewLockSQLDB(conf.DBLock.DB, lockTableName, logger.Session("lock-db"))
	if err != nil {
		logger.Error("failed-to-connect-lock-database", err, lager.Data{"dbConfig": conf.DBLock.DB})
		os.Exit(1)
	}
	defer lockDB.Close()
	prdl := sync.NewDatabaseLock(logger)
	dbLockMaintainer := prdl.InitDBLockRunner(conf.DBLock.LockRetryInterval, conf.DBLock.LockTTL, guid, lockDB, func() {}, func() {
		os.Exit(1)
	})
	members = append(grouper.Members{{"db-lock-maintainer", dbLockMaintainer}}, members...)

	healthServer, err := healthendpoint.NewServerWithBasicAuth(conf.Health, []healthendpoint.Checker{}, logger.Session("health-server"), promRegistry, time.Now)
	if err != nil {
		logger.Error("failed to create health server", err)
		os.Exit(1)
	}
	members = append(grouper.Members{{"health_server", healthServer}}, members...)

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))

	logger.Info("started")

	err = <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}
