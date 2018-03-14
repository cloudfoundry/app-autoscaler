package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"time"

	"autoscaler/cf"
	utils "autoscaler/commons"
	"autoscaler/db"
	"autoscaler/db/sqldb"
	alogger "autoscaler/logger"
	"autoscaler/metricscollector"
	"autoscaler/metricscollector/collector"
	"autoscaler/metricscollector/config"
	"autoscaler/metricscollector/server"
	"autoscaler/models"
	sync "autoscaler/sync"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
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
	tlsConfig := &tls.Config{InsecureSkipVerify: conf.Cf.SkipSSLValidation}
	noaa := consumer.New(dopplerUrl, tlsConfig, nil)
	noaa.RefreshTokenFrom(cfClient)

	var instanceMetricsDB db.InstanceMetricsDB
	instanceMetricsDB, err = sqldb.NewInstanceMetricsSQLDB(conf.Db.InstanceMetricsDbUrl, logger.Session("instancemetrics-db"))
	if err != nil {
		logger.Error("failed to connect instancemetrics database", err, lager.Data{"url": conf.Db.InstanceMetricsDbUrl})
		os.Exit(1)
	}
	defer instanceMetricsDB.Close()

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.Db.PolicyDbUrl, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"url": conf.Db.PolicyDbUrl})
		os.Exit(1)
	}
	defer policyDB.Close()

	var createAppCollector func(string, chan *models.AppInstanceMetric) collector.AppCollector
	if conf.Collector.CollectMethod == config.CollectMethodPolling {
		createAppCollector = func(appId string, dataChan chan *models.AppInstanceMetric) collector.AppCollector {
			return collector.NewAppPoller(logger.Session("app-poller"), appId, conf.Collector.CollectInterval, cfClient, noaa, mcClock, dataChan)
		}
	} else {
		createAppCollector = func(appId string, dataChan chan *models.AppInstanceMetric) collector.AppCollector {
			noaaConsumer := consumer.New(dopplerUrl, tlsConfig, nil)
			noaaConsumer.RefreshTokenFrom(cfClient)
			return collector.NewAppStreamer(logger.Session("app-streamer"), appId, conf.Collector.CollectInterval, cfClient, noaaConsumer, mcClock, dataChan)
		}
	}

	collectServer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		mc := collector.NewCollector(conf.Collector.RefreshInterval, conf.Collector.CollectInterval, conf.Collector.SaveInterval, logger.Session("collector"), policyDB, instanceMetricsDB, mcClock, createAppCollector)
		mc.Start()

		close(ready)

		<-signals
		mc.Stop()

		return nil
	})

	httpServer, err := server.NewServer(logger.Session("http_server"), conf, cfClient, noaa, instanceMetricsDB)
	if err != nil {
		logger.Error("failed to create http server", err)
		os.Exit(1)
	}

	members := grouper.Members{
		{"collector", collectServer},
		{"http_server", httpServer},
	}

	guid, err := utils.GenerateGUID(logger)
	if err != nil {
		logger.Error("failed-to-generate-guid", err)
	}
	const lockTableName = "mc_lock"
	if conf.EnableDBLock {
		logger.Debug("database-lock-feature-enabled")
		var lockDB db.LockDB
		lockDB, err = sqldb.NewLockSQLDB(conf.DBLock.LockDBURL, lockTableName, logger.Session("lock-db"))
		if err != nil {
			logger.Error("failed-to-connect-lock-database", err, lager.Data{"url": conf.DBLock.LockDBURL})
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

		serviceClient := metricscollector.NewServiceClient(consulClient, mcClock)
		if !conf.EnableDBLock {
			lockMaintainer := serviceClient.NewMetricsCollectorLockRunner(
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
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %s\n", err.Error())
		os.Exit(1)
	}
	logger := lager.NewLogger("metricscollector")

	keyPatterns := []string{"[Pp]wd", "[Pp]ass", "[Ss]ecret", "[Tt]oken"}

	redactedSink, err := alogger.NewRedactingWriterWithURLCredSink(os.Stdout, logLevel, keyPatterns, nil)
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
