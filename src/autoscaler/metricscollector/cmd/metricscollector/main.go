package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"

	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/healthendpoint"
	"autoscaler/helpers"
	"autoscaler/metricscollector/collector"
	"autoscaler/metricscollector/config"
	"autoscaler/metricscollector/server"
	"autoscaler/models"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/noaa/consumer"
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

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "metricscollector")
	mcClock := clock.NewClock()

	cfClient := cf.NewCFClient(&conf.CF, logger.Session("cf"), mcClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"Api": conf.CF.API})
		os.Exit(1)
	}

	dopplerUrl := cfClient.GetEndpoints().DopplerEndpoint
	logger.Info("create-noaa-client", map[string]interface{}{"dopplerUrl": dopplerUrl})
	tlsConfig := &tls.Config{InsecureSkipVerify: conf.CF.SkipSSLValidation}
	noaa := consumer.New(dopplerUrl, tlsConfig, nil)
	noaa.RefreshTokenFrom(cfClient)

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

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "metricscollector")
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricscollector", "policyDB", policyDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricscollector", "instanceMetricsDB", instanceMetricsDB),
		httpStatusCollector,
	}, true, logger.Session("metricscollector-prometheus"))

	var createAppCollector func(string, chan *models.AppInstanceMetric) collector.AppCollector
	if conf.Collector.CollectMethod == config.CollectMethodPolling {
		createAppCollector = func(appId string, dataChan chan *models.AppInstanceMetric) collector.AppCollector {
			return collector.NewAppPoller(logger.Session("app-poller"), appId, conf.Collector.CollectInterval, conf.Collector.MetricCacheSizePerApp, conf.Collector.PersistMetrics, cfClient, noaa, mcClock, dataChan)
		}
	} else {
		createAppCollector = func(appId string, dataChan chan *models.AppInstanceMetric) collector.AppCollector {
			noaaConsumer := consumer.New(dopplerUrl, tlsConfig, nil)
			noaaConsumer.RefreshTokenFrom(cfClient)
			return collector.NewAppStreamer(logger.Session("app-streamer"), appId, conf.Collector.CollectInterval, conf.Collector.MetricCacheSizePerApp, conf.Collector.PersistMetrics, cfClient, noaaConsumer, mcClock, dataChan)
		}
	}

	mc := collector.NewCollector(conf.Collector.RefreshInterval, conf.Collector.CollectInterval, conf.Collector.PersistMetrics, conf.Collector.SaveInterval,
		conf.Server.NodeIndex, len(conf.Server.NodeAddrs), logger.Session("collector"),
		policyDB, instanceMetricsDB, mcClock, createAppCollector)

	collectServer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		logger.Info("starting collector", lager.Data{"NodeIndex": conf.Server.NodeIndex, "NodeAddrs": conf.Server.NodeAddrs})
		mc.Start()

		close(ready)

		<-signals
		mc.Stop()

		return nil
	})

	httpServer, err := server.NewServer(logger.Session("http_server"), &conf.Server, mc.QueryMetrics, httpStatusCollector)
	if err != nil {
		logger.Error("failed to create http server", err)
		os.Exit(1)
	}

	healthServer, err := healthendpoint.NewServerWithBasicAuth(logger.Session("health-server"), conf.Health.Port, promRegistry, conf.Health.HealthCheckUsername, conf.Health.HealthCheckPassword, conf.Health.HealthCheckUsernameHash, conf.Health.HealthCheckPasswordHash)

	if err != nil {
		logger.Error("failed to create health server", err)
		os.Exit(1)
	}
	members := grouper.Members{
		{"collector", collectServer},
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
