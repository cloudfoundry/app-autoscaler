package sqldb

import (
	"autoscaler/db"
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
	. "github.com/lib/pq"
	"github.com/jmoiron/sqlx"

	"context"
	"database/sql"
	"time"
	"strings"
)

type InstanceMetricsSQLDB struct {
	logger   lager.Logger
	dbConfig db.DatabaseConfig
	sqldb    *sqlx.DB
}

func NewInstanceMetricsSQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*InstanceMetricsSQLDB, error) {
	database, err := db.GetConnection(dbConfig.URL)
	if err != nil {
		return nil, err
	}

	sqldb, err := sqlx.Open(database.DriverName, database.DSN)
	if err != nil {
		logger.Error("failed-open-instancemetrics-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("failed-ping-instancemetrics-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(dbConfig.MaxIdleConnections)
	sqldb.SetMaxOpenConns(dbConfig.MaxOpenConnections)

	return &InstanceMetricsSQLDB{
		sqldb:    sqldb,
		logger:   logger,
		dbConfig: dbConfig,
	}, nil
}

func (idb *InstanceMetricsSQLDB) Close() error {
	err := idb.sqldb.Close()
	if err != nil {
		idb.logger.Error("failed-close-instancemetrics-db", err, lager.Data{"dbConfig": idb.dbConfig})
		return err
	}
	return nil
}

func (idb *InstanceMetricsSQLDB) SaveMetric(metric *models.AppInstanceMetric) error {
	query := idb.sqldb.Rebind("INSERT INTO appinstancemetrics(appid, instanceindex, collectedat, name, unit, value, timestamp) values(?, ?, ?, ?, ?, ?, ?)")
	_, err := idb.sqldb.Exec(query, metric.AppId, metric.InstanceIndex, metric.CollectedAt, metric.Name, metric.Unit, metric.Value, metric.Timestamp)

	if err != nil {
		idb.logger.Error("failed-insert-instancemetric-into-appinstancemetrics-table", err, lager.Data{"query": query, "metric": metric})
	}
	return err
}

func (idb *InstanceMetricsSQLDB) SaveMetricsInBulk(metrics []*models.AppInstanceMetric) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	txn, err := idb.sqldb.BeginTx(ctx, nil)
	if err != nil {
		idb.logger.Error("failed-to-start-transaction", err)
		return err
	}
	switch idb.sqldb.DriverName() {
	case "postgres":
		stmt, err := txn.Prepare(CopyIn("appinstancemetrics", "appid", "instanceindex", "collectedat", "name", "unit", "value", "timestamp"))
		if err != nil {
			idb.logger.Error("failed-to-prepare-statement", err)
			txn.Rollback()
			return err
		}
		for _, metric := range metrics {
			_, err := stmt.Exec(metric.AppId, metric.InstanceIndex, metric.CollectedAt, metric.Name, metric.Unit, metric.Value, metric.Timestamp)
			if err != nil {
				idb.logger.Error("failed-to-execute", err)
				txn.Rollback()
				return err
			}
		}

		_, err = stmt.Exec()
		if err != nil {
			idb.logger.Error("failed-to-execute-statement", err)
			txn.Rollback()
			return err
		}

		err = stmt.Close()
		if err != nil {
			idb.logger.Error("failed-to-close-statement", err)
			txn.Rollback()
			return err
		}
	case "mysql":
		sqlStr :="INSERT INTO appinstancemetrics(appid, instanceindex, collectedat, name, unit, value, timestamp)VALUES"
		vals := []interface{}{}
		if metrics == nil || len(metrics) == 0 {
			txn.Rollback()
			return nil
		}
		for _, metric := range metrics {
			sqlStr += "(?, ?, ?, ?, ?, ?, ?),"
			vals = append(vals, metric.AppId, metric.InstanceIndex, metric.CollectedAt, metric.Name, metric.Unit, metric.Value, metric.Timestamp)
		}
		sqlStr = strings.TrimSuffix(sqlStr, ",")

		stmt, err := txn.Prepare(sqlStr)
		if err != nil {
			idb.logger.Error("failed-to-prepare-statement", err)
			txn.Rollback()
			return err
		}

		_, err = stmt.Exec(vals...)
		if err != nil {
			idb.logger.Error("failed-to-execute-statement", err)
			txn.Rollback()
			return err
		}
		err = stmt.Close()
		if err != nil {
			idb.logger.Error("failed-to-close-statement", err)
			txn.Rollback()
			return err
		}
	}

	err = txn.Commit()
	if err != nil {
		idb.logger.Error("failed-to-commit-transaction", err)
		txn.Rollback()
		return err
	}

	return nil
}

func (idb *InstanceMetricsSQLDB) RetrieveInstanceMetrics(appid string, instanceIndex int, name string, start int64, end int64, orderType db.OrderType) ([]*models.AppInstanceMetric, error) {
	var orderStr string
	if orderType == db.ASC {
		orderStr = db.ASCSTR
	} else {
		orderStr = db.DESCSTR
	}
	query := idb.sqldb.Rebind("SELECT instanceindex, collectedat, unit, value, timestamp FROM appinstancemetrics WHERE " +
		" appid = ? " +
		" AND name = ? " +
		" AND timestamp >= ?" +
		" AND timestamp <= ?" +
		" ORDER BY timestamp " + orderStr + ", instanceindex")

	queryByInstanceIndex := idb.sqldb.Rebind("SELECT instanceindex, collectedat, unit, value, timestamp FROM appinstancemetrics WHERE " +
		" appid = ? " +
		" AND instanceindex = ?" +
		" AND name = ? " +
		" AND timestamp >= ?" +
		" AND timestamp <= ?" +
		" ORDER BY timestamp " + orderStr)

	if end < 0 {
		end = time.Now().UnixNano()
	}
	var rows *sql.Rows
	var err error
	if instanceIndex >= 0 {
		rows, err = idb.sqldb.Query(queryByInstanceIndex, appid, instanceIndex, name, start, end)
		if err != nil {
			idb.logger.Error("failed-retrieve-instancemetrics-from-appinstancemetrics-table", err,
				lager.Data{"query": query, "appid": appid, "instanceindex": instanceIndex, "metricName": name, "start": start, "end": end, "orderType": orderType})
			return nil, err
		}
	} else {
		rows, err = idb.sqldb.Query(query, appid, name, start, end)
		if err != nil {
			idb.logger.Error("failed-retrieve-instancemetrics-from-appinstancemetrics-table", err,
				lager.Data{"query": query, "appid": appid, "metricName": name, "start": start, "end": end, "orderType": orderType})
			return nil, err
		}
	}

	defer rows.Close()

	mtrcs := []*models.AppInstanceMetric{}
	var index uint32
	var collectedAt, timestamp int64
	var unit, value string

	for rows.Next() {
		if err := rows.Scan(&index, &collectedAt, &unit, &value, &timestamp); err != nil {
			idb.logger.Error("failed-scan-instancemetric-from-search-result", err)
			return nil, err
		}

		length := len(mtrcs)
		if (length > 0) && (timestamp == mtrcs[length-1].Timestamp) && (index == mtrcs[length-1].InstanceIndex) {
			continue
		}

		metric := models.AppInstanceMetric{
			AppId:         appid,
			InstanceIndex: index,
			CollectedAt:   collectedAt,
			Name:          name,
			Unit:          unit,
			Value:         value,
			Timestamp:     timestamp,
		}
		mtrcs = append(mtrcs, &metric)
	}
	return mtrcs, nil
}
func (idb *InstanceMetricsSQLDB) PruneInstanceMetrics(before int64) error {
	query := idb.sqldb.Rebind("DELETE FROM appinstancemetrics WHERE timestamp <= ?")
	_, err := idb.sqldb.Exec(query, before)
	if err != nil {
		idb.logger.Error("failed-prune-instancemetric-from-appinstancemetrics-table", err, lager.Data{"query": query, "before": before})
	}

	return err
}
func (idb *InstanceMetricsSQLDB) GetDBStatus() sql.DBStats {
	return idb.sqldb.Stats()
}
