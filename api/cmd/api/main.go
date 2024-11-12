package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/brokerserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/publicapiserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	var path string
	var err error
	var conf *config.Config

	flag.StringVar(&path, "c", "", "config file")
	flag.Parse()

	vcapConfiguration, err := configutil.NewVCAPConfigurationReader()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to read vcap configuration : %s\n", err.Error())
		os.Exit(1)
	}

	conf, err = config.LoadConfig(path, vcapConfiguration)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to read config file '%s' : %s\n", path, err.Error())
		os.Exit(1)
	}

	err = conf.Validate()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to validate configuration : %s\n", err.Error())
		os.Exit(1)
	}

	helpers.SetupOpenTelemetry()

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "api")

	members := grouper.Members{}

	policyDb := sqldb.CreatePolicyDb(conf.Db[db.PolicyDb], logger)
	defer func() { _ = policyDb.Close() }()
	logger.Debug("Connected to PolicyDB", lager.Data{"dbConfig": conf.Db[db.PolicyDb]})

	credentialProvider := cred_helper.CredentialsProvider(conf.CredHelperImpl, conf.StoredProcedureConfig, conf.Db, 10*time.Second, 10*time.Minute, logger, policyDb)
	defer func() { _ = credentialProvider.Close() }()

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "golangapiserver")

	paClock := clock.NewClock()
	cfClient := cf.NewCFClient(&conf.CF, logger.Session("cf"), paClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"API": conf.CF.API})
		os.Exit(1)
	}
	logger.Debug("Successfully logged into CF", lager.Data{"API": conf.CF.API})

	bindingDb, err := sqldb.NewBindingSQLDB(conf.Db[db.BindingDb], logger.Session("bindingdb-db"))
	if err != nil {
		logger.Error("failed to connect bindingdb database", err, lager.Data{"dbConfig": conf.Db[db.BindingDb]})
		os.Exit(1)
	}
	defer func() { _ = bindingDb.Close() }()
	checkBindingFunc := func(appId string) bool {
		return bindingDb.CheckServiceBinding(appId)
	}
	brokerServer := brokerserver.NewBrokerServer(logger.Session("broker_http_server"), conf,
		bindingDb, policyDb, httpStatusCollector, cfClient, credentialProvider)

	rateLimiter := ratelimiter.DefaultRateLimiter(conf.RateLimit.MaxAmount, conf.RateLimit.ValidDuration, logger.Session("api-ratelimiter"))

	publicApiServer := publicapiserver.NewPublicApiServer(
		logger.Session("public_api_http_server"), conf, policyDb, bindingDb,
		credentialProvider, checkBindingFunc, cfClient, httpStatusCollector,
		rateLimiter, brokerServer)

	err = publicApiServer.Setup()
	if err != nil {
		logger.Error("failed to setup public api server", err)
		os.Exit(1)
	}

	mtlsServer, err := publicApiServer.GetMtlsServer()
	if err != nil {
		logger.Error("failed to create public api http server", err)
		os.Exit(1)
	}

	healthServer, err := publicApiServer.GetHealthServer()
	if err != nil {
		logger.Error("failed to create health http server", err)
		os.Exit(1)
	}

	brokerHttpServer, err := brokerServer.GetServer()
	if err != nil {
		logger.Error("failed to create broker http server", err)
		os.Exit(1)
	}

	unifiedServer, err := publicApiServer.GetUnifiedServer()
	if err != nil {
		logger.Error("failed to create public api http server", err)
		os.Exit(1)
	}

	members = append(members,
		grouper.Member{"public_api_http_server", mtlsServer},
		grouper.Member{"broker", brokerHttpServer},
		grouper.Member{"health_server", healthServer},
		grouper.Member{"unified_server", unifiedServer},
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
