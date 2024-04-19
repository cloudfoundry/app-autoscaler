package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/config"
	mgHelpers "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/go-loggregator/v9"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
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

	conf, err := loadConfig(path)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		os.Exit(1)
	}

	helpers.SetupOpenTelemetry()

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "metricsgateway")
	gatewayClock := clock.NewClock()
	loggregatorClientTLSConfig, err := loggregator.NewEgressTLSConfig(conf.Nozzle.RLPClientTLS.CACertFile, conf.Nozzle.RLPClientTLS.CertFile, conf.Nozzle.RLPClientTLS.KeyFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		os.Exit(1)
	}

	metricServerClientTLSConfig, err := conf.Emitter.MetricsServerClientTLS.CreateClientConfig()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		os.Exit(1)
	}

	policyDb := sqldb.CreatePolicyDb(conf.AppManager.PolicyDB, logger)
	defer func() { _ = policyDb.Close() }()

	envelopeCounterCollector := healthendpoint.NewCounterCollector()

	envelopChan := make(chan *loggregator_v2.Envelope, conf.EnvelopChanSize)
	emitters := createEmitters(logger, conf.Emitter.BufferSize, gatewayClock, conf.Emitter.KeepAliveInterval, conf.MetricServerAddrs, metricServerClientTLSConfig, conf.Emitter.HandshakeTimeout, conf.Emitter.MaxSetupRetryCount, conf.Emitter.MaxCloseRetryCount, conf.Emitter.RetryDelay)
	appManager := metricsgateway.NewAppManager(logger, gatewayClock, conf.AppManager.AppRefreshInterval, policyDb)
	dispatcher := metricsgateway.NewDispatcher(logger, envelopChan, emitters)
	nozzles := createNozzles(logger, conf.NozzleCount, conf.Nozzle.ShardID, conf.Nozzle.RLPAddr, loggregatorClientTLSConfig, envelopChan, appManager.GetAppIDs, envelopeCounterCollector)
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricsgateway", "policyDB", policyDb),
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

	healthServer, err := healthendpoint.NewServerWithBasicAuth(conf.Health, []healthendpoint.Checker{}, logger.Session("health-server"), promRegistry, time.Now)

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
		return nil, fmt.Errorf("failed to open config file %q: %w", path, err)
	}

	configFileBytes, err := io.ReadAll(configFile)
	_ = configFile.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read data from config file %q: %w", path, err)
	}

	conf, err := config.LoadConfig(configFileBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	err = conf.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate configuration: %w", err)
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
		//TODO mgHelpers.NewWSHelper should be injected via a provider function because metricServerClientTLSConfig has nothing to do with the emitter only the WebSocket.
		emitter := metricsgateway.NewEnvelopeEmitter(logger, bufferSize, eclock, keepAliveInterval, mgHelpers.NewWSHelper(metricsServerAddrs[i]+routes.EnvelopePath, metricServerClientTLSConfig, handshakeTimeout, logger, maxSetupRetryCount, maxCloseRetryCount, retryDelay))
		emitters[i] = emitter
	}
	return emitters
}
