package sqldb

import (
	"database/sql"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"

	"autoscaler/db"
	"autoscaler/healthendpoint"
	"autoscaler/models"

	"time"
)

type ScalingEngineSQLDB struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sql.DB
}

func NewScalingEngineSQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*ScalingEngineSQLDB, error) {
	sqldb, err := sql.Open(db.PostgresDriverName, dbConfig.URL)
	if err != nil {
		logger.Error("open-scaling-engine-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("ping-scaling-engine-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(dbConfig.MaxIdleConnections)
	sqldb.SetMaxOpenConns(dbConfig.MaxOpenConnections)
	return &ScalingEngineSQLDB{
		dbConfig: dbConfig,
		logger:   logger,
		sqldb:    sqldb,
	}, nil
}

func (sdb *ScalingEngineSQLDB) Close() error {
	err := sdb.sqldb.Close()
	if err != nil {
		sdb.logger.Error("close-scaling-engine-db", err, lager.Data{"dbConfig": sdb.dbConfig})
		return err
	}
	return nil
}

func (sdb *ScalingEngineSQLDB) SaveScalingHistory(history *models.AppScalingHistory) error {
	query := "INSERT INTO scalinghistory" +
		"(appid, timestamp, scalingtype, status, oldinstances, newinstances, reason, message, error) " +
		" VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	_, err := sdb.sqldb.Exec(query, history.AppId, history.Timestamp, history.ScalingType, history.Status,
		history.OldInstances, history.NewInstances, history.Reason, history.Message, history.Error)

	if err != nil {
		sdb.logger.Error("save-scaling-history", err, lager.Data{"query": query, "history": history})
	}
	return err
}

func (sdb *ScalingEngineSQLDB) RetrieveScalingHistories(appId string, start int64, end int64, orderType db.OrderType, includeAll bool) ([]*models.AppScalingHistory, error) {
	var orderStr string
	if orderType == db.DESC {
		orderStr = db.DESCSTR
	} else {
		orderStr = db.ASCSTR
	}

	query := "SELECT timestamp, scalingtype, status, oldinstances, newinstances, reason, message, error FROM scalinghistory WHERE" +
		" appid = $1 " +
		" AND timestamp >= $2" +
		" AND timestamp <= $3" +
		" ORDER BY timestamp " + orderStr

	if end < 0 {
		end = time.Now().UnixNano()
	}

	histories := []*models.AppScalingHistory{}
	rows, err := sdb.sqldb.Query(query, appId, start, end)
	if err != nil {
		sdb.logger.Error("retrieve-scaling-histories", err,
			lager.Data{"query": query, "appid": appId, "start": start, "end": end, "orderType": orderType})
		return nil, err
	}

	defer rows.Close()

	var timestamp int64
	var scalingType, status, oldInstances, newInstances int
	var reason, message, errorMsg string

	for rows.Next() {
		if err = rows.Scan(&timestamp, &scalingType, &status, &oldInstances, &newInstances, &reason, &message, &errorMsg); err != nil {
			sdb.logger.Error("retrieve-scaling-history-scan", err)
			return nil, err
		}

		history := models.AppScalingHistory{
			AppId:        appId,
			Timestamp:    timestamp,
			ScalingType:  models.ScalingType(scalingType),
			Status:       models.ScalingStatus(status),
			OldInstances: oldInstances,
			NewInstances: newInstances,
			Reason:       reason,
			Message:      message,
			Error:        errorMsg,
		}

		if includeAll || history.Status != models.ScalingStatusIgnored {
			histories = append(histories, &history)
		}
	}
	return histories, nil
}

func (sdb *ScalingEngineSQLDB) PruneScalingHistories(before int64) error {
	query := "DELETE FROM scalinghistory WHERE timestamp <= $1"
	_, err := sdb.sqldb.Exec(query, before)
	if err != nil {
		sdb.logger.Error("failed-prune-scaling-histories-from-scalinghistory-table", err, lager.Data{"query": query, "before": before})
	}
	return err
}

func (sdb *ScalingEngineSQLDB) CanScaleApp(appId string) (bool, int64, error) {
	query := "SELECT expireat FROM scalingcooldown WHERE appid = $1"
	rows, err := sdb.sqldb.Query(query, appId)
	if err != nil {
		sdb.logger.Error("can-scale-app-query-record", err, lager.Data{"query": query, "appid": appId})
		return false, 0, err
	}
	defer rows.Close()

	var expireAt int64 = 0
	if rows.Next() {
		if err = rows.Scan(&expireAt); err != nil {
			sdb.logger.Error("can-scale-app-scan", err, lager.Data{"query": query, "appid": appId})
			return false, expireAt, err
		}
		if expireAt < time.Now().UnixNano() {
			return true, expireAt, nil
		} else {
			return false, expireAt, nil
		}
	}
	return true, expireAt, nil
}

func (sdb *ScalingEngineSQLDB) UpdateScalingCooldownExpireTime(appId string, expireAt int64) error {
	_, err := sdb.sqldb.Exec("DELETE FROM scalingcooldown WHERE appid = $1", appId)
	if err != nil {
		sdb.logger.Error("update-scaling-cooldown-time-delete", err, lager.Data{"appid": appId})
		return err
	}

	_, err = sdb.sqldb.Exec("INSERT INTO scalingcooldown(appid, expireat) values($1, $2)", appId, expireAt)
	if err != nil {
		sdb.logger.Error("update-scaling-cooldown-time-insert", err, lager.Data{"appid": appId, "expireAt": expireAt})
		return err
	}
	return nil
}

func (sdb *ScalingEngineSQLDB) GetActiveSchedule(appId string) (*models.ActiveSchedule, error) {
	query := "SELECT scheduleid, instancemincount, instancemaxcount, initialmininstancecount" +
		" FROM activeschedule WHERE appid = $1"

	var scheduleId string
	var instanceMin, instanceMax, instanceMinInitial int

	err := sdb.sqldb.QueryRow(query, appId).Scan(&scheduleId, &instanceMin, &instanceMax, &instanceMinInitial)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		sdb.logger.Error("failed-get-active-schedule-query-row-scan", err, lager.Data{"query": query, "appid": appId})
		return nil, err
	}

	return &models.ActiveSchedule{
		ScheduleId:         scheduleId,
		InstanceMin:        instanceMin,
		InstanceMax:        instanceMax,
		InstanceMinInitial: instanceMinInitial,
	}, nil
}

func (sdb *ScalingEngineSQLDB) GetActiveSchedules() (map[string]string, error) {
	query := "SELECT scheduleid, appid FROM activeschedule"
	rows, err := sdb.sqldb.Query(query)
	if err != nil {
		sdb.logger.Error("failed-get-active-schedules", err, lager.Data{"query": query})
		return nil, err
	}
	defer rows.Close()

	schedules := make(map[string]string)
	var id, appId string
	for rows.Next() {
		if err = rows.Scan(&id, &appId); err != nil {
			sdb.logger.Error("failed-get-active-schedules-scan", err, lager.Data{"query": query})
			return nil, err
		}
		schedules[appId] = id
	}
	return schedules, nil
}

func (sdb *ScalingEngineSQLDB) RemoveActiveSchedule(appId string) error {
	query := "DELETE FROM activeschedule WHERE appid = $1"
	_, err := sdb.sqldb.Exec(query, appId)
	if err != nil {
		sdb.logger.Error("failed-remove-active-scheudle", err, lager.Data{"appid": appId})
	}
	return err
}

func (sdb *ScalingEngineSQLDB) SetActiveSchedule(appId string, schedule *models.ActiveSchedule) error {
	err := sdb.RemoveActiveSchedule(appId)
	if err != nil {
		sdb.logger.Error("failed-set-active-scheudle-remove", err, lager.Data{"appid": appId})
		return err
	}

	query := "INSERT INTO activeschedule(appid, scheduleid, instancemincount, instancemaxcount, initialmininstancecount) " +
		" VALUES ($1, $2, $3, $4, $5)"
	_, err = sdb.sqldb.Exec(query, appId, schedule.ScheduleId, schedule.InstanceMin, schedule.InstanceMax, schedule.InstanceMinInitial)

	if err != nil {
		sdb.logger.Error("failed-set-active-scheudle-insert", err, lager.Data{"appid": appId, "schedule": schedule})
	}
	return err
}

func (sdb *ScalingEngineSQLDB) EmitHealthMetrics(h healthendpoint.Health, cclock clock.Clock, interval time.Duration) {
	go func() {
		ticker := cclock.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C() {
			h.Set("openConnection_scalingEngineDB", float64(sdb.sqldb.Stats().OpenConnections))
		}
	}()
}
