package main

import (
	"flag"
	"fmt"
	"os"

	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/syncer"
	"autoscaler/syncer/config"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
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

	var policyDB db.PolicyDB

	policyDB, err = sqldb.NewPolicySQLDB(conf.Db.PolicyDbUrl, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"url": conf.Db.PolicyDbUrl})
		os.Exit(1)
	}
	defer policyDB.Close()

	var schedulerDB db.SchedulerDB
	schedulerDB, err = sqldb.NewSchedulerSQLDB(conf.Db.SchedulerDbUrl, logger.Session("scheduler-db"))
	if err != nil {
		logger.Error("failed to connect scheduler database", err, lager.Data{"url": conf.Db.SchedulerDbUrl})
		os.Exit(1)
	}
	defer schedulerDB.Close()

	srClock := clock.NewClock()

	activeScheduleSyncer := syncer.NewActiveScheduleSyncer(policyDB, schedulerDB, logger.Session("activeScheduleSyncer"))
	syncerRunner := syncer.NewSyncerRunner(activeScheduleSyncer, conf.SynchronizeInterval, srClock, logger.Session("syncerRunner"))

	members := grouper.Members{
		{"syncerRunner", syncerRunner},
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
	logger := lager.NewLogger("syncer")
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
