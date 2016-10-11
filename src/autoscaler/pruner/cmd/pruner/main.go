package main

import (
	"flag"
	"fmt"
	"os"

	"autoscaler/db/sqldb"
	"autoscaler/pruner"
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

	metricsDb, err := sqldb.NewMetricsSQLDB(conf.MetricsDb.DbUrl, logger.Session("metrics-db"))
	if err != nil {
		logger.Error("failed to connect metrics db", err, lager.Data{"url": conf.MetricsDb.DbUrl})
		os.Exit(1)
	}
	defer metricsDb.Close()

	appMetricsDb, err := sqldb.NewAppMetricSQLDB(conf.AppMetricsDb.DbUrl, logger.Session("appmetrics-db"))
	if err != nil {
		logger.Error("failed to connect app metrics db", err, lager.Data{"url": conf.AppMetricsDb.DbUrl})
		os.Exit(1)
	}
	defer appMetricsDb.Close()

	prunerLoggerSessionName := "metricsdbpruner"
	metricDbPruner := pruner.NewMetricsDbPruner(metricsDb, conf.MetricsDb.CutoffDays, prClock, logger.Session(prunerLoggerSessionName))
	metricsDbPrunerRunner := pruner.NewDbPrunerRunner(metricDbPruner, conf.MetricsDb.RefreshInterval, prClock, logger.Session(prunerLoggerSessionName))

	prunerLoggerSessionName = "appmetricsdbpruner"
	appMetricsDbPruner := pruner.NewAppMetricsDbPruner(appMetricsDb, conf.AppMetricsDb.CutoffDays, prClock, logger.Session(prunerLoggerSessionName))
	appMetricsDbPrunerRunner := pruner.NewDbPrunerRunner(appMetricsDbPruner, conf.AppMetricsDb.RefreshInterval, prClock, logger.Session(prunerLoggerSessionName))

	members := grouper.Members{
		{"metricsdbpruner", metricsDbPrunerRunner},
		{"appmetricsdbpruner", appMetricsDbPrunerRunner},
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
