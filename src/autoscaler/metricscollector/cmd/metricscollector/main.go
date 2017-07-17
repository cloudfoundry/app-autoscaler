package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/metricscollector"
	"autoscaler/metricscollector/collector"
	"autoscaler/metricscollector/config"
	"autoscaler/metricscollector/server"
	"autoscaler/models"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket"
	cfenv "github.com/cloudfoundry-community/go-cfenv"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/hashicorp/consul/api"
	"github.com/nu7hatch/gouuid"
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
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
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

	var createAppCollector func(string) collector.AppCollector
	if conf.Collector.CollectMethod == config.CollectMethodPolling {
		createAppCollector = func(appId string) collector.AppCollector {
			return collector.NewAppPoller(logger.Session("app-poller"), appId, conf.Collector.CollectInterval, cfClient, noaa, instanceMetricsDB, mcClock)
		}
	} else {
		createAppCollector = func(appId string) collector.AppCollector {
			noaaConsumer := consumer.New(dopplerUrl, tlsConfig, nil)
			noaaConsumer.RefreshTokenFrom(cfClient)
			return collector.NewAppStreamer(logger.Session("app-streamer"), appId, conf.Collector.CollectInterval, cfClient, noaaConsumer, instanceMetricsDB, mcClock)
		}
	}

	collectServer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		mc := collector.NewCollector(conf.Collector.RefreshInterval, logger.Session("collector"), policyDB, mcClock, createAppCollector)
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

	if conf.EnableDBLock {
		logger.Info("database-lock-feature-enabled")
		var lockDB db.LockDB
		lockDB, err = sqldb.NewLockSQLDB(conf.DBLock.LockDBURL, logger.Session("lock-db"))
		if err != nil {
			logger.Error("failed-to-connect-lock-database", err, lager.Data{"url": conf.DBLock.LockDBURL})
			os.Exit(1)
		}
		defer lockDB.Close()
		dbLockMaintainer := initDBLockRunner(conf, logger, lockDB)
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
				generateGUID(logger),
				conf.Lock.LockRetryInterval,
				conf.Lock.LockTTL,
			)
			members = append(grouper.Members{{"lock-maintainer", lockMaintainer}}, members...)
		}
		registrationRunner := initializeRegistrationRunner(logger, consulClient, conf.Server.Port, mcClock)

		members = append(members, grouper.Member{"registration", registrationRunner})
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

func initDBLockRunner(conf *config.Config, logger lager.Logger, lockDB db.LockDB) ifrit.Runner {
	dbLockMaintainer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		ttl := conf.DBLock.LockTTL
		lockTicker := time.NewTicker(ttl)
		readyFlag := true
		owner := getOwner(logger, conf)
		if owner == "" {
			logger.Info("failed-to-get-owner-details")
			os.Exit(1)
		}
		lock := &models.Lock{Owner: owner, Ttl: ttl}
		isLockAcquired, lockErr := lockDB.Lock(lock)
		if lockErr != nil {
			logger.Error("failed-to-acquire-lock-in-first-attempt", lockErr)
		}
		if isLockAcquired {
			logger.Info("lock-acquired-in-first-attempt", lager.Data{"owner": owner, "isLockAcquired": isLockAcquired})
			close(ready)
			readyFlag = false
		}
		for {
			select {
			case <-signals:
				logger.Info("received-interrupt-signal", lager.Data{"owner": owner})
				lockTicker.Stop()
				releaseErr := lockDB.Release(owner)
				if releaseErr != nil {
					logger.Error("failed-to-release-lock ", releaseErr)
				} else {
					logger.Info("successfully-released-lock", lager.Data{"owner": owner})
				}
				readyFlag = true
				return nil

			case <-lockTicker.C:
				logger.Info("retry-acquiring-lock", lager.Data{"owner": owner})
				lock := &models.Lock{Owner: owner, Ttl: ttl}
				isLockAcquired, lockErr := lockDB.Lock(lock)
				if lockErr != nil {
					logger.Error("failed-to-acquire-lock", lockErr)
					releaseErr := lockDB.Release(owner)
					if releaseErr != nil {
						logger.Error("failed-to-release-lock ", releaseErr)
					} else {
						logger.Info("successfully-released-lock", lager.Data{"owner": owner})
					}
					os.Exit(1)
				}
				if isLockAcquired && readyFlag {
					close(ready)
					readyFlag = false
					logger.Info("successfully-acquired-lock", lager.Data{"owner": owner})
				}
			}
		}
	})
	return dbLockMaintainer
}

func initLoggerFromConfig(conf *config.LoggingConfig) lager.Logger {
	logLevel, err := getLogLevel(conf.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %s\n", err.Error())
		os.Exit(1)
	}
	logger := lager.NewLogger("metricscollector")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, logLevel))

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

func initializeRegistrationRunner(
	logger lager.Logger,
	consulClient consuladapter.Client,
	port int,
	clock clock.Clock) ifrit.Runner {
	registration := &api.AgentServiceRegistration{
		Name: "metricscollector",
		Port: port,
		Check: &api.AgentServiceCheck{
			TTL: "20s",
		},
	}
	return locket.NewRegistrationRunner(logger, registration, consulClient, locket.RetryInterval, clock)
}

func generateGUID(logger lager.Logger) string {
	uuid, err := uuid.NewV4()
	if err != nil {
		logger.Fatal("Couldn't generate uuid", err)
	}
	return uuid.String()
}

func getOwner(logger lager.Logger, conf *config.Config) string {
	var owner string
	if strings.TrimSpace(os.Getenv("VCAP_APPLICATION")) != "" {
		appEnv, _ := cfenv.Current()
		owner = appEnv.ID
		logger.Info("ownership found in VCAP_APPLICATION", lager.Data{"owner": owner})
	} else if conf.DBLock.Owner != "" {
		owner = conf.DBLock.Owner
		logger.Info("ownership found in config file", lager.Data{"owner": owner})
	}
	return owner
}
