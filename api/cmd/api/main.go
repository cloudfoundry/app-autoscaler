package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/brokerserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/publicapiserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
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

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "api")

	members := grouper.Members{}

	var policyDb db.PolicyDB
	policyDb, err = sqldb.NewPolicySQLDB(conf.DB[db.PolicyDb], logger.Session("policydb-db"))
	if err != nil {
		logger.Error("failed to connect to policydb database", err, lager.Data{"dbConfig": conf.DB[db.PolicyDb]})
		os.Exit(1)
	}
	defer func() { _ = policyDb.Close() }()

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "golangapiserver")
	prometheusCollectors := []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "golangapiserver", "policyDB", policyDb),
		httpStatusCollector,
	}

	paClock := clock.NewClock()
	cfClient := cf.NewCFClient(&conf.CF, logger.Session("cf"), paClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"API": conf.CF.API})
		os.Exit(1)
	}

	var credentials cred_helper.Credentials
	switch conf.CredHelperImpl {
	case "stored_procedure":
		if conf.StoredProcedureConfig == nil {
			logger.Error("cannot create a storedProcedureCredHelper without StoredProcedureConfig", err, lager.Data{"dbConfig": conf.DB[db.StoredProcedureDb]})
			os.Exit(1)
		}
		var storedProcedureDb db.StoredProcedureDB
		storedProcedureDb, err = sqldb.NewStoredProcedureSQLDb(*conf.StoredProcedureConfig, conf.DB[db.StoredProcedureDb], logger.Session("storedprocedure-db"))
		if err != nil {
			logger.Error("failed to connect to storedProcedureDb database", err, lager.Data{"dbConfig": conf.DB[db.StoredProcedureDb]})
			os.Exit(1)
		}
		defer func() { _ = storedProcedureDb.Close() }()
		credentials = cred_helper.NewStoredProcedureCredHelper(storedProcedureDb, cred_helper.MaxRetry, logger.Session("storedprocedure-cred-helper"))
	default:
		credentials = cred_helper.NewCustomMetricsCredHelper(policyDb, cred_helper.MaxRetry, logger)
	}

	var checkBindingFunc api.CheckBindingFunc
	var bindingDB db.BindingDB

	if !conf.UseBuildInMode {
		bindingDB, err = sqldb.NewBindingSQLDB(conf.DB[db.BindingDb], logger.Session("bindingdb-db"))
		if err != nil {
			logger.Error("failed to connect bindingdb database", err, lager.Data{"dbConfig": conf.DB[db.BindingDb]})
			os.Exit(1)
		}
		defer func() { _ = bindingDB.Close() }()
		prometheusCollectors = append(prometheusCollectors, healthendpoint.NewDatabaseStatusCollector("autoscaler", "golangapiserver", "bindingDB", bindingDB))
		checkBindingFunc = func(appId string) bool {
			return bindingDB.CheckServiceBinding(appId)
		}
		brokerHttpServer, err := brokerserver.NewBrokerServer(logger.Session("broker_http_server"), conf,
			bindingDB, policyDb, httpStatusCollector, cfClient, credentials)
		if err != nil {
			logger.Error("failed to create broker http server", err)
			os.Exit(1)
		}
		members = append(members, grouper.Member{"broker_http_server", brokerHttpServer})
	} else {
		checkBindingFunc = func(appId string) bool {
			return true
		}
	}

	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, prometheusCollectors, true, logger.Session("golangapiserver-prometheus"))

	rateLimiter := ratelimiter.DefaultRateLimiter(conf.RateLimit.MaxAmount, conf.RateLimit.ValidDuration, logger.Session("api-ratelimiter"))
	publicApiHttpServer, err := publicapiserver.NewPublicApiServer(logger.Session("public_api_http_server"), conf,
		policyDb, credentials, checkBindingFunc, cfClient, httpStatusCollector, rateLimiter, bindingDB)
	if err != nil {
		logger.Error("failed to create public api http server", err)
		os.Exit(1)
	}
	healthServer, err := healthendpoint.NewServerWithBasicAuth(conf.Health, []healthendpoint.Checker{}, logger.Session("health-server"), promRegistry, time.Now)
	if err != nil {
		logger.Error("failed to create health server", err)
		os.Exit(1)
	}

	members = append(members, grouper.Member{"public_api_http_server", publicApiHttpServer}, grouper.Member{"health_server", healthServer})

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))

	logger.Info("started")

	err = <-monitor.Wait()

	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}
