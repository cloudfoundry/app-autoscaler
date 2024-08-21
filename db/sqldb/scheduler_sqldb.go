package sqldb

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"database/sql"
	"strconv"
)

type SchedulerSQLDB struct {
	sqldb    *sql.DB
	logger   lager.Logger
	dbConfig db.DatabaseConfig
}

func NewSchedulerSQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*SchedulerSQLDB, error) {
	database, err := db.GetConnection(dbConfig.URL)
	if err != nil {
		return nil, err
	}
	sqldb, err := sql.Open(database.DriverName, database.DataSourceName)
	if err != nil {
		logger.Error("failed-open-scheduler-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("failed-ping-scheduler-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(int(dbConfig.MaxIdleConnections))
	sqldb.SetMaxOpenConns(int(dbConfig.MaxOpenConnections))
	sqldb.SetConnMaxIdleTime(dbConfig.ConnectionMaxIdleTime)

	return &SchedulerSQLDB{
		dbConfig: dbConfig,
		logger:   logger,
		sqldb:    sqldb,
	}, nil
}

func (sdb *SchedulerSQLDB) Close() error {
	err := sdb.sqldb.Close()
	if err != nil {
		sdb.logger.Error("failed-close-scheduler-db", err, lager.Data{"dbConfig": sdb.dbConfig})
		return err
	}
	return nil
}

func (sdb *SchedulerSQLDB) GetActiveSchedules() (map[string]*models.ActiveSchedule, error) {
	query := "SELECT id, app_id, instance_min_count, instance_max_count, initial_min_instance_count FROM app_scaling_active_schedule"
	rows, err := sdb.sqldb.Query(query)
	if err != nil {
		sdb.logger.Error("failed-get-active-schedules-query", err, lager.Data{"query": query})
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	schedules := make(map[string]*models.ActiveSchedule)
	var id int64
	var appId string
	var instanceMin, instanceMax int
	minInitial := sql.NullInt64{}
	for rows.Next() {
		if err = rows.Scan(&id, &appId, &instanceMin, &instanceMax, &minInitial); err != nil {
			sdb.logger.Error("failed-get-active-schedules-scan", err)
			return nil, err
		}
		instanceMinInitial := 0
		if minInitial.Valid {
			instanceMinInitial = int(minInitial.Int64)
		}

		schedule := models.ActiveSchedule{
			ScheduleId:         strconv.FormatInt(id, 10),
			InstanceMin:        instanceMin,
			InstanceMax:        instanceMax,
			InstanceMinInitial: instanceMinInitial,
		}
		schedules[appId] = &schedule
	}
	return schedules, rows.Err()
}

func (sdb *SchedulerSQLDB) GetDBStatus() sql.DBStats {
	return sdb.sqldb.Stats()
}
