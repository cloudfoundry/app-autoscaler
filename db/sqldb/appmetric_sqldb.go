package sqldb

import (
	"context"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"database/sql"
	"time"
)

type AppMetricSQLDB struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sqlx.DB
}

func NewAppMetricSQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*AppMetricSQLDB, error) {
	var err error
	database, err := db.GetConnection(dbConfig.URL)
	if err != nil {
		return nil, err
	}

	sqldb, err := sqlx.Open(database.DriverName, database.DSN)
	if err != nil {
		logger.Error("open-AppMetric-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		_ = sqldb.Close()
		logger.Error("ping-AppMetric-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}
	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(dbConfig.MaxIdleConnections)
	sqldb.SetMaxOpenConns(dbConfig.MaxOpenConnections)
	sqldb.SetConnMaxIdleTime(dbConfig.ConnectionMaxIdleTime)

	return &AppMetricSQLDB{
		dbConfig: dbConfig,
		logger:   logger,
		sqldb:    sqldb,
	}, nil
}

func (adb *AppMetricSQLDB) Close() error {
	return adb.sqldb.Close()
}

func (adb *AppMetricSQLDB) SaveAppMetric(appMetric *models.AppMetric) error {
	query := adb.sqldb.Rebind("INSERT INTO app_metric(app_id, metric_type, unit, timestamp, value) values(?, ?, ?, ?, ?)")
	_, err := adb.sqldb.Exec(query, appMetric.AppId, appMetric.MetricType, appMetric.Unit, appMetric.Timestamp, appMetric.Value)

	if err != nil {
		adb.logger.Error("insert-metric-into-app-metric-table", err, lager.Data{"query": query, "appMetric": appMetric})
	}

	return err
}
func (adb *AppMetricSQLDB) SaveAppMetricsInBulk(appMetrics []*models.AppMetric) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	txn, err := adb.sqldb.BeginTx(ctx, nil)
	if err != nil {
		adb.logger.Error("failed-to-start-transaction", err)
		return err
	}

	switch adb.sqldb.DriverName() {
	case "postgres":
		stmt, err := txn.Prepare(pq.CopyIn("app_metric", "app_id", "metric_type", "unit", "timestamp", "value"))
		if err != nil {
			adb.logger.Error("failed-to-prepare-statement", err)
			_ = txn.Rollback()
			return err
		}
		for _, appMetric := range appMetrics {
			_, err := stmt.Exec(appMetric.AppId, appMetric.MetricType, appMetric.Unit, appMetric.Timestamp, appMetric.Value)
			if err != nil {
				adb.logger.Error("failed-to-execute", err)
				_ = txn.Rollback()
				return err
			}
		}
		_, err = stmt.Exec()
		if err != nil {
			adb.logger.Error("failed-to-execute-statement", err)
			_ = txn.Rollback()
			return err
		}
		err = stmt.Close()
		if err != nil {
			adb.logger.Error("failed-to-close-statement", err)
			_ = txn.Rollback()
			return err
		}

	case "mysql":
		sqlStr := "INSERT INTO app_metric(app_id,metric_type,unit,timestamp,value)VALUES"
		vals := []interface{}{}
		if len(appMetrics) == 0 {
			err = txn.Rollback()
			if err != nil {
				return err
			}
			return nil
		}
		for _, appMetric := range appMetrics {
			sqlStr += "(?, ?, ?, ?, ?),"
			vals = append(vals, appMetric.AppId, appMetric.MetricType, appMetric.Unit, appMetric.Timestamp, appMetric.Value)
		}
		sqlStr = strings.TrimSuffix(sqlStr, ",")

		stmt, err := txn.Prepare(sqlStr)
		if err != nil {
			adb.logger.Error("failed-to-prepare-statement", err)
			_ = txn.Rollback()
			return err
		}

		_, err = stmt.Exec(vals...)
		if err != nil {
			adb.logger.Error("failed-to-execute-statement", err)
			_ = txn.Rollback()
			return err
		}
		err = stmt.Close()
		if err != nil {
			adb.logger.Error("failed-to-close-statement", err)
			_ = txn.Rollback()
			return err
		}
	}

	err = txn.Commit()
	if err != nil {
		adb.logger.Error("failed-to-commit-transaction", err)
		_ = txn.Rollback()
		return err
	}

	return nil
}
func (adb *AppMetricSQLDB) RetrieveAppMetrics(appIdP string, metricTypeP string, startP int64, endP int64, orderType db.OrderType) ([]*models.AppMetric, error) {
	var orderStr string
	if orderType == db.ASC {
		orderStr = db.ASCSTR
	} else {
		orderStr = db.DESCSTR
	}

	if endP < 0 {
		endP = time.Now().UnixNano()
	}

	query := adb.sqldb.Rebind("SELECT app_id,metric_type,value,unit,timestamp FROM app_metric WHERE app_id=? AND metric_type=? AND timestamp>=? AND timestamp<=? ORDER BY timestamp " + orderStr)
	appMetricList := []*models.AppMetric{}
	rows, err := adb.sqldb.Query(query, appIdP, metricTypeP, startP, endP)
	if err != nil {
		adb.logger.Error("retrieve-app-metric-list-from-app_metric-table", err, lager.Data{"query": query})
		return nil, err
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	var appId string
	var metricType string
	var unit string
	var value string
	var timestamp int64

	for rows.Next() {
		if err = rows.Scan(&appId, &metricType, &value, &unit, &timestamp); err != nil {
			adb.logger.Error("scan-appmetric-from-search-result", err)
			return nil, err
		}
		appMetric := &models.AppMetric{
			AppId:      appId,
			MetricType: metricType,
			Value:      value,
			Unit:       unit,
			Timestamp:  timestamp,
		}
		appMetricList = append(appMetricList, appMetric)
	}
	return appMetricList, nil
}

func (adb *AppMetricSQLDB) PruneAppMetrics(before int64) error {
	query := adb.sqldb.Rebind("DELETE FROM app_metric WHERE timestamp <= ?")
	_, err := adb.sqldb.Exec(query, before)
	if err != nil {
		adb.logger.Error("prune-metrics-from-app_metric-table", err, lager.Data{"query": query, "before": before})
	}

	return err
}
func (adb *AppMetricSQLDB) GetDBStatus() sql.DBStats {
	return adb.sqldb.Stats()
}
