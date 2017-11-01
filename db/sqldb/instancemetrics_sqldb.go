package sqldb

import (
	"database/sql"
	"time"

	"code.cloudfoundry.org/lager"
	. "github.com/lib/pq"

	"autoscaler/db"
	"autoscaler/models"
)

type InstanceMetricsSQLDB struct {
	logger lager.Logger
	url    string
	sqldb  *sql.DB
}

func NewInstanceMetricsSQLDB(url string, logger lager.Logger) (*InstanceMetricsSQLDB, error) {
	sqldb, err := sql.Open(db.PostgresDriverName, url)
	if err != nil {
		logger.Error("failed-open-instancemetrics-db", err, lager.Data{"url": url})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("failed-ping-instancemetrics-db", err, lager.Data{"url": url})
		return nil, err
	}

	return &InstanceMetricsSQLDB{
		sqldb:  sqldb,
		logger: logger,
		url:    url,
	}, nil
}

func (idb *InstanceMetricsSQLDB) Close() error {
	err := idb.sqldb.Close()
	if err != nil {
		idb.logger.Error("failed-close-instancemetrics-db", err, lager.Data{"url": idb.url})
		return err
	}
	return nil
}

func (idb *InstanceMetricsSQLDB) SaveMetric(metric *models.AppInstanceMetric) error {
	query := "INSERT INTO appinstancemetrics(appid, instanceindex, collectedat, name, unit, value, timestamp) values($1, $2, $3, $4, $5, $6, $7)"
	_, err := idb.sqldb.Exec(query, metric.AppId, metric.InstanceIndex, metric.CollectedAt, metric.Name, metric.Unit, metric.Value, metric.Timestamp)

	if err != nil {
		idb.logger.Error("failed-insert-instancemetric-into-appinstancemetrics-table", err, lager.Data{"query": query, "metric": metric})
	}
	return err
}

func (idb *InstanceMetricsSQLDB) SaveMetricsInBulk(metrics []*models.AppInstanceMetric) error {
	txn, err := idb.sqldb.Begin()
	if err != nil {
		idb.logger.Error("failed-to-start-transaction", err)
		return err
	}

	stmt, err := txn.Prepare(CopyIn("appinstancemetrics", "appid", "instanceindex", "collectedat", "name", "unit", "value", "timestamp"))
	if err != nil {
		idb.logger.Error("failed-to-prepare-statement", err)
		return err
	}
	for _, metric := range metrics {
		_, err := stmt.Exec(metric.AppId, metric.InstanceIndex, metric.CollectedAt, metric.Name, metric.Unit, metric.Value, metric.Timestamp)
		if err != nil {
			idb.logger.Error("failed-to-execute", err)
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		idb.logger.Error("failed-to-execute-statement", err)
		return err
	}

	err = stmt.Close()
	if err != nil {
		idb.logger.Error("failed-to-close-statement", err)
		return err
	}

	err = txn.Commit()
	if err != nil {
		idb.logger.Error("failed-to-commit-transaction", err)
		return err
	}

	return nil
}

func (idb *InstanceMetricsSQLDB) RetrieveInstanceMetrics(appid string, name string, start int64, end int64, orderType db.OrderType) ([]*models.AppInstanceMetric, error) {
	var orderStr string
	if orderType == db.ASC {
		orderStr = db.ASCSTR
	} else {
		orderStr = db.DESCSTR
	}
	query := "SELECT instanceindex, collectedat, unit, value, timestamp FROM appinstancemetrics WHERE " +
		" appid = $1 " +
		" AND name = $2 " +
		" AND timestamp >= $3" +
		" AND timestamp <= $4" +
		" ORDER BY timestamp " + orderStr + ", instanceindex"

	if end < 0 {
		end = time.Now().UnixNano()
	}

	rows, err := idb.sqldb.Query(query, appid, name, start, end)
	if err != nil {
		idb.logger.Error("failed-retrieve-instancemetrics-from-appinstancemetrics-table", err,
			lager.Data{"query": query, "appid": appid, "metricName": name, "start": start, "end": end, "orderType": orderType})
		return nil, err
	}
	defer rows.Close()

	mtrcs := []*models.AppInstanceMetric{}
	var index uint32
	var collectedAt, timestamp int64
	var unit, value string

	for rows.Next() {
		if err = rows.Scan(&index, &collectedAt, &unit, &value, &timestamp); err != nil {
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
	query := "DELETE FROM appinstancemetrics WHERE timestamp <= $1"
	_, err := idb.sqldb.Exec(query, before)
	if err != nil {
		idb.logger.Error("failed-prune-instancemetric-from-appinstancemetrics-table", err, lager.Data{"query": query, "before": before})
	}

	return err
}
