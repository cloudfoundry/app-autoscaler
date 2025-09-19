package main

import (
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/manager"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/patrickmn/go-cache"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
)

func main() {
	path := startup.ParseFlags()

	vcapConfiguration, _ := startup.LoadVCAPConfiguration()

	conf, err := startup.LoadAndValidateConfig(path, vcapConfiguration, config.LoadConfig)
	if err != nil {
		os.Exit(1)
	}

	startup.SetupEnvironment()

	logger := startup.InitLogger(&conf.Logging, "metricsforwarder")
	mfClock := clock.NewClock()

	policyDb := sqldb.CreatePolicyDb(conf.Db[db.PolicyDb], logger)
	defer func() { _ = policyDb.Close() }()

	bindingDB := sqldb.CreateBindingDB(conf.Db[db.BindingDb], logger)
	defer func() { _ = bindingDB.Close() }()

	credentialProvider := cred_helper.CredentialsProvider(conf.CredHelperImpl, conf.StoredProcedureConfig, conf.Db, conf.CacheTTL, conf.CacheCleanupInterval, logger, policyDb)
	defer func() { _ = credentialProvider.Close() }()

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "metricsforwarder")

	allowedMetricCache := cache.New(conf.CacheTTL, conf.CacheCleanupInterval)
	customMetricsServer := createCustomMetricsServer(conf, logger, policyDb, bindingDB, credentialProvider, allowedMetricCache, httpStatusCollector)
	cacheUpdater := cacheUpdater(logger, mfClock, conf, policyDb, allowedMetricCache)

	members := grouper.Members{
		{"cacheUpdater", cacheUpdater},
		{"custom_metrics_server", customMetricsServer},
	}

	err = startup.StartServices(logger, members)
	if err != nil {
		os.Exit(1)
	}
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

func createCustomMetricsServer(conf *config.Config, logger lager.Logger, policyDB *sqldb.PolicySQLDB, bindingDB *sqldb.BindingSQLDB, credentialProvider cred_helper.Credentials, allowedMetricCache *cache.Cache, httpStatusCollector healthendpoint.HTTPStatusCollector) ifrit.Runner {
	rateLimiter := ratelimiter.DefaultRateLimiter(conf.RateLimit.MaxAmount, conf.RateLimit.ValidDuration, logger.Session("metricforwarder-ratelimiter"))
	httpServer, err := server.NewServer(logger.Session("custom_metrics_server"), conf, policyDB, bindingDB, credentialProvider, *allowedMetricCache, httpStatusCollector, rateLimiter)
	startup.ExitOnError(err, logger, "Failed to create client to custom metrics server")
	return httpServer
}
