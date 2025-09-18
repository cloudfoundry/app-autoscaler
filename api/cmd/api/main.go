package main

import (
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/brokerserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/publicapiserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
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

	logger := startup.InitLogger(&conf.Logging, "api")

	policyDb := sqldb.CreatePolicyDb(conf.Db[db.PolicyDb], logger)
	defer func() { _ = policyDb.Close() }()
	logger.Debug("Connected to PolicyDB", lager.Data{"dbConfig": conf.Db[db.PolicyDb]})

	credentialProvider := cred_helper.CredentialsProvider(conf.CredHelperImpl, conf.StoredProcedureConfig, conf.Db, 10*time.Second, 10*time.Minute, logger, policyDb)
	defer func() { _ = credentialProvider.Close() }()

	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "golangapiserver")

	paClock := clock.NewClock()
	cfClient := cf.NewCFClient(&conf.CF, logger.Session("cf"), paClock)
	err = cfClient.Login()
	startup.ExitOnError(err, logger, "failed to login cloud foundry", lager.Data{"API": conf.CF.API})
	logger.Debug("Successfully logged into CF", lager.Data{"API": conf.CF.API})

	bindingDb, err := sqldb.NewBindingSQLDB(conf.Db[db.BindingDb], logger.Session("bindingdb-db"))
	startup.ExitOnError(err, logger, "failed to connect bindingdb database", lager.Data{"dbConfig": conf.Db[db.BindingDb]})
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

	mtlsServer, err := publicApiServer.CreateMtlsServer()
	startup.ExitOnError(err, logger, "failed to create public api http server")

	healthServer, err := publicApiServer.CreateHealthServer()
	startup.ExitOnError(err, logger, "failed to create health http server")

	brokerHttpServer, err := brokerServer.CreateServer()
	startup.ExitOnError(err, logger, "failed to create broker http server")

	unifiedServer, err := publicApiServer.CreateCFServer()
	startup.ExitOnError(err, logger, "failed to create public api http server")

	members := grouper.Members{}
	members = append(members,
		grouper.Member{Name: "public_api_http_server", Runner: mtlsServer},
		grouper.Member{Name: "broker", Runner: brokerHttpServer},
		grouper.Member{Name: "health_server", Runner: healthServer},
		grouper.Member{Name: "unified_server", Runner: unifiedServer},
	)

	err = startup.StartServices(logger, members)
	if err != nil {
		os.Exit(1)
	}
}
