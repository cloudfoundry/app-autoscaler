package sqldb

import (
	"database/sql"
	"errors"

	"code.cloudfoundry.org/lager"
	"dataaggregator/appmetric"
	"dataaggregator/config"
	"dataaggregator/policy"
	"fmt"
	_ "github.com/lib/pq"
)

const PostgresDriverName = "postgres"

type SQLDB struct {
	policyDb       *sql.DB
	appMetricDb    *sql.DB
	logger         lager.Logger
	policyDbUrl    string
	appMetricDbUrl string
}

func NewSQLDB(conf *config.Config, logger lager.Logger) (*SQLDB, error) {
	sqldb := &SQLDB{}
	sqldb.policyDbUrl = conf.PolicyDbUrl
	sqldb.appMetricDbUrl = conf.AppMetricDbUrl
	sqldb.logger = logger

	var err error
	fmt.Println(conf.PolicyDbUrl)
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

	sqldb.appMetricDb, err = sql.Open(PostgresDriverName, conf.AppMetricDbUrl)
	if err != nil {
		logger.Error("open-appmetric-db", err)
		sqldb.Close()
		return nil, err
	}

	err = sqldb.appMetricDb.Ping()
	if err != nil {
		sqldb.appMetricDb = nil
		sqldb.Close()
		return nil, err
	}

	return sqldb, nil
}

func (db *SQLDB) Close() error {
	var hasError bool

	if db.policyDb != nil {
		err := db.policyDb.Close()
		if err != nil {
			db.logger.Error("Close-policy-db", err)
			hasError = true
		}
	}
	if db.appMetricDb != nil {
		err := db.appMetricDb.Close()
		if err != nil {
			db.logger.Error("Close-appmetric-db", err)
			hasError = true
		}
	}

	if hasError {
		return errors.New("Error closing policy db or appmetric db")
	}

	return nil
}

func (db *SQLDB) RetrievePolicies() ([]*policy.PolicyJson, error) {
	query := "SELECT app_id,policy_json FROM policy_json WHERE 1=1 "
	policyList := []*policy.PolicyJson{}
	rows, err := db.policyDb.Query(query)
	if err != nil {
		db.logger.Error("retrive-policy-list-from-policy_json-table", err,
			lager.Data{"query": query})
		return policyList, err
	}

	defer rows.Close()

	var appId string
	var policyStr string

	for rows.Next() {
		if err = rows.Scan(&appId, &policyStr); err != nil {
			db.logger.Error("scan-policy-from-search-result", err)
		}
		policy := policy.PolicyJson{
			AppId:     appId,
			PolicyStr: policyStr,
		}
		policyList = append(policyList, &policy)
	}
	return policyList, nil
}
func (db *SQLDB) SaveAppMetric(appMetric *appmetric.AppMetric) error {
	query := "INSERT INTO app_metric(app_id, metric_type, unit, timestamp, value) values($1, $2, $3, $4, $5)"
	_, err := db.appMetricDb.Exec(query, appMetric.AppId, appMetric.MetricType, appMetric.Unit, appMetric.Timestamp, appMetric.Value)

	if err != nil {
		db.logger.Error("insert-metric-into-app-metric-table", err, lager.Data{"query": query, "appMetric": appMetric})
	}

	return err
}
