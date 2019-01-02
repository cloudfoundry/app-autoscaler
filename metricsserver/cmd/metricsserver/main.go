package main

import (
	"flag"
	"fmt"
	"os"

	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/healthendpoint"
	"autoscaler/helpers"
	"autoscaler/metricsserver/collector"
	"autoscaler/metricsserver/config"
	"autoscaler/models"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
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

	cfhttp.Initialize(conf.HttpClientTimeout)

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "metricsserver")
	msClock := clock.NewClock()

	var instanceMetricsDB db.InstanceMetricsDB
	instanceMetricsDB, err = sqldb.NewInstanceMetricsSQLDB(conf.DB.InstanceMetricsDB, logger.Session("instancemetrics-db"))
	if err != nil {
		logger.Error("failed to connect instancemetrics database", err, lager.Data{"dbConfig": conf.DB.InstanceMetricsDB})
		os.Exit(1)
	}
	defer instanceMetricsDB.Close()

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.DB.PolicyDB, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"dbConfig": conf.DB.PolicyDB})
		os.Exit(1)
	}
	defer policyDB.Close()

	var metricsChan = make(chan *models.AppInstanceMetric)

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "metricsserver")
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricsserver", "instanceMetricsDB", instanceMetricsDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricsserver", "policyDB", policyDB),
		httpStatusCollector,
	}, true, logger.Session("metricsserver-prometheus"))

	ms := collector.NewCollector(logger.Session("metricsserver-collector"), conf.Collector.RefreshInterval, conf.Collector.CollectInterval, conf.Collector.IsMetricsPersistencySupported,
		conf.Collector.SaveInterval, conf.Server.NodeIndex, len(conf.Server.NodeAddrs), conf.Collector.MetricCacheSizePerApp, policyDB, instanceMetricsDB, msClock, metricsChan)

	var envelopeChannels = make([]chan *loggregator_v2.Envelope, conf.Collector.EnvelopeProcessorCount)

	getAppIDsFunc := func() map[string]bool {
		appIds, err := policyDB.GetAppIds()
		if err != nil {
			logger.Error("failed to get application ids", err)
			os.Exit(1)
		}
		return appIds
	}

	envelopeProcessors, err := createEnvelopeProcessors(logger, msClock, conf, envelopeChannels, metricsChan, getAppIDsFunc)
	if err != nil {
		logger.Error("failed to create Envelope Processors", err)
		os.Exit(1)
	}

	metricsServer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		logger.Info("starting collector", lager.Data{"NodeIndex": conf.Server.NodeIndex, "NodeAddrs": conf.Server.NodeAddrs})

		for _, envelopeProcessor := range envelopeProcessors {
			envelopeProcessor.Start()
		}

		ms.Start()

		close(ready)

		<-signals
		ms.Stop()

		return nil
	})

	httpServer, err := collector.NewWSServer(logger.Session("http_server"), conf, envelopeChannels, httpStatusCollector)
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
		{"metrics_server", metricsServer},
		{"http_server", httpServer},
		{"health_server", healthServer},
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

func createEnvelopeProcessors(logger lager.Logger, clock clock.Clock, conf *config.Config, envelopeChan []chan *loggregator_v2.Envelope, metricChan chan<- *models.AppInstanceMetric,
	getAppIDs func() map[string]bool) ([]*collector.EnvelopeProcessor, error) {
	count := conf.Collector.EnvelopeProcessorCount
	envelopeProcessors := make([]*collector.EnvelopeProcessor, count)

	for i := 0; i < count; i++ {
		envelopeProcessors[i] = collector.NewEnvelopeProcessor(logger, conf.Collector.CollectInterval, clock, conf.Server.NodeIndex, len(conf.Server.NodeAddrs),
			envelopeChan[i], metricChan, getAppIDs)
	}
	return envelopeProcessors, nil
}
