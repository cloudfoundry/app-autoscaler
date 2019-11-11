package sqldb

import (
	"autoscaler/db"
	"autoscaler/models"
	"database/sql"
	"encoding/json"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"
)

type PolicySQLDB struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sql.DB
}

func NewPolicySQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*PolicySQLDB, error) {
	sqldb, err := sql.Open(db.PostgresDriverName, dbConfig.URL)
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
	query := "SELECT policy_json FROM policy_json WHERE app_id = $1"
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
	query := "INSERT INTO policy_json (app_id, policy_json, guid) VALUES ($1,$2, $3) " +
		"ON CONFLICT(app_id) DO UPDATE SET policy_json=EXCLUDED.policy_json, guid=EXCLUDED.guid"

	_, err := pdb.sqldb.Exec(query, appId, policyJSON, policyGuid)
	if err != nil {
		pdb.logger.Error("save-app-policy", err, lager.Data{"query": query, "app_id": appId, "policyJSON": policyJSON, "policyGuid": policyGuid})
	}
	return err
}

func (pdb *PolicySQLDB) SetOrUpdateDefaultAppPolicy(boundApps []string, oldPolicyGuid string, newPolicy string, newPolicyGuid string) ([]string, error) {
	if len(boundApps) == 0 && oldPolicyGuid == "" {
		return nil, nil
	}

	var modifiedApps []string

	tx, err := pdb.sqldb.Begin()
	if err != nil {
		pdb.logger.Error("set-or-update-app-policies-begin-transaction", err, lager.Data{"newPolicyGuid": newPolicyGuid, "newPolicy": newPolicy, "oldPolicyGuid": oldPolicyGuid})
		return nil, err
	}
	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr == nil {
			// a rollback was executed!
			pdb.logger.Info("set-or-update-app-policies-transaction-rollback-executed", lager.Data{"newPolicyGuid": newPolicyGuid, "newPolicy": newPolicy, "oldPolicyGuid": oldPolicyGuid})
		} else {
			if rollbackErr != sql.ErrTxDone {
				pdb.logger.Error("set-or-update-app-policies-transaction-rollback-error", err, lager.Data{"newPolicyGuid": newPolicyGuid, "newPolicy": newPolicy, "oldPolicyGuid": oldPolicyGuid})
			}
		}
	}()

	// first replace an already existing default policy
	if oldPolicyGuid != "" {
		query := "UPDATE policy_json SET guid = $2, policy_json = $3 WHERE guid = $1 RETURNING app_id"

		rows, err := tx.Query(query, oldPolicyGuid, newPolicyGuid, newPolicy)
		if err != nil {
			pdb.logger.Error("rollback-set-or-update-app-policies", err, lager.Data{"query": query, "oldPolicyGuid": oldPolicyGuid, "newPolicyGuid": newPolicyGuid, "newPolicy": newPolicy})
			return nil, err
		}
		defer func() {
			if err := rows.Close(); err != nil {
				pdb.logger.Error("rollback-set-or-update-app-policies-close-rows", err, lager.Data{"query": query, "oldPolicyGuid": oldPolicyGuid, "newPolicyGuid": newPolicyGuid, "newPolicy": newPolicy})
			}
		}()

		var id string
		for rows.Next() {
			if err = rows.Scan(&id); err != nil {
				pdb.logger.Error("get-appids-scan", err)
				return nil, err
			}
			modifiedApps = append(modifiedApps, id)

			// a modified app cannot be an app which has no default policy, so remove it from bound apps
			if len(boundApps) > 0 {
				filteredApps := boundApps[:0]
				for _, x := range boundApps {
					if x != id {
						filteredApps = append(filteredApps, x)
					}
				}
				boundApps = filteredApps
			}
		}
	}

	// set the default policy on apps which do not have one
	for _, appId := range boundApps {
		query := "INSERT INTO policy_json (app_id, policy_json, guid) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING"
		res, err := tx.Exec(query, appId, newPolicy, newPolicyGuid)
		if err != nil {
			pdb.logger.Error("set-default-app-policy", err, lager.Data{"query": query, "appId": appId, "newPolicyGuid": newPolicyGuid, "newPolicy": newPolicy})
			return nil, err
		}
		count, err := res.RowsAffected()
		if err != nil {
			pdb.logger.Error("set-default-app-policy-determine-rows-affected", err, lager.Data{"query": query, "appId": appId, "newPolicyGuid": newPolicyGuid, "newPolicy": newPolicy})
			return nil, err
		}
		if count == 1 {
			modifiedApps = append(modifiedApps, appId)
		}
	}

	err = tx.Commit()
	if err != nil {
		pdb.logger.Error("update-app-policies-commit", err, lager.Data{"newPolicyGuid": newPolicyGuid, "newPolicy": newPolicy})
		return nil, err
	}
	return modifiedApps, nil
}

func (pdb *PolicySQLDB) DeletePolicy(appId string) error {
	query := "DELETE FROM policy_json WHERE app_id = $1"
	_, err := pdb.sqldb.Exec(query, appId)
	if err != nil {
		pdb.logger.Error("failed-to-delete-application-details", err, lager.Data{"query": query, "appId": appId})
	}
	return err
}

func (pdb *PolicySQLDB) DeletePoliciesByPolicyGuid(policyGuid string) ([]string, error) {
	var appIds []string

	tx, err := pdb.sqldb.Begin()
	if err != nil {
		pdb.logger.Error("delete-policies-by-policy-guid-begin-transaction", err, lager.Data{"policyGuid": policyGuid})
		return nil, err
	}
	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr == nil {
			// a rollback was executed!
			pdb.logger.Info("delete-policies-by-policy-guid-transaction-rollback-executed", lager.Data{"policyGuid": policyGuid})
		} else {
			if rollbackErr != sql.ErrTxDone {
				pdb.logger.Error("delete-policies-by-policy-guid-transaction-rollback-error", err, lager.Data{"policyGuid": policyGuid})
			}
		}
	}()

	query := "DELETE FROM policy_json WHERE guid = $1 RETURNING app_id"
	rows, err := tx.Query(query, policyGuid)
	if err != nil {
		pdb.logger.Error("failed-to-delete-policies-by-policy-guid", err, lager.Data{"query": query, "policyGuid": policyGuid})
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			pdb.logger.Error("delete-policies-by-policy-guid-close-rows", err, lager.Data{"query": query, "policyGuid": policyGuid})
		}
	}()

	var id string
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			pdb.logger.Error("get-appids-scan", err)
			return nil, err
		}
		appIds = append(appIds, id)
	}

	err = tx.Commit()
	if err != nil {
		pdb.logger.Error("delete-policies-by-policy-guid-commit", err, lager.Data{"policyGuid": policyGuid})
		return nil, err
	}
	return appIds, nil
}

func (pdb *PolicySQLDB) GetDBStatus() sql.DBStats {
	return pdb.sqldb.Stats()
}

func (pdb *PolicySQLDB) GetCredential(appId string) (*models.Credential, error) {
	var password string
	var username string
	query := "SELECT username,password from credentials WHERE id = $1"
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
	query := "INSERT INTO credentials (id, username, password, updated_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP) " +
		"ON CONFLICT(id) DO UPDATE SET username=EXCLUDED.username, password=EXCLUDED.password, updated_at=CURRENT_TIMESTAMP"
	_, err := pdb.sqldb.Exec(query, appId, cred.Username, cred.Password)
	if err != nil {
		pdb.logger.Error("save-custom-metric-credential", err, lager.Data{"query": query, "app_id": appId})
	}
	return err
}
func (pdb *PolicySQLDB) DeleteCredential(appId string) error {
	query := "DELETE FROM credentials WHERE id = $1"
	_, err := pdb.sqldb.Exec(query, appId)
	if err != nil {
		pdb.logger.Error("failed-to-delete-custom-metric-credential", err, lager.Data{"query": query, "appId": appId})
	}
	return err
}
