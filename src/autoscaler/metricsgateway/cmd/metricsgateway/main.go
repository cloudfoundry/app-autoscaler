package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"code.cloudfoundry.org/cfhttp"

	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/healthendpoint"
	"autoscaler/helpers"
	"autoscaler/metricsgateway"
	"autoscaler/metricsgateway/config"
	mg_helpers "autoscaler/metricsgateway/helpers"
	"autoscaler/routes"

	"code.cloudfoundry.org/clock"
	loggregator "code.cloudfoundry.org/go-loggregator"
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

	conf, err := loadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		os.Exit(1)
	}
	logger := helpers.InitLoggerFromConfig(&conf.Logging, "metricsgateway")
	gatewayClock := clock.NewClock()
	loggregatorClientTLSConfig, err := loggregator.NewEgressTLSConfig(conf.Nozzle.RLPClientTLS.CACertFile, conf.Nozzle.RLPClientTLS.CertFile, conf.Nozzle.RLPClientTLS.KeyFile)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		os.Exit(1)
	}
	metricServerClientTLSConfig, err := cfhttp.NewTLSConfig(conf.Emitter.MetricsServerClientTLS.CertFile, conf.Emitter.MetricsServerClientTLS.KeyFile, conf.Emitter.MetricsServerClientTLS.CACertFile)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		os.Exit(1)
	}

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.AppManager.PolicyDB, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"dbConfig": conf.AppManager.PolicyDB})
		os.Exit(1)
	}
	defer policyDB.Close()
	envelopeCounterCollector := healthendpoint.NewCounterCollector()

	envelopChan := make(chan *loggregator_v2.Envelope, conf.EnvelopChanSize)
	emitters := createEmitters(logger, conf.Emitter.BufferSize, gatewayClock, conf.Emitter.KeepAliveInterval, conf.MetricServerAddrs, metricServerClientTLSConfig, conf.Emitter.HandshakeTimeout, conf.Emitter.MaxSetupRetryCount, conf.Emitter.MaxCloseRetryCount, conf.Emitter.RetryDelay)
	appManager := metricsgateway.NewAppManager(logger, gatewayClock, conf.AppManager.AppRefreshInterval, policyDB)
	dispatcher := metricsgateway.NewDispatcher(logger, envelopChan, emitters)
	nozzles := createNozzles(logger, conf.NozzleCount, conf.Nozzle.ShardID, conf.Nozzle.RLPAddr, loggregatorClientTLSConfig, envelopChan, appManager.GetAppIDs, envelopeCounterCollector)
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricsgateway", "policyDB", policyDB),
		healthendpoint.NewHTTPStatusCollector("autoscaler", "metricsgateway"),
		envelopeCounterCollector,
	}, true, logger.Session("metricsgateway-prometheus"))

	gatewayServer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		logger.Info("starting metricsgateway")
		appManager.Start()
		dispatcher.Start()
		for _, emitter := range emitters {
			err := emitter.Start()
			if err != nil {
				logger.Error("failed to start emitter", err)
				os.Exit(1)
			}
		}
		for _, nozzle := range nozzles {
			nozzle.Start()
		}

		close(ready)
		<-signals
		for _, nozzle := range nozzles {
			nozzle.Stop()
		}
		for _, emitter := range emitters {
			emitter.Stop()
		}
		dispatcher.Stop()
		appManager.Stop()

		return nil
	})

	healthServer, err := healthendpoint.NewServerWithBasicAuth(logger.Session("health-server"), conf.Health.Port, promRegistry, conf.Health.HealthCheckUsername, conf.Health.HealthCheckPassword, conf.Health.HealthCheckUsernameHash, conf.Health.HealthCheckPasswordHash)

	if err != nil {
		logger.Error("failed to create health server", err)
		os.Exit(1)
	}
	members := grouper.Members{
		{"metricsgateway", gatewayServer},
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

func loadConfig(path string) (*config.Config, error) {
	configFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %q: %s", path, err.Error())
	}

	configFileBytes, err := ioutil.ReadAll(configFile)
	configFile.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read data from config file %q: %s", path, err.Error())
	}

	conf, err := config.LoadConfig(configFileBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %s", path, err.Error())
	}

	err = conf.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate configuration: %s", err.Error())
	}
	return conf, nil
}

func createNozzles(logger lager.Logger, nozzleCount int, shardID string, rlpAddr string, loggregatorClientTLSConfig *tls.Config, envelopChan chan *loggregator_v2.Envelope, getAppIDsFunc metricsgateway.GetAppIDsFunc, envelopeCounterCollector healthendpoint.CounterCollector) []*metricsgateway.Nozzle {

	nozzles := make([]*metricsgateway.Nozzle, nozzleCount)
	for i := 0; i < nozzleCount; i++ {
		nozzles[i] = metricsgateway.NewNozzle(logger, i, shardID, rlpAddr, loggregatorClientTLSConfig, envelopChan, getAppIDsFunc, envelopeCounterCollector)
	}
	return nozzles
}

func createEmitters(logger lager.Logger, bufferSize int, eclock clock.Clock, keepAliveInterval time.Duration, metricsServerAddrs []string, metricServerClientTLSConfig *tls.Config, handshakeTimeout time.Duration, maxSetupRetryCount int, maxCloseRetryCount int, retryDelay time.Duration) []metricsgateway.Emitter {
	emitters := make([]metricsgateway.Emitter, len(metricsServerAddrs))
	for i := 0; i < len(metricsServerAddrs); i++ {
		emitter := metricsgateway.NewEnvelopeEmitter(logger, bufferSize, eclock, keepAliveInterval, mg_helpers.NewWSHelper(metricsServerAddrs[i]+routes.EnvelopePath, metricServerClientTLSConfig, handshakeTimeout, logger, maxSetupRetryCount, maxCloseRetryCount, retryDelay))
		emitters[i] = emitter
	}
	return emitters
}
