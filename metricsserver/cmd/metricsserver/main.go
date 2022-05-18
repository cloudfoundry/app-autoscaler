package main

import (
	"flag"
	"fmt"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsserver/collector"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsserver/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
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

	var conf *config.Config
	conf, err = config.LoadConfig(configFile)
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
	//nolint:staticcheck //TODO https://github.com/cloudfoundry/app-autoscaler-release/issues/549
	cfhttp.Initialize(conf.HttpClientTimeout)

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "metricsserver")
	msClock := clock.NewClock()

	var instanceMetricsDB db.InstanceMetricsDB
	instanceMetricsDB, err = sqldb.NewInstanceMetricsSQLDB(conf.DB.InstanceMetricsDB, logger.Session("instancemetrics-db"))
	if err != nil {
		logger.Error("failed to connect instancemetrics database", err)
		os.Exit(1)
	}
	defer func() { _ = instanceMetricsDB.Close() }()

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.DB.PolicyDB, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err)
		os.Exit(1)
	}
	defer func() { _ = policyDB.Close() }()

	metricsChan := make(chan *models.AppInstanceMetric, conf.Collector.MetricChannelSize)

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "metricsserver")
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricsserver", "instanceMetricsDB", instanceMetricsDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricsserver", "policyDB", policyDB),
		httpStatusCollector,
	}, true, logger.Session("metricsserver-prometheus"))

	coll := collector.NewCollector(logger.Session("metricsserver-collector"), conf.Collector.RefreshInterval, conf.Collector.CollectInterval, conf.Collector.PersistMetrics,
		conf.Collector.SaveInterval, conf.NodeIndex, len(conf.NodeAddrs), conf.Collector.MetricCacheSizePerApp, policyDB, instanceMetricsDB, msClock, metricsChan)

	envelopeChannels := make([]chan *loggregator_v2.Envelope, conf.Collector.EnvelopeProcessorCount)
	for i := 0; i < conf.Collector.EnvelopeProcessorCount; i++ {
		envelopeChannels[i] = make(chan *loggregator_v2.Envelope, conf.Collector.EnvelopeChannelSize)
	}

	envelopeProcessors, err := createEnvelopeProcessors(logger, msClock, conf, envelopeChannels, metricsChan, coll.GetAppIDs)
	if err != nil {
		logger.Error("failed to create Envelope Processors", err)
		os.Exit(1)
	}

	metricsServer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		logger.Info("starting collector", lager.Data{"NodeIndex": conf.NodeIndex, "NodeAddrs": conf.NodeAddrs})

		coll.Start()
		for _, envelopeProcessor := range envelopeProcessors {
			envelopeProcessor.Start()
		}

		close(ready)

		<-signals
		for _, envelopeProcessor := range envelopeProcessors {
			envelopeProcessor.Stop()
		}
		coll.Stop()
		return nil
	})

	wsServer, err := collector.NewWSServer(logger.Session("ws_server"), conf.Collector.TLS, conf.Collector.WSPort,
		conf.Collector.WSKeepAliveTime, envelopeChannels, httpStatusCollector)
	if err != nil {
		logger.Error("failed to create web socket server", err)
		os.Exit(1)
	}

	healthServer, err := healthendpoint.NewServerWithBasicAuth([]healthendpoint.Checker{}, logger.Session("health-server"), conf.Health.Port, promRegistry, conf.Health.HealthCheckUsername, conf.Health.HealthCheckPassword, conf.Health.HealthCheckUsernameHash, conf.Health.HealthCheckPasswordHash)
	if err != nil {
		logger.Error("failed to create health server", err)
		os.Exit(1)
	}

	serverConfig := collector.FromConfig(conf)

	httpServer, err := collector.NewServer(logger.Session("http_server"), &serverConfig, coll.QueryMetrics, httpStatusCollector)
	if err != nil {
		logger.Error("failed to create http server", err)
		os.Exit(1)
	}

	members := grouper.Members{
		{"metric_server", metricsServer},
		{"ws_server", wsServer},
		{"http_server", httpServer},
		{"health_server", healthServer},
	}
	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))

	logger.Info("started")

	err = <-monitor.Wait()
	if err != nil {
		logger.Error("exit-with-failure", err)
		os.Exit(1)
	}
	logger.Info("exit")
}

func createEnvelopeProcessors(logger lager.Logger, clock clock.Clock, conf *config.Config, envelopeChan []chan *loggregator_v2.Envelope, metricChan chan<- *models.AppInstanceMetric,
	getAppIDs func() map[string]bool) ([]collector.EnvelopeProcessor, error) {
	count := conf.Collector.EnvelopeProcessorCount
	envelopeProcessors := make([]collector.EnvelopeProcessor, count)

	for i := 0; i < count; i++ {
		envelopeProcessors[i] = collector.NewEnvelopeProcessor(logger, conf.Collector.CollectInterval, clock, i, count,
			envelopeChan[i], metricChan, getAppIDs)
	}
	return envelopeProcessors, nil
}
