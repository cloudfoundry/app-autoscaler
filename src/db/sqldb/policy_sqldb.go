package sqldb

import (
	"database/sql"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"

	"db"
)

const PostgresDriverName = "postgres"

type PolicySQLDB struct {
	url    string
	logger lager.Logger
	sqldb  *sql.DB
}

func NewPolicySQLDB(url string, logger lager.Logger) (*PolicySQLDB, error) {
	policyDB := &PolicySQLDB{
		url:    url,
		logger: logger,
	}

	var err error

	policyDB.sqldb, err = sql.Open(db.PostgresDriverName, url)
	if err != nil {
		logger.Error("open-policy-db", err, lager.Data{"url": url})
		return nil, err
	}

	err = policyDB.sqldb.Ping()
	if err != nil {
		policyDB.sqldb.Close()
		policyDB.sqldb = nil
		logger.Error("ping-policy-db", err, lager.Data{"url": url})
		return nil, err
	}

	return policyDB, nil
}

func (pdb *PolicySQLDB) Close() error {
	if pdb.sqldb != nil {
		err := pdb.sqldb.Close()
		if err != nil {
			pdb.logger.Error("Close-policy-db", err, lager.Data{"url": pdb.url})
			return err
		}
	}
	return nil
}

func (pdb *PolicySQLDB) GetAppIds() (map[string]bool, error) {
	appIds := make(map[string]bool)
	query := "SELECT app_id FROM policy_json"

	rows, err := pdb.sqldb.Query(query)
	if err != nil {
		pdb.logger.Error("retrive-appids-from-policy-table", err, lager.Data{"query": query})
		return nil, err
	}
	defer rows.Close()

	var id string
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			pdb.logger.Error("scan-appid-from-search-result", err)
			return nil, err
		}
		appIds[id] = true
	}
	return appIds, nil
}
