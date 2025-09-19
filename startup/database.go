package startup

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/lager/v3"
)

// DatabaseConnection manages a database connection with cleanup
type DatabaseConnection[T any] struct {
	DB     T
	Closer func() error
}

// CreatePolicyDB creates and connects to policy database
func CreatePolicyDB(dbConfig db.DatabaseConfig, logger lager.Logger) *DatabaseConnection[db.PolicyDB] {
	policyDb := sqldb.CreatePolicyDb(dbConfig, logger)
	return &DatabaseConnection[db.PolicyDB]{
		DB:     policyDb,
		Closer: policyDb.Close,
	}
}

// CreateAppMetricDB creates and connects to app metric database
func CreateAppMetricDB(dbConfig db.DatabaseConfig, logger lager.Logger) *DatabaseConnection[db.AppMetricDB] {
	appMetricDB, err := sqldb.NewAppMetricSQLDB(dbConfig, logger.Session("appmetric-db"))
	ExitOnError(err, logger, "failed to connect appmetrics db", lager.Data{"dbConfig": dbConfig})
	return &DatabaseConnection[db.AppMetricDB]{
		DB:     appMetricDB,
		Closer: appMetricDB.Close,
	}
}

// CreateScalingEngineDB creates and connects to scaling engine database
func CreateScalingEngineDB(dbConfig db.DatabaseConfig, logger lager.Logger) *DatabaseConnection[db.ScalingEngineDB] {
	scalingEngineDB, err := sqldb.NewScalingEngineSQLDB(dbConfig, logger.Session("scalingengine-db"))
	ExitOnError(err, logger, "failed to connect scalingengine db", lager.Data{"dbConfig": dbConfig})
	return &DatabaseConnection[db.ScalingEngineDB]{
		DB:     scalingEngineDB,
		Closer: scalingEngineDB.Close,
	}
}

// CreateSchedulerDB creates and connects to scheduler database
func CreateSchedulerDB(dbConfig db.DatabaseConfig, logger lager.Logger) *DatabaseConnection[db.SchedulerDB] {
	schedulerDB, err := sqldb.NewSchedulerSQLDB(dbConfig, logger.Session("scheduler-db"))
	ExitOnError(err, logger, "failed to connect scheduler database", lager.Data{"dbConfig": dbConfig})
	return &DatabaseConnection[db.SchedulerDB]{
		DB:     schedulerDB,
		Closer: schedulerDB.Close,
	}
}

// CreateBindingDB creates and connects to binding database
func CreateBindingDB(dbConfig db.DatabaseConfig, logger lager.Logger) *DatabaseConnection[db.BindingDB] {
	bindingDB := sqldb.CreateBindingDB(dbConfig, logger)
	return &DatabaseConnection[db.BindingDB]{
		DB:     bindingDB,
		Closer: bindingDB.Close,
	}
}

// CreateLockDB creates and connects to lock database
func CreateLockDB(dbConfig db.DatabaseConfig, lockTableName string, logger lager.Logger) *DatabaseConnection[db.LockDB] {
	lockDB, err := sqldb.NewLockSQLDB(dbConfig, lockTableName, logger.Session("lock-db"))
	ExitOnError(err, logger, "failed-to-connect-lock-database", lager.Data{"dbConfig": dbConfig})
	return &DatabaseConnection[db.LockDB]{
		DB:     lockDB,
		Closer: lockDB.Close,
	}
}

// CleanupDatabases handles cleanup for multiple database connections
func CleanupDatabases(connections ...interface{ Closer() error }) {
	for _, conn := range connections {
		if conn != nil {
			_ = conn.Closer()
		}
	}
}
