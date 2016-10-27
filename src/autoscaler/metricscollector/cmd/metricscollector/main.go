package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"time"

	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/metricscollector/collector"
	"autoscaler/metricscollector/config"
	"autoscaler/metricscollector/server"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/noaa/consumer"
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
	mcClock := clock.NewClock()

	cfClient := cf.NewCfClient(&conf.Cf, logger.Session("cf"), mcClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"Api": conf.Cf.Api})
		os.Exit(1)
	}

	dopplerUrl := cfClient.GetEndpoints().DopplerEndpoint
	logger.Info("create-noaa-client", map[string]interface{}{"dopplerUrl": dopplerUrl})
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	noaa := consumer.New(dopplerUrl, tlsConfig, nil)
	noaa.RefreshTokenFrom(cfClient)

	var metricsDB db.MetricsDB
	metricsDB, err = sqldb.NewMetricsSQLDB(conf.Db.MetricsDbUrl, logger.Session("metrics-db"))
	if err != nil {
		logger.Error("failed to connect metrics database", err, lager.Data{"url": conf.Db.MetricsDbUrl})
		os.Exit(1)
	}
	defer metricsDB.Close()

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.Db.PolicyDbUrl, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"url": conf.Db.PolicyDbUrl})
		os.Exit(1)
	}
	defer policyDB.Close()

	createPoller := func(appId string) collector.AppPoller {
		return collector.NewAppPoller(logger.Session("app-poller"), appId, conf.Collector.PollInterval, conf.Collector.RetryTimes, cfClient, noaa, metricsDB, mcClock)
	}

	collectServer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		mc := collector.NewCollector(conf.Collector.RefreshInterval, logger.Session("collector"), policyDB, mcClock, createPoller)
		mc.Start()

		close(ready)

		<-signals
		mc.Stop()

		return nil
	})
	httpServer := server.NewServer(logger, conf.Server, cfClient, noaa, metricsDB)

	members := grouper.Members{
		{"collector", collectServer},
		{"http_server", httpServer},
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
	logger := lager.NewLogger("metricscollector")
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
