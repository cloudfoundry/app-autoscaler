package sqldb

import (
	"database/sql"
	"encoding/json"
	"time"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"

	"autoscaler/db"
	"autoscaler/models"
)

type MetricsSQLDB struct {
	sqldb  *sql.DB
	logger lager.Logger
	url    string
}

func NewMetricsSQLDB(url string, logger lager.Logger) (*MetricsSQLDB, error) {
	metricsDB := &MetricsSQLDB{
		logger: logger,
		url:    url,
	}

	var err error
	metricsDB.sqldb, err = sql.Open(db.PostgresDriverName, url)
	if err != nil {
		logger.Error("open-metrics-db", err, lager.Data{"url": url})
		return nil, err
	}

	err = metricsDB.sqldb.Ping()
	if err != nil {
		metricsDB.sqldb.Close()
		logger.Error("ping-metrics-db", err, lager.Data{"url": url})
		return nil, err
	}

	return metricsDB, nil
}

func (mdb *MetricsSQLDB) Close() error {
	err := mdb.sqldb.Close()
	if err != nil {
		mdb.logger.Error("close-metrics-db", err, lager.Data{"url": mdb.url})
		return err
	}
	return nil
}

func (mdb *MetricsSQLDB) SaveMetric(metric *models.Metric) error {
	value, err := json.Marshal(metric.Instances)
	if err != nil {
		mdb.logger.Error("marshal-instance-metrics", err, lager.Data{"metric": metric})
		return err
	}

	query := "INSERT INTO applicationmetrics(appid, name, unit, timestamp, value) values($1, $2, $3, $4, $5)"
	_, err = mdb.sqldb.Exec(query, metric.AppId, metric.Name, metric.Unit, metric.Timestamp, string(value))

	if err != nil {
		mdb.logger.Error("insert-metric-into-applicationmetrics-table", err, lager.Data{"query": query, "metric": metric})
	}

	return err
}

func (mdb *MetricsSQLDB) RetrieveMetrics(appid string, name string, start int64, end int64) ([]*models.Metric, error) {
	query := "SELECT unit, timestamp, value FROM applicationmetrics WHERE " +
		" appid = $1 " +
		" AND name = $2 " +
		" AND timestamp >= $3" +
		" AND timestamp <= $4 ORDER BY timestamp"

	if end < 0 {
		end = time.Now().UnixNano()
	}

	mtrcs := []*models.Metric{}
	rows, err := mdb.sqldb.Query(query, appid, name, start, end)
	if err != nil {
		mdb.logger.Error("retrive-metrics-from-applicationmetrics-table", err,
			lager.Data{"query": query, "appid": appid, "metricName": name, "start": start, "end": end})
		return nil, err
	}

	defer rows.Close()

	var unit string
	var timestamp int64
	var value []byte

	for rows.Next() {
		if err = rows.Scan(&unit, &timestamp, &value); err != nil {
			mdb.logger.Error("scan-metric-from-search-result", err)
			return nil, err
		}

		inst := []models.InstanceMetric{}
		err := json.Unmarshal(value, &inst)
		if err != nil {
			mdb.logger.Error("unmarshal-instance-metrics", err, lager.Data{"value": value})
			return nil, err
		}

		metric := models.Metric{
			AppId:     appid,
			Name:      name,
			Unit:      unit,
			Timestamp: timestamp,
			Instances: inst,
		}
		mtrcs = append(mtrcs, &metric)
	}
	return mtrcs, nil
}

func (mdb *MetricsSQLDB) PruneMetrics(before int64) error {
	query := "DELETE FROM applicationmetrics WHERE timestamp <= $1"
	_, err := mdb.sqldb.Exec(query, before)
	if err != nil {
		mdb.logger.Error("prune-metric-from-applicationmetrics-table", err, lager.Data{"query": query, "before": before})
	}

	return err
}
