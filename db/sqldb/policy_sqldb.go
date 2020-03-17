package sqldb

import (
	"autoscaler/db"
	"autoscaler/models"
	"database/sql"
	"encoding/json"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type PolicySQLDB struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sqlx.DB
}

func NewPolicySQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*PolicySQLDB, error) {
	database, err := db.GetConnection(dbConfig.URL)
	if err != nil {
		return nil, err
	}

	sqldb, err := sqlx.Open(database.DriverName, database.DSN)
	if err != nil {
		logger.Error("open-policy-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("ping-policy-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(dbConfig.MaxIdleConnections)
	sqldb.SetMaxOpenConns(dbConfig.MaxOpenConnections)

	return &PolicySQLDB{
		dbConfig: dbConfig,
		logger:   logger,
		sqldb:    sqldb,
	}, nil
}

func (pdb *PolicySQLDB) Close() error {
	err := pdb.sqldb.Close()
	if err != nil {
		pdb.logger.Error("Close-policy-db", err, lager.Data{"dbConfig": pdb.dbConfig})
		return err
	}
	return nil
}

func (pdb *PolicySQLDB) GetAppIds() (map[string]bool, error) {
	appIds := make(map[string]bool)
	query := "SELECT app_id FROM policy_json"

	rows, err := pdb.sqldb.Query(query)
	if err != nil {
		pdb.logger.Error("get-appids-from-policy-table", err, lager.Data{"query": query})
		return nil, err
	}
	defer rows.Close()

	var id string
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			pdb.logger.Error("get-appids-scan", err)
			return nil, err
		}
		appIds[id] = true
	}
	return appIds, nil
}

func (pdb *PolicySQLDB) RetrievePolicies() ([]*models.PolicyJson, error) {
	query := "SELECT app_id,policy_json FROM policy_json WHERE 1=1 "
	policyList := []*models.PolicyJson{}
	rows, err := pdb.sqldb.Query(query)
	if err != nil {
		pdb.logger.Error("retrive-policy-list-from-policy_json-table", err,
			lager.Data{"query": query})
		return policyList, err
	}

	defer rows.Close()

	var appId string
	var policyStr string

	for rows.Next() {
		if err = rows.Scan(&appId, &policyStr); err != nil {
			pdb.logger.Error("scan-policy-from-search-result", err)
			return nil, err
		}
		policyJson := models.PolicyJson{
			AppId:     appId,
			PolicyStr: policyStr,
		}
		policyList = append(policyList, &policyJson)
	}
	return policyList, nil
}

func (pdb *PolicySQLDB) GetAppPolicy(appId string) (*models.ScalingPolicy, error) {
	var policyJson []byte
	query := pdb.sqldb.Rebind("SELECT policy_json FROM policy_json WHERE app_id =?")
	err := pdb.sqldb.QueryRow(query, appId).Scan(&policyJson)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		pdb.logger.Error("get-app-policy-from-policy-table", err, lager.Data{"query": query, "appid": appId})
		return nil, err
	}

	scalingPolicy := &models.ScalingPolicy{}
	err = json.Unmarshal(policyJson, scalingPolicy)
	if err != nil {
		pdb.logger.Error("get-app-policy-unmarshal", err, lager.Data{"policyJson": string(policyJson)})
		return nil, err
	}
	return scalingPolicy, nil
}

func (pdb *PolicySQLDB) SaveAppPolicy(appId string, policyJSON string, policyGuid string) error {
	var query string
	queryPrefix := "INSERT INTO policy_json (app_id, policy_json, guid) VALUES (?,?,?) "
	switch pdb.sqldb.DriverName(){
	case "postgres":
		query = pdb.sqldb.Rebind(queryPrefix + "ON CONFLICT(app_id) DO UPDATE SET policy_json=EXCLUDED.policy_json, guid=EXCLUDED.guid")
	case "mysql":
		query = pdb.sqldb.Rebind(queryPrefix + "ON DUPLICATE KEY UPDATE policy_json=VALUES(policy_json), guid=VALUES(guid)")
	}
	_, err := pdb.sqldb.Exec(query, appId, policyJSON, policyGuid)
	if err != nil {
		pdb.logger.Error("save-app-policy", err, lager.Data{"query": query, "app_id": appId, "policyJSON": policyJSON, "policyGuid": policyGuid})
	}
	return err
}

func (pdb *PolicySQLDB) DeletePolicy(appId string) error {
	query := pdb.sqldb.Rebind("DELETE FROM policy_json WHERE app_id =?")
	_, err := pdb.sqldb.Exec(query, appId)
	if err != nil {
		pdb.logger.Error("failed-to-delete-application-details", err, lager.Data{"query": query, "appId": appId})
	}
	return err
}

func (pdb *PolicySQLDB) GetDBStatus() sql.DBStats {
	return pdb.sqldb.Stats()
}

func (pdb *PolicySQLDB) GetCredential(appId string) (*models.Credential, error) {
	var password string
	var username string
	query := pdb.sqldb.Rebind("SELECT username,password from credentials WHERE id =?")
	err := pdb.sqldb.QueryRow(query, appId).Scan(&username, &password)
	if err != nil {
		pdb.logger.Error("get-custom-metrics-creds-from-credentials-table", err, lager.Data{"query": query})
		return nil, err
	}
	return &models.Credential{
		Username: username,
		Password: password,
	}, nil
}
func (pdb *PolicySQLDB) SaveCredential(appId string, cred models.Credential) error {
	var query string
	queryPrefix := "INSERT INTO credentials (id, username, password, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP) "
	switch pdb.sqldb.DriverName() {
	case "postgres":
		query = pdb.sqldb.Rebind(queryPrefix + "ON CONFLICT(id) DO UPDATE SET username=EXCLUDED.username, password=EXCLUDED.password, updated_at=CURRENT_TIMESTAMP")
	case "mysql":
		query = pdb.sqldb.Rebind(queryPrefix + "ON DUPLICATE KEY UPDATE username=VALUES(username), password=VALUES(password), updated_at=CURRENT_TIMESTAMP")
	}
	_, err := pdb.sqldb.Exec(query, appId, cred.Username, cred.Password)
	if err != nil {
		pdb.logger.Error("save-custom-metric-credential", err, lager.Data{"query": query, "app_id": appId})
	}
	return err
}
func (pdb *PolicySQLDB) DeleteCredential(appId string) error {
	query := pdb.sqldb.Rebind("DELETE FROM credentials WHERE id =?")
	_, err := pdb.sqldb.Exec(query, appId)
	if err != nil {
		pdb.logger.Error("failed-to-delete-custom-metric-credential", err, lager.Data{"query": query, "appId": appId})
	}
	return err
}
