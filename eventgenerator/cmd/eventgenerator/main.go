package main

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/generator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/metric"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/prometheus/client_golang/prometheus"
	circuit "github.com/rubyist/circuitbreaker"

	"flag"
	"fmt"
	"os"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	var path string
	flag.StringVar(&path, "c", "", "config file")
	flag.Parse()

	vcapConfiguration, err := configutil.NewVCAPConfigurationReader()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to read vcap configuration : %s\n", err.Error())
	}

	conf, err := config.LoadConfig(path, vcapConfiguration)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to read config file '%s' : %s\n", path, err.Error())
		os.Exit(1)
	}

	err = conf.Validate()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to validate configuration : %s\n", err.Error())
		os.Exit(1)
	}

	helpers.SetupOpenTelemetry()

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "eventgenerator")

	egClock := clock.NewClock()

	appMetricDB, err := sqldb.NewAppMetricSQLDB(conf.Db[db.AppMetricsDb], logger.Session("appMetric-db"))
	if err != nil {
		logger.Error("failed to connect app-metric database", err, lager.Data{"dbConfig": conf.Db[db.AppMetricsDb]})
		os.Exit(1)
	}
	defer func() { _ = appMetricDB.Close() }()

	policyDb := sqldb.CreatePolicyDb(conf.Db[db.PolicyDb], logger)
	defer func() { _ = policyDb.Close() }()

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "eventgenerator")
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "eventgenerator", "appMetricDB", appMetricDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "eventgenerator", "policyDB", policyDb),
		httpStatusCollector,
	}, true, logger.Session("eventgenerator-prometheus"))

	//appManager := aggregator.NewAppManager(logger, egClock, conf.Aggregator.PolicyPollerInterval, conf.Pool.TotalInstances, conf.Pool.InstanceIndex, conf.Aggregator.MetricCacheSizePerApp, policyDb, appMetricDB)
	appManager := aggregator.NewAppManager(logger, egClock, *conf.Aggregator, *conf.Pool, policyDb, appMetricDB)

	triggersChan := make(chan []*models.Trigger, conf.Evaluator.TriggerArrayChannelSize)

	evaluationManager, err := generator.NewAppEvaluationManager(logger, conf.Evaluator.EvaluationManagerInterval, egClock, triggersChan, appManager.GetPolicies, *conf.CircuitBreaker)
	if err != nil {
		logger.Error("failed to create Evaluation Manager", err)
		os.Exit(1)
	}

	evaluators, err := createEvaluators(logger, conf, triggersChan, appManager.QueryAppMetrics, evaluationManager.GetBreaker, evaluationManager.SetCoolDownExpired)
	if err != nil {
		logger.Error("failed to create Evaluators", err)
		os.Exit(1)
	}

	appMonitorsChan := make(chan *models.AppMonitor, conf.Aggregator.AppMonitorChannelSize)
	appMetricChan := make(chan *models.AppMetric, conf.Aggregator.AppMetricChannelSize)

	fetcherFactory := metric.NewLogCacheFetcherFactory(metric.StandardLogCacheFetcherCreator)
	metricFetcher, err := fetcherFactory.CreateFetcher(logger, *conf)
	if err != nil {
		logger.Error("failed to create metric fetcher", err)
		os.Exit(1)
	}

	metricPollers, err := createMetricPollers(logger, conf, appMonitorsChan, appMetricChan, metricFetcher)
	if err != nil {
		logger.Error("failed to create MetricPoller", err)
		os.Exit(1)
	}
	anAggregator, err := aggregator.NewAggregator(logger, egClock, conf.Aggregator.AggregatorExecuteInterval, conf.Aggregator.SaveInterval, appMonitorsChan, appManager.GetPolicies, appManager.SaveMetricToCache, conf.DefaultStatWindowSecs, appMetricChan, appMetricDB)
	if err != nil {
		logger.Error("failed to create Aggregator", err)
		os.Exit(1)
	}

	eventGenerator := ifrit.RunFunc(runFunc(appManager, evaluators, evaluationManager, metricPollers, anAggregator))

	eventgeneratorServer := server.NewServer(
		logger.Session("http_server"), conf, appMetricDB,
		policyDb, appManager.QueryAppMetrics, httpStatusCollector)

	mtlsServer, err := eventgeneratorServer.CreateMtlsServer()
	if err != nil {
		logger.Error("failed to create http server", err)
		os.Exit(1)
	}

	healthServer, err := eventgeneratorServer.CreateHealthServer()
	if err != nil {
		logger.Error("failed to create health server", err)
		os.Exit(1)
	}

	xm := auth.NewXfccAuthMiddleware(logger, conf.CFServer.XFCC)
	cfServer, err := eventgeneratorServer.CreateCFServer(xm)
	if err != nil {
		logger.Error("failed to create cf server", err)
		os.Exit(1)
	}

	members := grouper.Members{
		{Name: "eventGenerator", Runner: eventGenerator},
		{Name: "https_server", Runner: mtlsServer},
		{Name: "health_server", Runner: healthServer},
		{Name: "cf_server", Runner: cfServer},
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

func runFunc(appManager *aggregator.AppManager, evaluators []*generator.Evaluator, evaluationManager *generator.AppEvaluationManager, metricPollers []*aggregator.MetricPoller, anAggregator *aggregator.Aggregator) func(signals <-chan os.Signal, ready chan<- struct{}) error {
	return func(signals <-chan os.Signal, ready chan<- struct{}) error {
		appManager.Start()

		for _, evaluator := range evaluators {
			evaluator.Start()
		}
		evaluationManager.Start()

		for _, metricPoller := range metricPollers {
			metricPoller.Start()
		}
		anAggregator.Start()

		close(ready)

		<-signals
		anAggregator.Stop()
		evaluationManager.Stop()
		appManager.Stop()

		return nil
	}
}

func createEvaluators(logger lager.Logger, conf *config.Config, triggersChan chan []*models.Trigger, queryMetrics aggregator.QueryAppMetricsFunc, getBreaker func(string) *circuit.Breaker, setCoolDownExpired func(string, int64)) ([]*generator.Evaluator, error) {
	count := conf.Evaluator.EvaluatorCount

	seClient, err := helpers.CreateHTTPSClient(&conf.ScalingEngine.TLSClientCerts, helpers.DefaultClientConfig(), logger.Session("scaling_client"))
	if err != nil {
		logger.Error("failed to create http client for ScalingEngine", err, lager.Data{"scalingengineTLS": conf.ScalingEngine.TLSClientCerts})
		os.Exit(1)
	}

	evaluators := make([]*generator.Evaluator, count)
	for i := range evaluators {
		evaluators[i] = generator.NewEvaluator(logger, seClient, conf.ScalingEngine.ScalingEngineURL, triggersChan,
			conf.DefaultBreachDurationSecs, queryMetrics, getBreaker, setCoolDownExpired)
	}

	return evaluators, nil
}

func createMetricPollers(logger lager.Logger, conf *config.Config, appMonitorsChan chan *models.AppMonitor, appMetricChan chan *models.AppMetric, metricClient metric.Fetcher) ([]*aggregator.MetricPoller, error) {
	pollers := make([]*aggregator.MetricPoller, conf.Aggregator.MetricPollerCount)
	for i := range pollers {
		pollers[i] = aggregator.NewMetricPoller(logger, metricClient, appMonitorsChan, appMetricChan)
	}
	return pollers, nil
}
