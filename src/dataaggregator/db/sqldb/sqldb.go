package sqldb

import (
	"database/sql"
	"errors"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"

	"dataaggregator/config"
	"dataaggregator/policy"
	"fmt"
)

const PostgresDriverName = "postgres"

type SQLDB struct {
	policyDb    *sql.DB
	logger      lager.Logger
	policyDbUrl string
}

func NewSQLDB(conf *config.Config, logger lager.Logger) (*SQLDB, error) {
	sqldb := &SQLDB{}
	sqldb.policyDbUrl = conf.PolicyDbUrl
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

	if hasError {
		return errors.New("Error closing policy db")
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
