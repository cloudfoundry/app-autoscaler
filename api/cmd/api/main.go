package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

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
	"code.cloudfoundry.org/lager/v3"
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

	policyDb := sqldb.CreatePolicyDb(conf.DB[db.PolicyDb], logger)
	defer func() { _ = policyDb.Close() }()
	logger.Debug("Connected to PolicyDB", lager.Data{"dbConfig": conf.DB[db.PolicyDb]})

	credentialProvider := cred_helper.CredentialsProvider(conf.CredHelperImpl, conf.StoredProcedureConfig, conf.DB, 10*time.Second, 10*time.Minute, logger, policyDb)
	defer func() { _ = credentialProvider.Close() }()

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "golangapiserver")
	prometheusCollectors := []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "golangapiserver", "policyDB", policyDb),
		httpStatusCollector,
	}

	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.TraceContext{})

	paClock := clock.NewClock()
	cfClient := cf.NewCFClient(&conf.CF, logger.Session("cf"), paClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"API": conf.CF.API})
		os.Exit(1)
	}
	logger.Debug("Successfully logged into CF", lager.Data{"API": conf.CF.API})

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
			bindingDB, policyDb, httpStatusCollector, cfClient, credentialProvider)
		if err != nil {
			logger.Error("failed to create broker http server", err)
			os.Exit(1)
		}
		members = append(members, grouper.Member{"broker_http_server", brokerHttpServer})
	} else {
		checkBindingFunc = func(appId string) bool { return true }
	}

	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, prometheusCollectors, true, logger.Session("golangapiserver-prometheus"))

	publicApiHttpServer := createApiServer(conf, logger, policyDb, credentialProvider, checkBindingFunc, cfClient, httpStatusCollector, bindingDB)
	healthServer, err := healthendpoint.NewServerWithBasicAuth(conf.Health, []healthendpoint.Checker{}, logger.Session("health-server"), promRegistry, time.Now)
	if err != nil {
		logger.Fatal("Failed to create health server", err)
		os.Exit(1)
	}
	logger.Debug("Successfully created health server")

	members = append(members,
		grouper.Member{"public_api_http_server", publicApiHttpServer},
		grouper.Member{"health_server", healthServer},
	)

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))

	logger.Info("started")

	err = <-monitor.Wait()

	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}

func createApiServer(conf *config.Config, logger lager.Logger, policyDb *sqldb.PolicySQLDB, credentialProvider cred_helper.Credentials, checkBindingFunc api.CheckBindingFunc, cfClient cf.CFClient, httpStatusCollector healthendpoint.HTTPStatusCollector, bindingDB db.BindingDB) ifrit.Runner {
	rateLimiter := ratelimiter.DefaultRateLimiter(conf.RateLimit.MaxAmount, conf.RateLimit.ValidDuration, logger.Session("api-ratelimiter"))
	publicApiHttpServer, err := publicapiserver.NewPublicApiServer(logger.Session("public_api_http_server"), conf, policyDb, credentialProvider, checkBindingFunc, cfClient, httpStatusCollector, rateLimiter, bindingDB)
	if err != nil {
		logger.Error("failed to create public api http server", err)
		os.Exit(1)
	}
	return publicApiHttpServer
}
