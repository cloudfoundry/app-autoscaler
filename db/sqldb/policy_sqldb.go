package sqldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"github.com/uptrace/opentelemetry-go-extra/otelsqlx"

	"code.cloudfoundry.org/lager/v3"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

var _ db.PolicyDB = &PolicySQLDB{}

type PolicySQLDB struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sqlx.DB
}

func (pdb *PolicySQLDB) Ping() error {
	return pdb.sqldb.Ping()
}

func NewPolicySQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*PolicySQLDB, error) {
	database, err := db.GetConnection(dbConfig.URL)
	if err != nil {
		return nil, err
	}

	sqldb, err := otelsqlx.Open(database.DriverName, database.DataSourceName, otelsql.WithAttributes(database.OTELAttribute))
	if err != nil {
		logger.Error("open-policy-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		_ = sqldb.Close()
		logger.Error("ping-policy-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(int(dbConfig.MaxIdleConnections))
	sqldb.SetMaxOpenConns(int(dbConfig.MaxOpenConnections))
	sqldb.SetConnMaxIdleTime(dbConfig.ConnectionMaxIdleTime)

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

func (pdb *PolicySQLDB) GetAppIds(ctx context.Context) (map[string]bool, error) {
	appIds := make(map[string]bool)
	query := "SELECT app_id FROM policy_json"

	rows, err := pdb.sqldb.QueryContext(ctx, query)
	if err != nil {
		pdb.logger.Error("get-appids-from-policy-table", err, lager.Data{"query": query})
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var id string
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			pdb.logger.Error("get-appids-scan", err)
			return nil, err
		}
		appIds[id] = true
	}
	return appIds, rows.Err()
}

func (pdb *PolicySQLDB) RetrievePolicies() ([]*models.PolicyJson, error) {
	query := "SELECT app_id,policy_json FROM policy_json WHERE 1=1 "
	policyList := []*models.PolicyJson{}
	rows, err := pdb.sqldb.Query(query)
	if err != nil {
		pdb.logger.Error("retrieve-policy-list-from-policy_json-table", err,
			lager.Data{"query": query})
		return policyList, err
	}

	defer func() { _ = rows.Close() }()

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
	return policyList, rows.Err()
}

func (pdb *PolicySQLDB) GetAppPolicy(ctx context.Context, appId string) (*models.ScalingPolicy, error) {
	var policyJson []byte
	query := pdb.sqldb.Rebind("SELECT policy_json FROM policy_json WHERE app_id =?")
	err := pdb.sqldb.QueryRowContext(ctx, query, appId).Scan(&policyJson)

	if errors.Is(err, sql.ErrNoRows) {
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

func (pdb *PolicySQLDB) SaveAppPolicy(ctx context.Context, appId string, policy *models.ScalingPolicy, policyGuid string) error {
	var query string
	queryPrefix := "INSERT INTO policy_json (app_id, policy_json, guid) VALUES (?,?,?) "
	switch pdb.sqldb.DriverName() {
	case "pgx":
		query = pdb.sqldb.Rebind(queryPrefix + "ON CONFLICT(app_id) DO UPDATE SET policy_json=EXCLUDED.policy_json, guid=EXCLUDED.guid")
	case "mysql":
		query = pdb.sqldb.Rebind(queryPrefix + "ON DUPLICATE KEY UPDATE policy_json=VALUES(policy_json), guid=VALUES(guid)")
	}
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("SaveAppPolicy failed to marshal policy:  %w", err)
	}
	_, err = pdb.sqldb.ExecContext(ctx, query, appId, policyJSON, policyGuid)
	if err != nil {
		pdb.logger.Error("save-app-policy", err, lager.Data{"query": query, "app_id": appId, "policyJSON": policyJSON, "policyGuid": policyGuid})
	}
	return err
}

func (pdb *PolicySQLDB) SetOrUpdateDefaultAppPolicy(ctx context.Context, boundApps []string, oldPolicyGuid string, policy *models.ScalingPolicy, newPolicyGuid string) ([]string, error) {
	if len(boundApps) == 0 && oldPolicyGuid == "" {
		return nil, nil
	}

	var modifiedApps []string
	newPolicyBytes, err := json.Marshal(policy)
	if err != nil {
		pdb.logger.Error("marshal policy", err, lager.Data{"newPolicyGuid": newPolicyGuid, "policy": policy, "oldPolicyGuid": oldPolicyGuid})
		return nil, err
	}
	policyJson := string(newPolicyBytes)

	tx, err := pdb.sqldb.Beginx()
	if err != nil {
		pdb.logger.Error("set-or-update-app-policies-begin-transaction", err, lager.Data{"newPolicyGuid": newPolicyGuid, "policyJson": policyJson, "oldPolicyGuid": oldPolicyGuid})
		return nil, err
	}

	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr == nil {
			// a rollback was executed!
			pdb.logger.Info("set-or-update-app-policies-transaction-rollback-executed", lager.Data{"newPolicyGuid": newPolicyGuid, "policyJson": policyJson, "oldPolicyGuid": oldPolicyGuid})
		} else {
			if errors.Is(err, sql.ErrTxDone) {
				pdb.logger.Error("set-or-update-app-policies-transaction-rollback-error", err, lager.Data{"newPolicyGuid": newPolicyGuid, "policyJson": policyJson, "oldPolicyGuid": oldPolicyGuid})
			}
		}
	}()

	// first replace an already existing default policy
	if oldPolicyGuid != "" {
		// determine which apps had the existing default policy set
		query := tx.Rebind("SELECT app_id FROM policy_json WHERE guid = ?")

		rows, err := tx.QueryContext(ctx, query, oldPolicyGuid)
		if err != nil {
			pdb.logger.Error("rollback-set-or-update-app-policies", err, lager.Data{"query": query, "oldPolicyGuid": oldPolicyGuid})
			return nil, err
		}

		defer func() {
			if err := rows.Close(); err != nil {
				pdb.logger.Error("rollback-set-or-update-app-policies-close-rows", err, lager.Data{"query": query, "oldPolicyGuid": oldPolicyGuid})
			}
			_ = rows.Err()
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

		// now replace the default policy
		query = tx.Rebind("UPDATE policy_json SET guid = ?, policy_json = ? WHERE guid = ?")

		_, err = tx.ExecContext(ctx, query, newPolicyGuid, policyJson, oldPolicyGuid)
		if err != nil {
			pdb.logger.Error("rollback-set-or-update-app-policies", err, lager.Data{"query": query, "oldPolicyGuid": oldPolicyGuid, "newPolicyGuid": newPolicyGuid, "policyJson": policyJson})
			return nil, err
		}
	}
	// set the default policy on apps which do not have one
	for _, appId := range boundApps {
		var query string
		queryPrefix := "INSERT INTO policy_json (app_id, policy_json, guid) VALUES (?, ?, ?)"
		switch pdb.sqldb.DriverName() {
		case "pgx":
			query = tx.Rebind(queryPrefix + " ON CONFLICT DO NOTHING")
		case "mysql":
			query = tx.Rebind(queryPrefix + " ON DUPLICATE KEY UPDATE app_id = app_id")
		}
		res, err := tx.Exec(query, appId, policyJson, newPolicyGuid)
		if err != nil {
			pdb.logger.Error("set-default-app-policy", err, lager.Data{"query": query, "appId": appId, "newPolicyGuid": newPolicyGuid, "policyJson": policyJson})
			return nil, err
		}
		count, err := res.RowsAffected()
		if err != nil {
			pdb.logger.Error("set-default-app-policy-determine-rows-affected", err, lager.Data{"query": query, "appId": appId, "newPolicyGuid": newPolicyGuid, "policyJson": policyJson})
			return nil, err
		}
		if count == 1 {
			modifiedApps = append(modifiedApps, appId)
		}
	}

	err = tx.Commit()
	if err != nil {
		pdb.logger.Error("update-app-policies-commit", err, lager.Data{"newPolicyGuid": newPolicyGuid, "policyJson": policyJson})
		return nil, err
	}
	return modifiedApps, nil
}

func (pdb *PolicySQLDB) DeletePolicy(ctx context.Context, appId string) error {
	query := pdb.sqldb.Rebind("DELETE FROM policy_json WHERE app_id =?")
	_, err := pdb.sqldb.ExecContext(ctx, query, appId)
	if err != nil {
		pdb.logger.Error("failed-to-delete-application-details", err, lager.Data{"query": query, "appId": appId})
	}
	return err
}

func (pdb *PolicySQLDB) DeletePoliciesByPolicyGuid(ctx context.Context, policyGuid string) ([]string, error) {
	var appIds []string

	tx, err := pdb.sqldb.Beginx()
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
			if errors.Is(rollbackErr, sql.ErrTxDone) {
				pdb.logger.Error("delete-policies-by-policy-guid-transaction-rollback-error", err, lager.Data{"policyGuid": policyGuid})
			}
		}
	}()

	// first determine for which apps the policy will be removed
	query := tx.Rebind("SELECT app_id FROM policy_json WHERE guid = ?")
	rows, err := tx.QueryContext(ctx, query, policyGuid)
	if err != nil {
		pdb.logger.Error("failed-to-delete-policies-by-policy-guid", err, lager.Data{"query": query, "policyGuid": policyGuid})
		return nil, err
	}
	defer func() {
		_ = rows.Err()
		err := rows.Close()
		if err != nil {
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

	// then actually delete them
	query = tx.Rebind("DELETE FROM policy_json WHERE guid = ?")
	_, err = tx.ExecContext(ctx, query, policyGuid)
	if err != nil {
		pdb.logger.Error("failed-to-delete-policies-by-policy-guid", err, lager.Data{"query": query, "policyGuid": policyGuid})
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			pdb.logger.Error("delete-policies-by-policy-guid-close-rows", err, lager.Data{"query": query, "policyGuid": policyGuid})
		}
	}()

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
func (pdb *PolicySQLDB) SaveCredential(ctx context.Context, appId string, cred models.Credential) error {
	var query string
	queryPrefix := "INSERT INTO credentials (id, username, password, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP) "
	switch pdb.sqldb.DriverName() {
	case "pgx":
		query = pdb.sqldb.Rebind(queryPrefix + "ON CONFLICT(id) DO UPDATE SET username=EXCLUDED.username, password=EXCLUDED.password, updated_at=CURRENT_TIMESTAMP")
	case "mysql":
		query = pdb.sqldb.Rebind(queryPrefix + "ON DUPLICATE KEY UPDATE username=VALUES(username), password=VALUES(password), updated_at=CURRENT_TIMESTAMP")
	}
	_, err := pdb.sqldb.ExecContext(ctx, query, appId, cred.Username, cred.Password)
	if err != nil {
		pdb.logger.Error("save-custom-metric-credential", err, lager.Data{"query": query, "app_id": appId})
	}
	return err
}
func (pdb *PolicySQLDB) DeleteCredential(ctx context.Context, appId string) error {
	query := pdb.sqldb.Rebind("DELETE FROM credentials WHERE id =?")
	_, err := pdb.sqldb.ExecContext(ctx, query, appId)
	if err != nil {
		pdb.logger.Error("failed-to-delete-custom-metric-credential", err, lager.Data{"query": query, "appId": appId})
	}
	return err
}
