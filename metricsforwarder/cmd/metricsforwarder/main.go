package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/manager"
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

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "metricsforwarder")
	mfClock := clock.NewClock()

	policyDb := sqldb.CreatePolicyDb(conf.Db[db.PolicyDb], logger)
	defer func() { _ = policyDb.Close() }()

	credentialProvider := cred_helper.CredentialsProvider(conf.CredHelperImpl, conf.StoredProcedureConfig, conf.Db, conf.CacheTTL, conf.CacheCleanupInterval, logger, policyDb)
	defer func() { _ = credentialProvider.Close() }()

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "metricsforwarder")

	allowedMetricCache := cache.New(conf.CacheTTL, conf.CacheCleanupInterval)
	customMetricsServer := createCustomMetricsServer(conf, logger, policyDb, credentialProvider, allowedMetricCache, httpStatusCollector)
	cacheUpdater := cacheUpdater(logger, mfClock, conf, policyDb, allowedMetricCache)
	healthServer := createHealthServer(policyDb, credentialProvider, logger, conf, createPrometheusRegistry(policyDb, httpStatusCollector, logger))

	members := grouper.Members{
		{"cacheUpdater", cacheUpdater},
		{"custom_metrics_server", customMetricsServer},
		{"health_server", healthServer},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))

	logger.Info("started")

	err = <-monitor.Wait()
	if err != nil {
		logger.Fatal("Exited with Error", err)
		os.Exit(1)
	}
	logger.Info("exited")
}

func createPrometheusRegistry(policyDB *sqldb.PolicySQLDB, httpStatusCollector healthendpoint.HTTPStatusCollector, logger lager.Logger) *prometheus.Registry {
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricsforwarder", "policyDB", policyDB),
		httpStatusCollector,
	}, true, logger.Session("metricsforwarder-prometheus"))
	return promRegistry
}

func cacheUpdater(logger lager.Logger, mfClock clock.Clock, conf *config.Config, policyDB *sqldb.PolicySQLDB, allowedMetricCache *cache.Cache) ifrit.RunFunc {
	policyManager := manager.NewPolicyManager(logger, mfClock, conf.PolicyPollerInterval, policyDB, *allowedMetricCache, conf.CacheTTL)
	cacheUpdater := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		policyManager.Start()
		close(ready)
		<-signals
		policyManager.Stop()
		return nil
	})
	return cacheUpdater
}

func createCustomMetricsServer(conf *config.Config, logger lager.Logger, policyDB *sqldb.PolicySQLDB, credentialProvider cred_helper.Credentials, allowedMetricCache *cache.Cache, httpStatusCollector healthendpoint.HTTPStatusCollector) ifrit.Runner {
	rateLimiter := ratelimiter.DefaultRateLimiter(conf.RateLimit.MaxAmount, conf.RateLimit.ValidDuration, logger.Session("metricforwarder-ratelimiter"))
	httpServer, err := server.NewServer(logger.Session("custom_metrics_server"), conf, policyDB, credentialProvider, *allowedMetricCache, httpStatusCollector, rateLimiter)
	if err != nil {
		logger.Fatal("Failed to create client to custom metrics server", err)
		os.Exit(1)
	}
	return httpServer
}

func createHealthServer(policyDB *sqldb.PolicySQLDB, credDb cred_helper.Credentials, logger lager.Logger, conf *config.Config, promRegistry *prometheus.Registry) ifrit.Runner {
	checkers := []healthendpoint.Checker{healthendpoint.DbChecker(db.PolicyDb, policyDB), healthendpoint.DbChecker(db.StoredProcedureDb, credDb)}
	healthServer, err := healthendpoint.NewServerWithBasicAuth(conf.Health, checkers, logger.Session("health-server"), promRegistry, time.Now)
	if err != nil {
		logger.Fatal("Failed to create health server:", err)
		os.Exit(1)
	}
	return healthServer
}
