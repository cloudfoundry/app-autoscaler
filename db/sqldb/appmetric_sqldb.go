package sqldb

import (
	"autoscaler/db"
	"autoscaler/models"
	"code.cloudfoundry.org/lager"
	"database/sql"
	_ "github.com/lib/pq"
)

type AppMetricSQLDB struct {
	url    string
	logger lager.Logger
	sqldb  *sql.DB
}

func NewAppMetricSQLDB(url string, logger lager.Logger) (*AppMetricSQLDB, error) {
	appMetricDB := &AppMetricSQLDB{
		url:    url,
		logger: logger,
	}

	var err error

	appMetricDB.sqldb, err = sql.Open(db.PostgresDriverName, url)
	if err != nil {
		logger.Error("open-AppMetric-db", err, lager.Data{"url": url})
		return nil, err
	}

	err = appMetricDB.sqldb.Ping()
	if err != nil {
		appMetricDB.sqldb.Close()
		logger.Error("ping-AppMetric-db", err, lager.Data{"url": url})
		return nil, err
	}

	return appMetricDB, nil
}

func (adb *AppMetricSQLDB) Close() error {
	err := adb.sqldb.Close()
	if err != nil {
		adb.logger.Error("Close-AppMetric-db", err, lager.Data{"url": adb.url})
		return err
	}
	return nil
}
func (adb *AppMetricSQLDB) SaveAppMetric(appMetric *models.AppMetric) error {
	query := "INSERT INTO app_metric(app_id, metric_type, unit, timestamp, value) values($1, $2, $3, $4, $5)"
	_, err := adb.sqldb.Exec(query, appMetric.AppId, appMetric.MetricType, appMetric.Unit, appMetric.Timestamp, appMetric.Value)

	if err != nil {
		adb.logger.Error("insert-metric-into-app-metric-table", err, lager.Data{"query": query, "appMetric": appMetric})
	}

	return err
}
func (adb *AppMetricSQLDB) RetrieveAppMetrics(appIdP string, metricTypeP string, startP int64, endP int64) ([]*models.AppMetric, error) {
	query := "SELECT app_id,metric_type,value,unit,timestamp FROM app_metric WHERE app_id=$1 AND metric_type=$2 AND timestamp>=$3 AND timestamp<=$4 ORDER BY timestamp ASC"
	appMetricList := []*models.AppMetric{}
	rows, err := adb.sqldb.Query(query, appIdP, metricTypeP, startP, endP)
	if err != nil {
		adb.logger.Error("retrieve-app-metric-list-from-app_metric-table", err, lager.Data{"query": query})
		return nil, err
	}
	defer rows.Close()
	var appId string
	var metricType string
	var unit string
	var timestamp int64
	for rows.Next() {
		var value int64
		if err = rows.Scan(&appId, &metricType, &value, &unit, &timestamp); err != nil {
			adb.logger.Error("scan-appmetric-from-search-result", err)
			return nil, err
		}
		appMetric := &models.AppMetric{
			AppId:      appId,
			MetricType: metricType,
			Value:      &value,
			Unit:       unit,
			Timestamp:  timestamp,
		}
		appMetricList = append(appMetricList, appMetric)
	}
	return appMetricList, nil
}

func (adb *AppMetricSQLDB) PruneAppMetrics(before int64) error {
	query := "DELETE FROM app_metric WHERE timestamp <= $1"
	_, err := adb.sqldb.Exec(query, before)
	if err != nil {
		adb.logger.Error("prune-metrics-from-app_metric-table", err, lager.Data{"query": query, "before": before})
	}

	return err
}
