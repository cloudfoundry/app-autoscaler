package main

import (
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/eventgenerator"
	"autoscaler/eventgenerator/aggregator"
	"autoscaler/eventgenerator/config"
	"autoscaler/eventgenerator/generator"
	"autoscaler/helpers"
	"autoscaler/models"
	sync "autoscaler/sync"

	"github.com/rubyist/circuitbreaker"

	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
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
		fmt.Fprintln(os.Stdout, "missing config file\nUsage:use '-c' option to specify the config file path")
		os.Exit(1)
	}

	conf, err := loadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		os.Exit(1)
	}

	logger := initLoggerFromConfig(&conf.Logging)
	egClock := clock.NewClock()

	appMetricDB, err := sqldb.NewAppMetricSQLDB(conf.DB.AppMetricDB, logger.Session("appMetric-db"))
	if err != nil {
		logger.Error("failed to connect app-metric database", err, lager.Data{"dbConfig": conf.DB.AppMetricDB})
		os.Exit(1)
	}
	defer appMetricDB.Close()

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.DB.PolicyDB, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"dbConfig": conf.DB.PolicyDB})
		os.Exit(1)
	}
	defer policyDB.Close()

	policyPoller := aggregator.NewPolicyPoller(logger, egClock, conf.Aggregator.PolicyPollerInterval, policyDB)

	triggersChan := make(chan []*models.Trigger, conf.Evaluator.TriggerArrayChannelSize)

	evaluationManager, err := generator.NewAppEvaluationManager(logger, conf.Evaluator.EvaluationManagerInterval, egClock,
		triggersChan, policyPoller.GetPolicies, conf.CircuitBreaker)
	if err != nil {
		logger.Error("failed to create Evaluation Manager", err)
		os.Exit(1)
	}

	evaluators, err := createEvaluators(logger, conf, triggersChan, appMetricDB, evaluationManager.GetBreaker)
	if err != nil {
		logger.Error("failed to create Evaluators", err)
		os.Exit(1)
	}

	appMonitorsChan := make(chan *models.AppMonitor, conf.Aggregator.AppMonitorChannelSize)
	appMetricChan := make(chan *models.AppMetric, conf.Aggregator.AppMetricChannelSize)
	metricPollers, err := createMetricPollers(logger, conf, appMonitorsChan, appMetricChan)
	aggregator, err := aggregator.NewAggregator(logger, egClock, conf.Aggregator.AggregatorExecuteInterval, conf.Aggregator.SaveInterval,
		appMonitorsChan, policyPoller.GetPolicies, conf.DefaultStatWindowSecs, appMetricChan, appMetricDB)
	if err != nil {
		logger.Error("failed to create Aggregator", err)
		os.Exit(1)
	}

	eventGenerator := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		policyPoller.Start()

		for _, evaluator := range evaluators {
			evaluator.Start()
		}
		evaluationManager.Start()

		for _, metricPoller := range metricPollers {
			metricPoller.Start()
		}
		aggregator.Start()

		close(ready)

		<-signals
		aggregator.Stop()
		evaluationManager.Stop()
		policyPoller.Stop()

		return nil
	})

	members := grouper.Members{
		{"eventGenerator", eventGenerator},
	}

	guid, err := helpers.GenerateGUID(logger)
	if err != nil {
		logger.Error("failed-to-generate-guid", err)
		os.Exit(1)
	}
	const lockTableName = "eg_lock"
	if conf.EnableDBLock {
		logger.Debug("database-lock-feature-enabled")
		var lockDB db.LockDB
		lockDB, err = sqldb.NewLockSQLDB(conf.DBLock.LockDB, lockTableName, logger.Session("lock-db"))
		if err != nil {
			logger.Error("failed-to-connect-lock-database", err, lager.Data{"dbConfig": conf.DBLock.LockDB})
			os.Exit(1)
		}
		defer lockDB.Close()
		mcdl := sync.NewDatabaseLock(logger)
		dbLockMaintainer := mcdl.InitDBLockRunner(conf.DBLock.LockRetryInterval, conf.DBLock.LockTTL, guid, lockDB)
		members = append(grouper.Members{{"db-lock-maintainer", dbLockMaintainer}}, members...)
	}

	if conf.Lock.ConsulClusterConfig != "" {
		consulClient, err := consuladapter.NewClientFromUrl(conf.Lock.ConsulClusterConfig)
		if err != nil {
			logger.Fatal("new consul client failed", err)
		}

		serviceClient := eventgenerator.NewServiceClient(consulClient, egClock)
		if !conf.EnableDBLock {
			lockMaintainer := serviceClient.NewEventGeneratorLockRunner(
				logger,
				guid,
				conf.Lock.LockRetryInterval,
				conf.Lock.LockTTL,
			)
			members = append(grouper.Members{{"lock-maintainer", lockMaintainer}}, members...)
		}
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
		fmt.Fprintf(os.Stdout, "failed to initialize logger: %s\n", err.Error())
		os.Exit(1)
	}
	logger := lager.NewLogger("eventgenerator")

	keyPatterns := []string{"[Pp]wd", "[Pp]ass", "[Ss]ecret", "[Tt]oken"}

	redactedSink, err := helpers.NewRedactingWriterWithURLCredSink(os.Stdout, logLevel, keyPatterns, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create redacted sink: %s", err.Error())
	}
	logger.RegisterSink(redactedSink)

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

func createEvaluators(logger lager.Logger, conf *config.Config, triggersChan chan []*models.Trigger,
	database db.AppMetricDB, getBreaker func(string) *circuit.Breaker) ([]*generator.Evaluator, error) {
	count := conf.Evaluator.EvaluatorCount
	scalingEngineUrl := conf.ScalingEngine.ScalingEngineUrl

	tlsCerts := &conf.ScalingEngine.TLSClientCerts
	if tlsCerts.CertFile == "" || tlsCerts.KeyFile == "" {
		tlsCerts = nil
	}

	client := cfhttp.NewClient()
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = count
	if tlsCerts != nil {
		tlsConfig, err := cfhttp.NewTLSConfig(tlsCerts.CertFile, tlsCerts.KeyFile, tlsCerts.CACertFile)
		if err != nil {
			return nil, err
		}
		client.Transport.(*http.Transport).TLSClientConfig = tlsConfig
	}

	evaluators := make([]*generator.Evaluator, count)
	for i := 0; i < count; i++ {
		evaluators[i] = generator.NewEvaluator(logger, client, scalingEngineUrl, triggersChan, database,
			conf.DefaultBreachDurationSecs, getBreaker)
	}

	return evaluators, nil
}

func createMetricPollers(logger lager.Logger, conf *config.Config, appChan chan *models.AppMonitor, appMetricChan chan *models.AppMetric) ([]*aggregator.MetricPoller, error) {
	tlsCerts := &conf.MetricCollector.TLSClientCerts
	if tlsCerts.CertFile == "" || tlsCerts.KeyFile == "" {
		tlsCerts = nil
	}

	count := conf.Aggregator.MetricPollerCount
	client := cfhttp.NewClient()
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = count
	if tlsCerts != nil {
		tlsConfig, err := cfhttp.NewTLSConfig(tlsCerts.CertFile, tlsCerts.KeyFile, tlsCerts.CACertFile)
		if err != nil {
			return nil, err
		}
		client.Transport.(*http.Transport).TLSClientConfig = tlsConfig
	}
	pollers := make([]*aggregator.MetricPoller, count)
	for i := 0; i < count; i++ {
		pollers[i] = aggregator.NewMetricPoller(logger, conf.MetricCollector.MetricCollectorUrl, appChan, client, appMetricChan)
	}

	return pollers, nil
}
