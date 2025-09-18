package main

import (
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/generator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/metric"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"
	"github.com/prometheus/client_golang/prometheus"
	circuit "github.com/rubyist/circuitbreaker"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/tedsuo/ifrit"
)

func main() {
	conf, logger := startup.Bootstrap("eventgenerator", config.LoadConfig)

	clock := clock.NewClock()

	// Database connections
	appMetricDB := startup.CreateAppMetricDB(conf.Db[db.AppMetricsDb], logger)
	defer func() { _ = appMetricDB.Closer() }()

	policyDb := startup.CreatePolicyDB(conf.Db[db.PolicyDb], logger)
	defer func() { _ = policyDb.Closer() }()

	// Setup components
	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "eventgenerator")
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "eventgenerator", "appMetricDB", appMetricDB.DB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "eventgenerator", "policyDB", policyDb.DB),
		httpStatusCollector,
	}, true, logger.Session("eventgenerator-prometheus"))

	appManager := aggregator.NewAppManager(logger, clock, *conf.Aggregator, *conf.Pool, policyDb.DB, appMetricDB.DB)
	triggersChan := make(chan []*models.Trigger, conf.Evaluator.TriggerArrayChannelSize)

	evaluationManager, err := generator.NewAppEvaluationManager(logger, conf.Evaluator.EvaluationManagerInterval, clock, triggersChan, appManager.GetPolicies, *conf.CircuitBreaker)
	startup.ExitOnError(err, logger, "failed to create Evaluation Manager")

	evaluators, err := createEvaluators(logger, conf, triggersChan, appManager.QueryAppMetrics, evaluationManager.GetBreaker, evaluationManager.SetCoolDownExpired)
	startup.ExitOnError(err, logger, "failed to create Evaluators")

	appMonitorsChan := make(chan *models.AppMonitor, conf.Aggregator.AppMonitorChannelSize)
	appMetricChan := make(chan *models.AppMetric, conf.Aggregator.AppMetricChannelSize)

	fetcherFactory := metric.NewLogCacheFetcherFactory(metric.StandardLogCacheFetcherCreator)
	metricFetcher, err := fetcherFactory.CreateFetcher(logger, *conf)
	startup.ExitOnError(err, logger, "failed to create metric fetcher")

	metricPollers, err := createMetricPollers(logger, conf, appMonitorsChan, appMetricChan, metricFetcher)
	startup.ExitOnError(err, logger, "failed to create MetricPoller")

	anAggregator, err := aggregator.NewAggregator(logger, clock, conf.Aggregator.AggregatorExecuteInterval, conf.Aggregator.SaveInterval, appMonitorsChan, appManager.GetPolicies, appManager.SaveMetricToCache, conf.DefaultStatWindowSecs, appMetricChan, appMetricDB.DB)
	startup.ExitOnError(err, logger, "failed to create Aggregator")

	eventGenerator := ifrit.RunFunc(runFunc(appManager, evaluators, evaluationManager, metricPollers, anAggregator))

	// Server setup
	eventgeneratorServer := server.NewServer(logger.Session("http_server"), conf, appMetricDB.DB, policyDb.DB, appManager.QueryAppMetrics, httpStatusCollector)
	xm := auth.NewXfccAuthMiddleware(logger, conf.CFServer.XFCC)

	// Start services
	startup.StartService(logger,
		startup.Server("eventGenerator", func() (ifrit.Runner, error) { return eventGenerator, nil }),
		startup.Server("https_server", eventgeneratorServer.CreateMtlsServer),
		startup.Server("health_server", eventgeneratorServer.CreateHealthServer),
		startup.Server("cf_server", func() (ifrit.Runner, error) { return eventgeneratorServer.CreateCFServer(xm) }),
	)
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
		return nil, err
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
