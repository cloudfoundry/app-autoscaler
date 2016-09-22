package main

import (
	"flag"
	"fmt"
	"os"

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

	metricsDB, err := sqldb.NewMetricsSQLDB(conf.Db.MetricsDbUrl, logger.Session("metrics-db"))
	if err != nil {
		logger.Error("failed to connect metrics db", err, lager.Data{"url": conf.Db.MetricsDbUrl})
		os.Exit(1)
	}
	defer metricsDB.Close()

	metricsDBPruner := createMetricsDBPrunerRunner(conf, metricsDB, prClock, logger)

	members := grouper.Members{
		{"metricsdb_pruner", metricsDBPruner},
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

func createMetricsDBPrunerRunner(conf *config.Config, metricsDB db.MetricsDB, prClock clock.Clock, logger lager.Logger) ifrit.Runner {
	metricsDBPruner := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		mdp := NewMetricsDBPruner(logger, metricsDB, conf.Pruner.IntervalInHours, conf.Pruner.CutoffDays, prClock)
		mdp.Start()

		close(ready)

		<-signals
		mdp.Stop()

		return nil
	})

	return metricsDBPruner
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
