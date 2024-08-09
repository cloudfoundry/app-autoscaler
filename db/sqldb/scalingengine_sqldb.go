package sqldb

import (
	"context"
	"fmt"
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"github.com/uptrace/opentelemetry-go-extra/otelsqlx"

	"code.cloudfoundry.org/lager/v3"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"database/sql"
	"time"
)

type ScalingEngineSQLDB struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sqlx.DB
}

func NewScalingEngineSQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*ScalingEngineSQLDB, error) {
	database, err := db.GetConnection(dbConfig.URL)
	if err != nil {
		return nil, err
	}

	sqldb, err := otelsqlx.Open(database.DriverName, database.DataSourceName, otelsql.WithAttributes(database.OTELAttribute))
	if err != nil {
		logger.Error("open-scaling-engine-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		_ = sqldb.Close()
		logger.Error("ping-scaling-engine-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(int(dbConfig.MaxIdleConnections))
	sqldb.SetMaxOpenConns(int(dbConfig.MaxOpenConnections))
	sqldb.SetConnMaxIdleTime(dbConfig.ConnectionMaxIdleTime)

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
	query := sdb.sqldb.Rebind("INSERT INTO scalinghistory" +
		"(appid, timestamp, scalingtype, status, oldinstances, newinstances, reason, message, error) " +
		" VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)")
	_, err := sdb.sqldb.Exec(query, history.AppId, history.Timestamp, history.ScalingType, history.Status,
		history.OldInstances, history.NewInstances, history.Reason, history.Message, history.Error)

	if err != nil {
		return fmt.Errorf("saveScalingHistory failed appId(%s) scalingtype(%d) reason(%s): %w", history.AppId, history.ScalingType, history.Reason, err)
	}
	return nil
}

func (sdb *ScalingEngineSQLDB) CountScalingHistories(ctx context.Context, appId string, start int64, end int64, includeAll bool) (int, error) {
	query := sdb.sqldb.Rebind("SELECT COUNT(*) FROM scalinghistory WHERE appid = ? AND timestamp >= ? AND timestamp <= ?" + statusFilter(includeAll))

	if end < 0 {
		end = time.Now().UnixNano()
	}

	var count int
	err := sdb.sqldb.GetContext(ctx, &count, query, appId, start, end)
	if err != nil {
		sdb.logger.Error("count-scaling-histories", err,
			lager.Data{"query": query, "appid": appId, "start": start, "end": end})
		return 0, err
	}

	return count, nil
}

func (sdb *ScalingEngineSQLDB) RetrieveScalingHistories(ctx context.Context, appId string, start int64, end int64, orderType db.OrderType, includeAll bool, page int, resultsPerPage int) ([]*models.AppScalingHistory, error) {
	query := sdb.sqldb.Rebind("SELECT timestamp, scalingtype, status, oldinstances, newinstances, reason, message, error FROM scalinghistory WHERE" +
		" appid = ? " +
		" AND timestamp >= ?" +
		" AND timestamp <= ?" +
		statusFilter(includeAll) +
		" ORDER BY timestamp " + orderTypeToString(orderType) +
		" LIMIT ? OFFSET ?")

	if end < 0 {
		end = time.Now().UnixNano()
	}

	histories := []*models.AppScalingHistory{}
	rows, err := sdb.sqldb.QueryContext(ctx, query, appId, start, end, resultsPerPage, (page-1)*resultsPerPage)
	if err != nil {
		sdb.logger.Error("retrieve-scaling-histories", err,
			lager.Data{"query": query, "appid": appId, "start": start, "end": end, "orderType": orderType})
		return nil, err
	}

	defer func() { _ = rows.Close() }()

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
		histories = append(histories, &history)
	}
	return histories, rows.Err()
}

func statusFilter(includeAll bool) string {
	statusFilter := " AND status != " + strconv.Itoa(int(models.ScalingStatusIgnored))
	if includeAll {
		statusFilter = ""
	}
	return statusFilter
}

func orderTypeToString(orderType db.OrderType) string {
	orderStr := db.ASCSTR
	if orderType == db.DESC {
		orderStr = db.DESCSTR
	}
	return orderStr
}

func (sdb *ScalingEngineSQLDB) PruneScalingHistories(ctx context.Context, before int64) error {
	query := sdb.sqldb.Rebind("DELETE FROM scalinghistory WHERE timestamp <= ?")
	_, err := sdb.sqldb.ExecContext(ctx, query, before)
	if err != nil {
		sdb.logger.Error("failed-prune-scaling-histories-from-scalinghistory-table", err, lager.Data{"query": query, "before": before})
	}
	return err
}

func (sdb *ScalingEngineSQLDB) PruneCooldowns(ctx context.Context, before int64) error {
	query := sdb.sqldb.Rebind("DELETE FROM scalingcooldown WHERE expireat < ?")
	_, err := sdb.sqldb.ExecContext(ctx, query, before)
	if err != nil {
		sdb.logger.Error("failed-prune-scaling-cooldowns-from-scalingcooldown-table", err, lager.Data{"query": query, "before": before})
	}
	return err
}

func (sdb *ScalingEngineSQLDB) CanScaleApp(appId string) (bool, int64, error) {
	query := sdb.sqldb.Rebind("SELECT expireat FROM scalingcooldown WHERE appid = ?")
	rows, err := sdb.sqldb.Query(query, appId)
	if err != nil {
		sdb.logger.Error("can-scale-app-query-record", err, lager.Data{"query": query, "appid": appId})
		return false, 0, err
	}
	defer func() { _ = rows.Close() }()

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
	return true, expireAt, rows.Err()
}

func (sdb *ScalingEngineSQLDB) UpdateScalingCooldownExpireTime(appId string, expireAt int64) error {
	_, err := sdb.sqldb.Exec(sdb.sqldb.Rebind("DELETE FROM scalingcooldown WHERE appid = ?"), appId)
	if err != nil {
		sdb.logger.Error("update-scaling-cooldown-time-delete", err, lager.Data{"appid": appId})
		return err
	}

	_, err = sdb.sqldb.Exec(sdb.sqldb.Rebind("INSERT INTO scalingcooldown(appid, expireat) values(?, ?)"), appId, expireAt)
	if err != nil {
		sdb.logger.Error("update-scaling-cooldown-time-insert", err, lager.Data{"appid": appId, "expireAt": expireAt})
		return err
	}
	return nil
}

func (sdb *ScalingEngineSQLDB) GetActiveSchedule(appId string) (*models.ActiveSchedule, error) {
	query := sdb.sqldb.Rebind("SELECT scheduleid, instancemincount, instancemaxcount, initialmininstancecount" +
		" FROM activeschedule WHERE appid = ?")

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
	defer func() { _ = rows.Close() }()

	schedules := make(map[string]string)
	var id, appId string
	for rows.Next() {
		if err = rows.Scan(&id, &appId); err != nil {
			sdb.logger.Error("failed-get-active-schedules-scan", err, lager.Data{"query": query})
			return nil, err
		}
		schedules[appId] = id
	}
	return schedules, rows.Err()
}

func (sdb *ScalingEngineSQLDB) RemoveActiveSchedule(appId string) error {
	query := sdb.sqldb.Rebind("DELETE FROM activeschedule WHERE appid = ?")
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

	query := sdb.sqldb.Rebind("INSERT INTO activeschedule(appid, scheduleid, instancemincount, instancemaxcount, initialmininstancecount) " +
		" VALUES (?, ?, ?, ?, ?)")
	_, err = sdb.sqldb.Exec(query, appId, schedule.ScheduleId, schedule.InstanceMin, schedule.InstanceMax, schedule.InstanceMinInitial)

	if err != nil {
		sdb.logger.Error("failed-set-active-scheudle-insert", err, lager.Data{"appid": appId, "schedule": schedule})
	}
	return err
}

func (sdb *ScalingEngineSQLDB) GetDBStatus() sql.DBStats {
	return sdb.sqldb.Stats()
}
