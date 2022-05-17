package main

import (
	"flag"
	"fmt"
	"os"

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

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "metricsforwarder")
	mfClock := clock.NewClock()

	policyDB, err := sqldb.NewPolicySQLDB(conf.Db[db.PolicyDb], logger.Session("policy-db"))
	if err != nil {
		logger.Fatal("Failed To connect to policyDB", err, lager.Data{"dbConfig": conf.Db[db.PolicyDb]})
		os.Exit(1)
	}
	defer func() { _ = policyDB.Close() }()

	credentialProvider, storedProcedureDb, err := credentialsProvider(conf, logger, err, policyDB)
	if err != nil {
		logger.Fatal("Failed to connect to storedProcedureDb", err)
		os.Exit(1)
	}

	defer func() {
		if storedProcedureDb != nil {
			_ = storedProcedureDb.Close()
		}
	}()

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "metricsforwarder")
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricsforwarder", "policyDB", policyDB),
		httpStatusCollector,
	}, true, logger.Session("metricsforwarder-prometheus"))

	allowedMetricCache := cache.New(conf.CacheTTL, conf.CacheCleanupInterval)

	rateLimiter := ratelimiter.DefaultRateLimiter(conf.RateLimit.MaxAmount, conf.RateLimit.ValidDuration, logger.Session("metricforwarder-ratelimiter"))
	httpServer, err := server.NewServer(logger.Session("custom_metrics_server"), conf, policyDB, credentialProvider, *allowedMetricCache, httpStatusCollector, rateLimiter)
	if err != nil {
		logger.Fatal("Failed to create client to custom metrics server", err)
		os.Exit(1)
	}

	policyManager := manager.NewPolicyManager(logger, mfClock, conf.PolicyPollerInterval, policyDB, *allowedMetricCache, conf.CacheTTL)

	cacheUpdater := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		policyManager.Start()
		close(ready)
		<-signals
		policyManager.Stop()
		return nil
	})

	checkers := []healthendpoint.Checker{healthendpoint.DbChecker("policyDb", policyDB), healthendpoint.DbChecker("storedProcedureDb", storedProcedureDb)}

	healthServer, err := healthendpoint.NewServerWithBasicAuth(
		checkers,
		logger.Session("health-server"),
		conf.Health.Port, promRegistry,
		conf.Health.HealthCheckUsername,
		conf.Health.HealthCheckPassword,
		conf.Health.HealthCheckUsernameHash,
		conf.Health.HealthCheckPasswordHash,
	)
	if err != nil {
		logger.Fatal("Failed to create health server:", err)
		os.Exit(1)
	}

	members := grouper.Members{
		{"cacheUpdater", cacheUpdater},
		{"custom_metrics_server", httpServer},
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

func credentialsProvider(conf *config.Config, logger lager.Logger, err error, policyDB db.PolicyDB) (cred_helper.Credentials, db.StoredProcedureDB, error) {
	var credentials cred_helper.Credentials
	var storedProcedureDb db.StoredProcedureDB
	switch conf.CredHelperImpl {
	case "stored_procedure":
		if conf.StoredProcedureConfig == nil {
			logger.Error("cannot create a storedProcedureCredHelper without StoredProcedureConfig", err, lager.Data{"dbConfig": conf.Db[db.StoredProcedureDb]})
			os.Exit(1)
		}
		storedProcedureDb, err = sqldb.NewStoredProcedureSQLDb(*conf.StoredProcedureConfig, conf.Db[db.StoredProcedureDb], logger.Session("storedprocedure-db"))
		if err != nil {
			logger.Error("failed to connect to storedProcedureDb database", err, lager.Data{"dbConfig": conf.Db[db.StoredProcedureDb]})
			os.Exit(1)
		}
		credentials = cred_helper.NewStoredProcedureCredHelper(storedProcedureDb, cred_helper.MaxRetry, logger.Session("storedprocedure-cred-helper"))
	default:
		credentialCache := cache.New(conf.CacheTTL, conf.CacheCleanupInterval)
		credentials = cred_helper.NewCustomMetricsCredHelperWithCache(policyDB, cred_helper.MaxRetry, *credentialCache, conf.CacheTTL, logger)
	}
	return credentials, storedProcedureDb, err
}
