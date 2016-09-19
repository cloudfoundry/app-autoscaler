package main

import (
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/eventgenerator/aggregator"
	"autoscaler/eventgenerator/config"
	"autoscaler/eventgenerator/generator"
	"autoscaler/eventgenerator/model"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"flag"
	"fmt"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
	"io/ioutil"
	"os"
)

func main() {
	var path string
	flag.StringVar(&path, "c", "", "config file")
	flag.Parse()
	if path == "" {
		fmt.Fprintln(os.Stderr, "missing config file\nUsage:use '-c' option to specify the config file path")
		os.Exit(1)
	}

	var conf *config.Config

	configFile, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to open config file '%s' : %s\n", path, err.Error())
		os.Exit(1)
	}
	configFileBytes, readError := ioutil.ReadAll(configFile)
	configFile.Close()
	if readError != nil {
		fmt.Fprintf(os.Stdout, "failed to read data from config file:%s", readError.Error())
		os.Exit(1)
	}
	conf, err = config.LoadConfig(configFileBytes)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to read config file '%s' : %s\n", path, err.Error())
		os.Exit(1)
	}

	err = conf.Validate()
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to validate configuration : %s\n", err.Error())
		os.Exit(1)
	}

	logger := initLoggerFromConfig(&conf.Logging)
	egClock := clock.NewClock()

	var appMetricDB db.AppMetricDB
	appMetricDB, err = sqldb.NewAppMetricSQLDB(conf.DB.AppMetricDBUrl, logger.Session("appMetric-db"))
	if err != nil {
		logger.Error("failed to connect app-metric database", err, lager.Data{"url": conf.DB.AppMetricDBUrl})
		os.Exit(1)
	}
	defer appMetricDB.Close()

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.DB.PolicyDBUrl, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"url": conf.DB.PolicyDBUrl})
		os.Exit(1)
	}
	defer policyDB.Close()
	triggerArrayChan := make(chan []*model.Trigger, conf.Evaluator.TriggerArrayChannelSize)
	appMonitorChan := make(chan *model.AppMonitor, conf.Aggregator.AppMonitorChannelSize)

	eventGeneratorServer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		evaluationManager := generator.NewAppEvaluationManager(conf.Evaluator.EvaluationManagerInterval, logger, egClock, triggerArrayChan, conf.Evaluator.EvaluatorCount, appMetricDB, conf.ScalingEngine.ScalingEngineUrl)
		aggregator := aggregator.NewAggregator(logger, egClock, conf.Aggregator.PolicyPollerInterval, policyDB, appMetricDB, conf.MetricCollector.MetricCollectorUrl, conf.Aggregator.MetricPollerCount, evaluationManager, appMonitorChan)
		evaluationManager.Start()
		aggregator.Start()
		close(ready)

		<-signals
		aggregator.Stop()
		evaluationManager.Stop()

		return nil
	})
	members := grouper.Members{
		{"eventGeneratorServer", eventGeneratorServer},
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
	logger := lager.NewLogger("eventgenerator")
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
