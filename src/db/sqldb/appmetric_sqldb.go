package sqldb

import (
	"code.cloudfoundry.org/lager"
	"database/sql"
	"eventgenerator/appmetric"
	_ "github.com/lib/pq"

	"db"
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
func (adb *AppMetricSQLDB) SaveAppMetric(appMetric *appmetric.AppMetric) error {
	query := "INSERT INTO app_metric(app_id, metric_type, unit, timestamp, value) values($1, $2, $3, $4, $5)"
	_, err := adb.sqldb.Exec(query, appMetric.AppId, appMetric.MetricType, appMetric.Unit, appMetric.Timestamp, appMetric.Value)

	if err != nil {
		adb.logger.Error("insert-metric-into-app-metric-table", err, lager.Data{"query": query, "appMetric": appMetric})
	}

	return err
}
