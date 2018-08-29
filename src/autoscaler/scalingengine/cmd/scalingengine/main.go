package main

import (
	"autoscaler/helpers"
	"flag"
	"fmt"
	"os"
	"time"

	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/healthendpoint"
	"autoscaler/scalingengine"
	"autoscaler/scalingengine/config"
	"autoscaler/scalingengine/schedule"
	"autoscaler/scalingengine/server"

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

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "scalingengine")

	promRegistry := prometheus.NewRegistry()
	healthRegistry := healthendpoint.New(promRegistry, map[string]prometheus.Gauge{
		// Number of concurrent http request
		"concurrentHTTPReq": prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "autoscaler",
				Subsystem: "scalingengine",
				Name:      "concurrentHTTPReq",
				Help:      "Number of concurrent http request",
			},
		),
		"openConnection_policyDB": prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "autoscaler",
				Subsystem: "scalingengine",
				Name:      "openConnection_policyDB",
				Help:      "Number of open connection to policy DB",
			},
		),
		"openConnection_scalingEngineDB": prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "autoscaler",
				Subsystem: "scalingengine",
				Name:      "openConnection_scalingEngineDB",
				Help:      "Number of open connection to scaling engine DB",
			},
		),
		"openConnection_schedulerDB": prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "autoscaler",
				Subsystem: "scalingengine",
				Name:      "openConnection_schedulerDB",
				Help:      "Number of open connection to scheduler DB",
			},
		),
	}, true, logger.Session("scalingengine-prometheus"))

	cfhttp.Initialize(5 * time.Second)

	eClock := clock.NewClock()
	cfClient := cf.NewCFClient(&conf.CF, logger.Session("cf"), eClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"API": conf.CF.API})
		os.Exit(1)
	}

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.DB.PolicyDB, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"dbConfig": conf.DB.PolicyDB})
		os.Exit(1)
	}
	policyDB.EmitHealthMetrics(healthRegistry, eClock, conf.Health.EmitInterval)
	defer policyDB.Close()

	var scalingEngineDB db.ScalingEngineDB
	scalingEngineDB, err = sqldb.NewScalingEngineSQLDB(conf.DB.ScalingEngineDB, logger.Session("scalingengine-db"))
	if err != nil {
		logger.Error("failed to connect scalingengine database", err, lager.Data{"dbConfig": conf.DB.ScalingEngineDB})
		os.Exit(1)
	}
	scalingEngineDB.EmitHealthMetrics(healthRegistry, eClock, conf.Health.EmitInterval)
	defer scalingEngineDB.Close()

	var schedulerDB db.SchedulerDB
	schedulerDB, err = sqldb.NewSchedulerSQLDB(conf.DB.SchedulerDB, logger.Session("scheduler-db"))
	if err != nil {
		logger.Error("failed to connect scheduler database", err, lager.Data{"dbConfig": conf.DB.SchedulerDB})
		os.Exit(1)
	}
	schedulerDB.EmitHealthMetrics(healthRegistry, eClock, conf.Health.EmitInterval)
	defer schedulerDB.Close()

	scalingEngine := scalingengine.NewScalingEngine(logger, cfClient, policyDB, scalingEngineDB, eClock, conf.DefaultCoolDownSecs, conf.LockSize)
	synchronizer := schedule.NewActiveScheduleSychronizer(logger, schedulerDB, scalingEngineDB, scalingEngine)

	httpServer, err := server.NewServer(logger.Session("http-server"), conf, scalingEngineDB, scalingEngine, synchronizer, healthRegistry)
	if err != nil {
		logger.Error("failed to create http server", err)
		os.Exit(1)
	}

	healthServer, err := healthendpoint.NewServer(logger.Session("health-server"), conf.Health.Port, promRegistry)
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
