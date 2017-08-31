package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/scalingengine"
	"autoscaler/scalingengine/config"
	"autoscaler/scalingengine/schedule"
	"autoscaler/scalingengine/server"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket"
	"github.com/hashicorp/consul/api"
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

	cfhttp.Initialize(5 * time.Second)

	logger := initLoggerFromConfig(&conf.Logging)
	eClock := clock.NewClock()

	cfClient := cf.NewCfClient(&conf.Cf, logger.Session("cf"), eClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"Api": conf.Cf.Api})
		os.Exit(1)
	}

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.Db.PolicyDbUrl, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"url": conf.Db.PolicyDbUrl})
		os.Exit(1)
	}
	defer policyDB.Close()

	var scalingEngineDB db.ScalingEngineDB
	scalingEngineDB, err = sqldb.NewScalingEngineSQLDB(conf.Db.ScalingEngineDbUrl, logger.Session("scalingengine-db"))
	if err != nil {
		logger.Error("failed to connect scalingengine database", err, lager.Data{"url": conf.Db.ScalingEngineDbUrl})
		os.Exit(1)
	}
	defer scalingEngineDB.Close()

	var schedulerDB db.SchedulerDB
	schedulerDB, err = sqldb.NewSchedulerSQLDB(conf.Db.SchedulerDbUrl, logger.Session("scheduler-db"))
	if err != nil {
		logger.Error("failed to connect scheduler database", err, lager.Data{"url": conf.Db.SchedulerDbUrl})
		os.Exit(1)
	}
	defer schedulerDB.Close()

	scalingEngine := scalingengine.NewScalingEngine(logger, cfClient, policyDB, scalingEngineDB, eClock, conf.DefaultCoolDownSecs)
	httpServer, err := server.NewServer(logger.Session("http-server"), conf, scalingEngineDB, scalingEngine)
	if err != nil {
		logger.Error("failed to create http server", err)
		os.Exit(1)
	}

	synchronizer := schedule.NewActiveScheduleSychronizer(logger.Session("synchronizer"), schedulerDB, scalingEngineDB, scalingEngine,
		conf.Synchronizer.ActiveScheduleSyncInterval, eClock)

	members := grouper.Members{
		{"http_server", httpServer},
		{"schedule_synchronizer", synchronizer},
	}

	if conf.Consul.Cluster != "" {
		consulClient, err := consuladapter.NewClientFromUrl(conf.Consul.Cluster)
		if err != nil {
			logger.Fatal("new consul client failed", err)
		}

		registrationRunner := initializeRegistrationRunner(logger, consulClient, conf.Server.Port, eClock)
		members = append(members, grouper.Member{"registration", registrationRunner})
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))

	logger.Info("started")

	err = <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}

func initLoggerFromConfig(conf *config.LoggingConfig) lager.Logger {
	logLevel, err := getLogLevel(conf.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %s\n", err.Error())
		os.Exit(1)
	}
	logger := lager.NewLogger("scalingengine")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, logLevel))

	return logger
}

func initializeRegistrationRunner(
	logger lager.Logger,
	consulClient consuladapter.Client,
	port int,
	clock clock.Clock) ifrit.Runner {

	registration := &api.AgentServiceRegistration{
		Name: "scalingengine",
		Port: port,
		Check: &api.AgentServiceCheck{
			TTL: "20s",
		},
	}

	return locket.NewRegistrationRunner(logger, registration, consulClient, locket.RetryInterval, clock)
}

func getLogLevel(level string) (lager.LogLevel, error) {
	switch level {
	case "debug":
		return lager.DEBUG, nil
	case "info":
		return lager.INFO, nil
	case "error":
		return lager.ERROR, nil
	case "fatal":
		return lager.FATAL, nil
	default:
		return -1, fmt.Errorf("Error: unsupported log level:%s", level)
	}
}
