package main

import (
	"autoscaler/db"
	"autoscaler/db/sqldb"
	helpers "autoscaler/helpers"
	"autoscaler/metricsforwarder/config"
	"autoscaler/metricsforwarder/server"
	"flag"
	"fmt"
	"os"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	cache "github.com/patrickmn/go-cache"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"

	"autoscaler/metricsforwarder/manager"
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

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.Db.PolicyDb, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed-to-connect-policy-database", err, lager.Data{"dbConfig": conf.Db.PolicyDb})
		os.Exit(1)
	}
	defer policyDB.Close()

	credentialCache := cache.New(conf.CacheTTL, conf.CacheCleanupInterval)
	allowedMetricCache := cache.New(conf.CacheTTL, conf.CacheCleanupInterval)

	httpServer, err := server.NewServer(logger.Session("custom_metrics_server"), conf, policyDB, *credentialCache, *allowedMetricCache)
	if err != nil {
		logger.Error("failed-to-create-custommetrics-server", err)
		os.Exit(1)
	}

	policyManager := manager.NewPolicyManager(logger, mfClock, conf.PolicyPollerInterval,
		len(conf.Server.NodeAddrs), conf.Server.NodeIndex, policyDB, *allowedMetricCache, conf.CacheTTL)

	cacheUpdater := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		policyManager.Start()

		close(ready)

		<-signals
		policyManager.Stop()
		return nil
	})

	members := grouper.Members{
		{"cacheUpdater", cacheUpdater},
		{"custom_metrics_server", httpServer},
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
