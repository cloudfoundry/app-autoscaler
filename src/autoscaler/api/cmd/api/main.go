package main

import (
	"flag"
	"fmt"
	"os"

	"autoscaler/api/brokerserver"
	"autoscaler/api/config"
	"autoscaler/api/publicapiserver"
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/helpers"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
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

	logger := helpers.InitLoggerFromConfig(&conf.Logging, "api")

	members := grouper.Members{}

	var policyDb db.PolicyDB
	policyDb, err = sqldb.NewPolicySQLDB(conf.DB.PolicyDB, logger.Session("policydb-db"))
	if err != nil {
		logger.Error("failed to connect to policydb database", err, lager.Data{"dbConfig": conf.DB.PolicyDB})
		os.Exit(1)
	}
	defer policyDb.Close()

	var bindingDB db.BindingDB
	bindingDB, err = sqldb.NewBindingSQLDB(conf.DB.BindingDB, logger.Session("bindingdb-db"))
	if err != nil {
		logger.Error("failed to connect bindingdb database", err, lager.Data{"dbConfig": conf.DB.BindingDB})
		os.Exit(1)
	}
	defer bindingDB.Close()

	if !conf.UseBuildInMode {
		brokerHttpServer, err := brokerserver.NewBrokerServer(logger.Session("broker_http_server"), conf, bindingDB, policyDb)
		if err != nil {
			logger.Error("failed to create broker http server", err)
			os.Exit(1)
		}
		members = append(members, grouper.Member{"broker_http_server", brokerHttpServer})
	}

	paClock := clock.NewClock()
	cfClient := cf.NewCFClient(&conf.CF, logger.Session("cf"), paClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"API": conf.CF.API})
		os.Exit(1)
	}

	publicApiHttpServer, err := publicapiserver.NewPublicApiServer(logger.Session("public_api_http_server"), conf, policyDb, bindingDB, cfClient)
	if err != nil {
		logger.Error("failed to create public api http server", err)
		os.Exit(1)
	}

	members = append(members, grouper.Member{"public_api_http_server", publicApiHttpServer})

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))

	logger.Info("started")

	err = <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}
	logger.Info("exited")
}
