package main

import (
	"autoscaler/helpers"
	"autoscaler/sync"
	"flag"
	"fmt"
	"os"
	"time"

	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/db/sqldb"
	"autoscaler/scalingengine"
	"autoscaler/scalingengine/config"
	"autoscaler/scalingengine/schedule"
	"autoscaler/scalingengine/server"

	"code.cloudfoundry.org/cfhttp"
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

	cfhttp.Initialize(5 * time.Second)

	logger := initLoggerFromConfig(&conf.Logging)
	eClock := clock.NewClock()

	cfClient := cf.NewCfClient(&conf.Cf, logger.Session("cf"), eClock)
	err = cfClient.Login()
	if err != nil {
		logger.Error("failed to login cloud foundry", err, lager.Data{"Api": conf.Cf.Api})
		os.Exit(1)
	}

	var policyDB db.PolicyDB
	policyDB, err = sqldb.NewPolicySQLDB(conf.Db.PolicyDb, logger.Session("policy-db"))
	if err != nil {
		logger.Error("failed to connect policy database", err, lager.Data{"dbConfig": conf.Db.PolicyDb})
		os.Exit(1)
	}
	defer policyDB.Close()

	var scalingEngineDB db.ScalingEngineDB
	scalingEngineDB, err = sqldb.NewScalingEngineSQLDB(conf.Db.ScalingEngineDb, logger.Session("scalingengine-db"))
	if err != nil {
		logger.Error("failed to connect scalingengine database", err, lager.Data{"dbConfig": conf.Db.ScalingEngineDb})
		os.Exit(1)
	}
	defer scalingEngineDB.Close()

	var schedulerDB db.SchedulerDB
	schedulerDB, err = sqldb.NewSchedulerSQLDB(conf.Db.SchedulerDb, logger.Session("scheduler-db"))
	if err != nil {
		logger.Error("failed to connect scheduler database", err, lager.Data{"dbConfig": conf.Db.SchedulerDb})
		os.Exit(1)
	}
	defer schedulerDB.Close()

	scalingEngine := scalingengine.NewScalingEngine(logger, cfClient, policyDB, scalingEngineDB, eClock, conf.DefaultCoolDownSecs, conf.LockSize)
	httpServer, err := server.NewServer(logger.Session("http-server"), conf, scalingEngineDB, scalingEngine)
	if err != nil {
		logger.Error("failed to create http server", err)
		os.Exit(1)
	}

	synchronizer := schedule.NewActiveScheduleSychronizer(logger.Session("synchronizer"), schedulerDB, scalingEngineDB, scalingEngine,
		conf.Synchronizer.ActiveScheduleSyncInterval, eClock)

	nonLockMembers := grouper.Members{
		{"http_server", httpServer},
	}

	lockMembers := grouper.Members{
		{"schedule_synchronizer", synchronizer},
	}

	if conf.EnableDBLock {
		const lockTableName = "scalingengine_lock"
		guid, err := helpers.GenerateGUID(logger)
		if err != nil {
			logger.Error("failed-to-generate-guid", err)
		}
		logger.Debug("database-lock-feature-enabled")
		var lockDB db.LockDB
		lockDB, err = sqldb.NewLockSQLDB(conf.DBLock.LockDB, lockTableName, logger.Session("lock-db"))
		if err != nil {
			logger.Error("failed-to-connect-lock-database", err, lager.Data{"dbConfig": conf.DBLock.LockDB})
			os.Exit(1)
		}
		defer func() {
			lockDB.Release(guid)
			lockDB.Close()
		}()

		sedl := sync.NewDatabaseLock(logger)
		dbLockMaintainer := sedl.InitDBLockRunner(conf.DBLock.LockRetryInterval, conf.DBLock.LockTTL, guid, lockDB)
		lockMembers = append(grouper.Members{{"db-lock-maintainer", dbLockMaintainer}}, lockMembers...)
	}

	nonLockMonitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, nonLockMembers)))

	var done = make(chan struct{})

	go func() {
		defer close(done)
		lockMonitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, lockMembers)))
		lmerr := <-lockMonitor.Wait()
		if lmerr != nil {
			logger.Error("sync-exited-with-failure", lmerr)
			os.Exit(1)
		}
	}()

	logger.Info("started")
	nlmerr := <-nonLockMonitor.Wait()
	if nlmerr != nil {
		logger.Error("http-server-exited-with-failure", nlmerr)
		os.Exit(1)
	}

	<-done
	logger.Info("exited")
}

func initLoggerFromConfig(conf *config.LoggingConfig) lager.Logger {
	logLevel, err := getLogLevel(conf.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %s\n", err.Error())
		os.Exit(1)
	}
	logger := lager.NewLogger("scalingengine")

	keyPatterns := []string{"[Pp]wd", "[Pp]ass", "[Ss]ecret", "[Tt]oken"}

	redactedSink, err := helpers.NewRedactingWriterWithURLCredSink(os.Stdout, logLevel, keyPatterns, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create redacted sink: %s", err.Error())
	}
	logger.RegisterSink(redactedSink)

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
