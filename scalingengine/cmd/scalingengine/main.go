package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/schedule"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/server"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
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
		_, _ = fmt.Fprintln(os.Stderr, "missing config file")
		os.Exit(1)
	}

	configFile, err := os.Open(path)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to open config file '%s' : %s\n", path, err.Error())
		os.Exit(1)
	}

	conf, err := config.LoadConfig(configFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to read config file '%s' : %s\n", path, err.Error())
		os.Exit(1)
	}
	_ = configFile.Close()

	err = conf.Validate()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to validate configuration : %s\n", err.Error())
		os.Exit(1)
	}

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "scalingengine")

	//nolint:staticcheck //TODO https://github.com/cloudfoundry/app-autoscaler-release/issues/549
	cfhttp.Initialize(conf.HttpClientTimeout)

	eClock := clock.NewClock()
	cfClient := cf.NewCFClient(&conf.CF, logger.Session("cf"), eClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"API": conf.CF.API})
		os.Exit(1)
	}

	policyDB, err := sqldb.NewPolicySQLDB(conf.DB.PolicyDB, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"dbConfig": conf.DB.PolicyDB})
		os.Exit(1)
	}
	defer func() { _ = policyDB.Close() }()

	scalingEngineDB, err := sqldb.NewScalingEngineSQLDB(conf.DB.ScalingEngineDB, logger.Session("scalingengine-db"))
	if err != nil {
		logger.Error("failed to connect scalingengine database", err, lager.Data{"dbConfig": conf.DB.ScalingEngineDB})
		os.Exit(1)
	}
	defer func() { _ = scalingEngineDB.Close() }()

	schedulerDB, err := sqldb.NewSchedulerSQLDB(conf.DB.SchedulerDB, logger.Session("scheduler-db"))
	if err != nil {
		logger.Error("failed to connect scheduler database", err, lager.Data{"dbConfig": conf.DB.SchedulerDB})
		os.Exit(1)
	}
	defer func() { _ = schedulerDB.Close() }()

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "scalingengine")
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "scalingengine", "policyDB", policyDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "scalingengine", "scalingengineDB", scalingEngineDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "scalingengine", "schedulerDB", schedulerDB),
		httpStatusCollector,
	}, true, logger.Session("scalingengine-prometheus"))

	scalingEngine := scalingengine.NewScalingEngine(logger, cfClient, policyDB, scalingEngineDB, eClock, conf.DefaultCoolDownSecs, conf.LockSize)
	synchronizer := schedule.NewActiveScheduleSychronizer(logger, schedulerDB, scalingEngineDB, scalingEngine)

	httpServer, err := server.NewServer(logger.Session("http-server"), conf, scalingEngineDB, scalingEngine, synchronizer, httpStatusCollector)
	if err != nil {
		logger.Error("failed to create http server", err)
		os.Exit(1)
	}

	healthServer, err := healthendpoint.NewServerWithBasicAuth(conf.Health, []healthendpoint.Checker{}, logger.Session("health-server"), promRegistry, time.Now)
	if err != nil {
		logger.Error("failed to create health server", err)
		os.Exit(1)
	}

	members := grouper.Members{
		{"http_server", httpServer},
		{"health_server", healthServer},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("started")
	err = <-monitor.Wait()
	if err != nil {
		logger.Error("http-server-exited-with-failure", err)
		os.Exit(1)
	}
	logger.Info("exited")
}
