package sqldb

import (
	"code.cloudfoundry.org/lager"
	"database/sql"
	_ "github.com/lib/pq"

	"autoscaler/db"
	"autoscaler/models"

	"time"
)

type HistorySQLDB struct {
	url    string
	logger lager.Logger
	sqldb  *sql.DB
}

func NewHistorySQLDB(url string, logger lager.Logger) (*HistorySQLDB, error) {
	sqldb, err := sql.Open(db.PostgresDriverName, url)
	if err != nil {
		logger.Error("open-scaling-history-db", err, lager.Data{"url": url})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("ping-scaling-history-db", err, lager.Data{"url": url})
		return nil, err
	}

	return &HistorySQLDB{
		url:    url,
		logger: logger,
		sqldb:  sqldb,
	}, nil
}

func (hdb *HistorySQLDB) Close() error {
	err := hdb.sqldb.Close()
	if err != nil {
		hdb.logger.Error("close-scaling-history-db", err, lager.Data{"url": hdb.url})
		return err
	}
	return nil
}

func (hdb *HistorySQLDB) SaveScalingHistory(history *models.AppScalingHistory) error {
	query := "INSERT INTO scalinghistory" +
		"(appid, timestamp, scalingtype, status, oldinstances, newinstances, reason, message, error) " +
		" VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	_, err := hdb.sqldb.Exec(query, history.AppId, history.Timestamp, history.ScalingType, history.Status,
		history.OldInstances, history.NewInstances, history.Reason, history.Message, history.Error)

	if err != nil {
		hdb.logger.Error("save-scaling-history", err, lager.Data{"query": query, "history": history})
	}
	return err
}

func (hdb *HistorySQLDB) RetrieveScalingHistories(appId string, start int64, end int64) ([]*models.AppScalingHistory, error) {
	query := "SELECT timestamp, scalingtype, status, oldinstances, newinstances, reason, message, error FROM scalinghistory WHERE" +
		" appid = $1 " +
		" AND timestamp >= $2" +
		" AND timestamp <= $3 ORDER BY timestamp"

	if end < 0 {
		end = time.Now().UnixNano()
	}

	histories := []*models.AppScalingHistory{}
	rows, err := hdb.sqldb.Query(query, appId, start, end)
	if err != nil {
		hdb.logger.Error("retrieve-scaling-histories", err,
			lager.Data{"query": query, "appid": appId, "start": start, "end": end})
		return nil, err
	}

	defer rows.Close()

	var timestamp int64
	var scalingType, status, oldInstances, newInstances int
	var reason, message, errorMsg string

	for rows.Next() {
		if err = rows.Scan(&timestamp, &scalingType, &status, &oldInstances, &newInstances, &reason, &message, &errorMsg); err != nil {
			hdb.logger.Error("retrieve-scaling-history-scan", err)
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
	return histories, nil
}

func (hdb *HistorySQLDB) CanScaleApp(appId string) (bool, error) {
	query := "SELECT expireat FROM scalingcooldown where appid = $1"
	rows, err := hdb.sqldb.Query(query, appId)
	if err != nil {
		hdb.logger.Error("can-scale-app-query-record", err, lager.Data{"query": query, "appid": appId})
		return false, err
	}
	defer rows.Close()

	if rows.Next() {
		var expireAt int64
		if err = rows.Scan(&expireAt); err != nil {
			hdb.logger.Error("can-scale-app-scan", err, lager.Data{"query": query, "appid": appId})
			return false, err
		}
		if expireAt < time.Now().UnixNano() {
			return true, nil
		} else {
			return false, nil
		}
	}

	return true, nil
}

func (hdb *HistorySQLDB) UpdateScalingCooldownExpireTime(appId string, expireAt int64) error {
	_, err := hdb.sqldb.Exec("DELETE FROM scalingcooldown where appid = $1", appId)
	if err != nil {
		hdb.logger.Error("update-scaling-cooldown-time-delete", err, lager.Data{"appid": appId})
		return err
	}

	_, err = hdb.sqldb.Exec("INSERT INTO scalingcooldown(appid, expireat) values($1, $2)", appId, expireAt)
	if err != nil {
		hdb.logger.Error("update-scaling-cooldown-time-insert", err, lager.Data{"appid": appId, "expireAt": expireAt})
		return err
	}
	return nil
}
