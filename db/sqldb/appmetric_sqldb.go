package sqldb

import (
	"context"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"github.com/uptrace/opentelemetry-go-extra/otelsqlx"

	"database/sql"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"github.com/jmoiron/sqlx"
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

	sqldb, err := otelsqlx.Open(database.DriverName, database.DataSourceName, otelsql.WithAttributes(database.OTELAttribute))
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
	sqldb.SetMaxIdleConns(int(dbConfig.MaxIdleConnections))
	sqldb.SetMaxOpenConns(int(dbConfig.MaxOpenConnections))
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
	if len(appMetrics) == 0 {
		return nil
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	txn, err := adb.sqldb.BeginTxx(ctx, nil)
	if err != nil {
		adb.logger.Error("failed-to-start-transaction", err)
		return err
	}

	sqlStr := "INSERT INTO app_metric(app_id, metric_type, unit, timestamp, value) VALUES (:app_id, :metric_type, :unit, :timestamp, :value)"

	_, err = txn.NamedExec(sqlStr, appMetrics)
	if err != nil {
		adb.logger.Error("failed-to-execute-statement", err)
		_ = txn.Rollback()
		return err
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
	defer func() { _ = rows.Close() }()
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
	return appMetricList, rows.Err()
}

func (adb *AppMetricSQLDB) PruneAppMetrics(ctx context.Context, before int64) error {
	query := adb.sqldb.Rebind("DELETE FROM app_metric WHERE timestamp <= ?")
	_, err := adb.sqldb.ExecContext(ctx, query, before)
	if err != nil {
		adb.logger.Error("prune-metrics-from-app_metric-table", err, lager.Data{"query": query, "before": before})
	}

	return err
}
func (adb *AppMetricSQLDB) GetDBStatus() sql.DBStats {
	return adb.sqldb.Stats()
}
