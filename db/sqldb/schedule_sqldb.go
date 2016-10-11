package sqldb

import (
	"code.cloudfoundry.org/lager"
	"database/sql"
	_ "github.com/lib/pq"

	"autoscaler/db"
	"autoscaler/models"

	"strconv"
)

type ScheduleSQLDB struct {
	url    string
	logger lager.Logger
	sqldb  *sql.DB
}

func NewScheduleSQLDB(url string, logger lager.Logger) (*ScheduleSQLDB, error) {
	sqldb, err := sql.Open(db.PostgresDriverName, url)
	if err != nil {
		logger.Error("failed-open-schedule-db", err, lager.Data{"url": url})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("failed-ping-schedule-db", err, lager.Data{"url": url})
		return nil, err
	}

	return &ScheduleSQLDB{
		url:    url,
		logger: logger,
		sqldb:  sqldb,
	}, nil
}

func (sdb *ScheduleSQLDB) Close() error {
	err := sdb.sqldb.Close()
	if err != nil {
		sdb.logger.Error("failed-close-schedule-db", err, lager.Data{"url": sdb.url})
		return err
	}
	return nil
}

func (sdb *ScheduleSQLDB) GetActiveSchedule(appId string) (*models.ActiveSchedule, error) {
	query := "SELECT active_schedule_id, instance_min_count, instance_max_count, initial_min_instance_count" +
		" FROM app_scaling_active_schedule WHERE app_id = $1 ORDER BY active_schedule_id DESC"
	logger := sdb.logger.WithData(lager.Data{"query": query, "appid": appId})

	rows, err := sdb.sqldb.Query(query, appId)
	if err != nil {
		logger.Error("failed-get-active-schedule-query", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var activeScheduleId int64
		var instanceMin, instanceMax int
		minInitial := sql.NullInt64{}
		if err = rows.Scan(&activeScheduleId, &instanceMin, &instanceMax, &minInitial); err != nil {
			logger.Error("failed-get-active-schedule-scan", err)
			return nil, err
		}

		instanceMinInitial := 0
		if minInitial.Valid {
			instanceMinInitial = int(minInitial.Int64)
		}

		return &models.ActiveSchedule{
			ScheduleId:         strconv.FormatInt(activeScheduleId, 10),
			InstanceMin:        instanceMin,
			InstanceMax:        instanceMax,
			InstanceMinInitial: instanceMinInitial,
		}, nil
	}
	return nil, nil
}
