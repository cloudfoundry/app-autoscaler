package sqldb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"

	"metricscollector/config"
	"metricscollector/metrics"
)

const PostgresDriverName = "postgres"

type SQLDB struct {
	metricsDb *sql.DB
	policyDb  *sql.DB
	logger    lager.Logger
	conf      *config.DbConfig
}

func NewSQLDB(conf *config.DbConfig, logger lager.Logger) (*SQLDB, error) {
	sqldb := &SQLDB{}
	sqldb.conf = conf
	sqldb.logger = logger

	var err error

	sqldb.metricsDb, err = sql.Open(PostgresDriverName, conf.MetricsDbUrl)
	if err != nil {
		logger.Error("open-metrics-db", err)
		return nil, err
	}

	err = sqldb.metricsDb.Ping()
	if err != nil {
		sqldb.metricsDb = nil
		return nil, err
	}

	sqldb.policyDb, err = sql.Open(PostgresDriverName, conf.PolicyDbUrl)
	if err != nil {
		logger.Error("open-policy-db", err)
		sqldb.Close()
		return nil, err
	}

	err = sqldb.policyDb.Ping()
	if err != nil {
		sqldb.policyDb = nil
		sqldb.Close()
		return nil, err
	}

	return sqldb, nil
}

func (db *SQLDB) Close() error {
	var hasError bool
	if db.metricsDb != nil {
		err := db.metricsDb.Close()
		if err != nil {
			db.logger.Error("Close-metrics-db", err)
			hasError = true
		}
	}

	if db.policyDb != nil {
		err := db.policyDb.Close()
		if err != nil {
			db.logger.Error("Close-policy-db", err)
			hasError = true
		}
	}

	if hasError {
		return errors.New("Error closing metrics or policy db")
	}

	return nil
}

func (db *SQLDB) SaveMetric(metric *metrics.Metric) error {
	value, err := json.Marshal(metric.Instances)
	if err != nil {
		db.logger.Error("marshal-instance-metrics", err, lager.Data{"metric": metric})
		return err
	}

	query := "INSERT INTO applicationmetrics(appid, name, unit, timestamp, value) values($1, $2, $3, $4, $5)"
	_, err = db.metricsDb.Exec(query, metric.AppId, metric.Name, metric.Unit, metric.TimeStamp, string(value))

	if err != nil {
		db.logger.Error("insert-metric-into-applicationmetrics-table", err, lager.Data{"query": query, "metric": metric})
	}

	return err
}

func (db *SQLDB) RetrieveMetrics(appid string, name string, start int64, end int64) ([]*metrics.Metric, error) {
	query := "SELECT unit, timestamp, value FROM applicationmetrics WHERE " +
		" appid = $1 " +
		" AND name = $2 " +
		" AND timestamp >= $3" +
		" AND timestamp <= $4"

	if end < 0 {
		end = time.Now().UnixNano()
	}

	mtrcs := []*metrics.Metric{}
	rows, err := db.metricsDb.Query(query, appid, name, start, end)
	if err != nil {
		db.logger.Error("retrive-metrics-from-applicationmetrics-table", err,
			lager.Data{"query": query, "appid": appid, "metricName": name, "start": start, "end": end})
		return mtrcs, err
	}

	defer rows.Close()

	var unit string
	var timestamp int64
	var value []byte

	for rows.Next() {
		if err = rows.Scan(&unit, &timestamp, &value); err != nil {
			db.logger.Error("scan-metric-from-search-result", err)
		}

		inst := []metrics.InstanceMetric{}
		err := json.Unmarshal(value, &inst)
		if err != nil {
			db.logger.Error("unmarshal-instance-metrics", err, lager.Data{"value": value})
			return mtrcs, err
		}

		metric := metrics.Metric{
			AppId:     appid,
			Name:      name,
			Unit:      unit,
			TimeStamp: timestamp,
			Instances: inst,
		}
		mtrcs = append(mtrcs, &metric)
	}
	return mtrcs, nil
}

func (db *SQLDB) PruneMetrics(before int64) error {
	query := "DELETE FROM applicationmetrics WHERE timestamp <= $1"
	_, err := db.metricsDb.Exec(query, before)
	if err != nil {
		db.logger.Error("prune-metric-from-applicationmetrics-table", err, lager.Data{"query": query, "before": before})
	}

	return err
}

func (db *SQLDB) GetAppIds() ([]string, error) {
	appIds := []string{}
	query := "SELECT app_id FROM policy_json"

	rows, err := db.policyDb.Query(query)
	if err != nil {
		db.logger.Error("retrive-appids-from-policy-table", err, lager.Data{"query": query})
		return nil, err
	}

	defer rows.Close()

	var id string
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			db.logger.Error("scan-appid-from-search-result", err)
			return nil, err
		}
		appIds = append(appIds, id)
	}
	return appIds, nil
}
