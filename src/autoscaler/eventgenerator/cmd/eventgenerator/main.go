package main

import (
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/eventgenerator"
	"autoscaler/eventgenerator/aggregator"
	"autoscaler/eventgenerator/config"
	"autoscaler/eventgenerator/generator"
	"autoscaler/models"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket"
	"github.com/hashicorp/consul/api"
	uuid "github.com/nu7hatch/gouuid"
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

	appMetricDB, err := sqldb.NewAppMetricSQLDB(conf.DB.AppMetricDBUrl, logger.Session("appMetric-db"))
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

	policyPoller := aggregator.NewPolicyPoller(logger, egClock, conf.Aggregator.PolicyPollerInterval, policyDB)

	triggersChan := make(chan []*models.Trigger, conf.Evaluator.TriggerArrayChannelSize)
	evaluators, err := createEvaluators(logger, conf, triggersChan, appMetricDB)
	if err != nil {
		logger.Error("failed to create Evaluators", err)
		os.Exit(1)
	}

	evaluationManager, err := generator.NewAppEvaluationManager(logger, conf.Evaluator.EvaluationManagerInterval, egClock,
		triggersChan, policyPoller.GetPolicies)
	if err != nil {
		logger.Error("failed to create Evaluation Manager", err)
		os.Exit(1)
	}

	appMonitorsChan := make(chan *models.AppMonitor, conf.Aggregator.AppMonitorChannelSize)
	metricPollers, err := createMetricPollers(logger, conf, appMonitorsChan, appMetricDB)
	aggregator, err := aggregator.NewAggregator(logger, egClock, conf.Aggregator.AggregatorExecuteInterval,
		appMonitorsChan, policyPoller.GetPolicies, conf.DefaultStatWindowSecs)
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

	if conf.Lock.ConsulClusterConfig != "" {
		consulClient, err := consuladapter.NewClientFromUrl(conf.Lock.ConsulClusterConfig)
		if err != nil {
			logger.Fatal("new consul client failed", err)
		}

		serviceClient := eventgenerator.NewServiceClient(consulClient, egClock)

		lockMaintainer := serviceClient.NewEventGeneratorLockRunner(
			logger,
			generateGUID(logger),
			conf.Lock.LockRetryInterval,
			conf.Lock.LockTTL,
		)
		registrationRunner := initializeRegistrationRunner(logger, consulClient, egClock)
		members = append(grouper.Members{{"lock-maintainer", lockMaintainer}}, members...)
		members = append(members, grouper.Member{"registration", registrationRunner})
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

	keyPatterns := []string{"[Pp]wd", "[Pp]ass", "[Ss]ecret", "[Tt]oken", "dbur[il]"}

	redactedSink, err := lager.NewRedactingWriterSink(os.Stdout, logLevel, keyPatterns, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create redacted sink", err.Error())
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

func createEvaluators(logger lager.Logger, conf *config.Config, triggersChan chan []*models.Trigger, database db.AppMetricDB) ([]*generator.Evaluator, error) {
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
		evaluators[i] = generator.NewEvaluator(logger, client, scalingEngineUrl, triggersChan, database, conf.DefaultBreachDurationSecs)
	}

	return evaluators, nil
}

func createMetricPollers(logger lager.Logger, conf *config.Config, appChan chan *models.AppMonitor, database db.AppMetricDB) ([]*aggregator.MetricPoller, error) {
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
		pollers[i] = aggregator.NewMetricPoller(logger, conf.MetricCollector.MetricCollectorUrl, appChan, client, database)
	}

	return pollers, nil
}

func initializeRegistrationRunner(
	logger lager.Logger,
	consulClient consuladapter.Client,
	clock clock.Clock) ifrit.Runner {
	registration := &api.AgentServiceRegistration{
		Name: "eventgenerator",
		Check: &api.AgentServiceCheck{
			TTL: "20s",
		},
	}
	return locket.NewRegistrationRunner(logger, registration, consulClient, locket.RetryInterval, clock)
}

func generateGUID(logger lager.Logger) string {
	uuid, err := uuid.NewV4()
	if err != nil {
		logger.Fatal("Couldn't generate uuid", err)
	}
	return uuid.String()
}
