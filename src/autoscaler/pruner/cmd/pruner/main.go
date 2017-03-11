package main

import (
	"flag"
	"fmt"
	"os"

	"autoscaler/db/sqldb"
	"autoscaler/pruner"
	"autoscaler/pruner/config"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket"
	"github.com/hashicorp/consul/api"
	uuid "github.com/nu7hatch/gouuid"
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

	logger := initLoggerFromConfig(&conf.Logging)
	prClock := clock.NewClock()

	instanceMetricsDb, err := sqldb.NewInstanceMetricsSQLDB(conf.InstanceMetricsDb.DbUrl, logger.Session("instancemetrics-db"))
	if err != nil {
		logger.Error("failed to connect instancemetrics db", err, lager.Data{"url": conf.InstanceMetricsDb.DbUrl})
		os.Exit(1)
	}
	defer instanceMetricsDb.Close()

	appMetricsDb, err := sqldb.NewAppMetricSQLDB(conf.AppMetricsDb.DbUrl, logger.Session("appmetrics-db"))
	if err != nil {
		logger.Error("failed to connect appmetrics db", err, lager.Data{"url": conf.AppMetricsDb.DbUrl})
		os.Exit(1)
	}
	defer appMetricsDb.Close()

	scalingEngineDb, err := sqldb.NewScalingEngineSQLDB(conf.ScalingEngineDb.DbUrl, logger.Session("scalingengine-db"))
	if err != nil {
		logger.Error("failed to connect scalingengine db", err, lager.Data{"url": conf.ScalingEngineDb.DbUrl})
		os.Exit(1)
	}
	defer scalingEngineDb.Close()

	prunerLoggerSessionName := "instancemetrics-dbpruner"
	instanceMetricDbPruner := pruner.NewInstanceMetricsDbPruner(instanceMetricsDb, conf.InstanceMetricsDb.CutoffDays, prClock, logger.Session(prunerLoggerSessionName))
	instanceMetricsDbPrunerRunner := pruner.NewDbPrunerRunner(instanceMetricDbPruner, conf.InstanceMetricsDb.RefreshInterval, prClock, logger.Session(prunerLoggerSessionName))

	prunerLoggerSessionName = "appmetrics-dbpruner"
	appMetricsDbPruner := pruner.NewAppMetricsDbPruner(appMetricsDb, conf.AppMetricsDb.CutoffDays, prClock, logger.Session(prunerLoggerSessionName))
	appMetricsDbPrunerRunner := pruner.NewDbPrunerRunner(appMetricsDbPruner, conf.AppMetricsDb.RefreshInterval, prClock, logger.Session(prunerLoggerSessionName))

	prunerLoggerSessionName = "scalingengine-dbpruner"
	scalingEngineDbPruner := pruner.NewScalingEngineDbPruner(scalingEngineDb, conf.ScalingEngineDb.CutoffDays, prClock, logger.Session(prunerLoggerSessionName))
	scalingEngineDbPrunerRunner := pruner.NewDbPrunerRunner(scalingEngineDbPruner, conf.ScalingEngineDb.RefreshInterval, prClock, logger.Session(prunerLoggerSessionName))

	consulClient, err := consuladapter.NewClientFromUrl(conf.Lock.ConsulClusterConfig)
	if err != nil {
		logger.Fatal("new consul client failed", err)
	}

	serviceClient := pruner.NewServiceClient(consulClient, prClock)

	lockMaintainer := serviceClient.NewPrunerLockRunner(
		logger,
		generateGUID(logger),
		conf.Lock.LockRetryInterval,
		conf.Lock.LockTTL,
	)

	registrationRunner := initializeRegistrationRunner(logger, consulClient, conf.Server.Port, prClock)

	members := grouper.Members{
		{"lock-maintainer", lockMaintainer},
		{"instancemetrics-dbpruner", instanceMetricsDbPrunerRunner},
		{"appmetrics-dbpruner", appMetricsDbPrunerRunner},
		{"scalingEngine-dbpruner", scalingEngineDbPrunerRunner},
		{"registration", registrationRunner},
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
	logger := lager.NewLogger("pruner")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, logLevel))

	return logger
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

func initializeRegistrationRunner(
	logger lager.Logger,
	consulClient consuladapter.Client,
	port int,
	clock clock.Clock) ifrit.Runner {
	registration := &api.AgentServiceRegistration{
		Name: "pruner",
		Port: port,
		Check: &api.AgentServiceCheck{
			TTL: "20s",
		},
	}
	return locket.NewRegistrationRunner(logger, registration, consulClient, locket.RetryInterval, clock)
}

func generateGUID(logger lager.Logger) string {
	uuid, err := uuid.NewV4()
	if err != nil {
		logger.Fatal("Couldn't generate uuid", err)
	}
	return uuid.String()
}
