package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"autoscaler/db"
	"autoscaler/db/sqldb"
	. "autoscaler/pruner"
	"autoscaler/pruner/config"

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
	prClock := clock.NewClock()

	metricsDb, err := sqldb.NewMetricsSQLDB(conf.Db.MetricsDbUrl, logger.Session("metrics-db"))
	if err != nil {
		logger.Error("failed to connect metrics db", err, lager.Data{"url": conf.Db.MetricsDbUrl})
		os.Exit(1)
	}
	defer metricsDb.Close()

	metricsDbPruner := createMetricsDbPrunerRunner(conf, metricsDb, prClock, logger)

	appMetricsDb, err := sqldb.NewAppMetricSQLDB(conf.Db.AppMetricsDbUrl, logger.Session("appmetrics-db"))
	if err != nil {
		logger.Error("failed to connect app metrics db", err, lager.Data{"url": conf.Db.AppMetricsDbUrl})
		os.Exit(1)
	}
	defer appMetricsDb.Close()

	appMetricsDbPruner := createAppMetricsDbPrunerRunner(conf, appMetricsDb, prClock, logger)

	members := grouper.Members{
		{"metricsdb_pruner", metricsDbPruner},
		{"appmetricsdb_pruner", appMetricsDbPruner},
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

func createMetricsDbPrunerRunner(conf *config.Config, metricsDb db.MetricsDB, prClock clock.Clock, logger lager.Logger) ifrit.Runner {
	interval := time.Duration(conf.Pruner.MetricsDbPruner.RefreshIntervalInHours) * time.Hour

	metricsDbPruner := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		mdp := NewMetricsDbPruner(logger, metricsDb, interval, conf.Pruner.MetricsDbPruner.CutoffDays, prClock)
		mdp.Start()

		close(ready)

		<-signals
		mdp.Stop()

		return nil
	})

	return metricsDbPruner
}

func createAppMetricsDbPrunerRunner(conf *config.Config, appMetricsDb db.AppMetricDB, prClock clock.Clock, logger lager.Logger) ifrit.Runner {
	interval := time.Duration(conf.Pruner.AppMetricsDbPruner.RefreshIntervalInHours) * time.Hour

	appMetricsDbPruner := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		amdp := NewAppMetricsDbPruner(logger, appMetricsDb, interval, conf.Pruner.AppMetricsDbPruner.CutoffDays, prClock)
		amdp.Start()

		close(ready)

		<-signals
		amdp.Stop()

		return nil
	})

	return appMetricsDbPruner
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
