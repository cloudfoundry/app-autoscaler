package sqldb

import (
	"autoscaler/db"
	"autoscaler/models"
	"database/sql"
	"encoding/json"
	"errors"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
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

func (pdb *PolicySQLDB) DeletePolicy(appId string) error {
	query := "DELETE FROM policy_json WHERE app_id = $1"
	_, err := pdb.sqldb.Exec(query, appId)
	if err != nil {
		pdb.logger.Error("failed-to-delete-application-details", err, lager.Data{"query": query, "appId": appId})
	}
	return err
}

func (pdb *PolicySQLDB) GetDBStatus() sql.DBStats {
	return pdb.sqldb.Stats()
}

func (pdb *PolicySQLDB) GetCustomMetricsCreds(appId string) (string, string, error) {
	var password string
	var username string
	query := "SELECT username,password from credentials WHERE id = $1"
	err := pdb.sqldb.QueryRow(query, appId).Scan(&username, &password)

	if err != nil {
		pdb.logger.Error("get-custom-metrics-creds-from-credentials-table", err, lager.Data{"query": query})
		return "", "", err
	}
	return username, password, nil
}

func (pdb *PolicySQLDB) ValidateCustomMetricsCreds(appid, username, password string) bool { //
	encryptedUsername, encryptedPassword, err := pdb.GetCustomMetricsCreds(appid)
	if err != nil {
		pdb.logger.Error("error-during-getting-binding-credentials", err)
		return false
	}
	isUsernameAuthenticated := bcrypt.CompareHashAndPassword([]byte(encryptedUsername), []byte(username))
	isPasswordAuthenticated := bcrypt.CompareHashAndPassword([]byte(encryptedPassword), []byte(password))
	if isPasswordAuthenticated == nil && isUsernameAuthenticated == nil { // password matching successfull
		return true
	}
	pdb.logger.Debug("failed-to-authorize-credentials")
	return false
}

func (pdb *PolicySQLDB) ValidateCustomMetricTypes(appGUID string, metricsConsumer *models.MetricsConsumer) (bool, error) {
	scalingPolicy, err := pdb.GetAppPolicy(appGUID)
	if err != nil {
		pdb.logger.Error("errorr-getting-policy", err, lager.Data{"appId": appGUID})
		return false, errors.New("not able to get policy details")
	}
	if err == nil && scalingPolicy == nil {
		pdb.logger.Debug("no-policy-found", lager.Data{"appId": appGUID})
		return false, errors.New("no policy found")
	}
	allowedMetricTypeSet := make(map[string]struct{}, len(scalingPolicy.ScalingRules))
	for _, metrictype := range scalingPolicy.ScalingRules {
		allowedMetricTypeSet[metrictype.MetricType] = struct{}{}
	}
	pdb.logger.Debug("allowed-metrics-types", lager.Data{"metrics": allowedMetricTypeSet})
	for _, metric := range metricsConsumer.CustomMetrics {
		_, ok := allowedMetricTypeSet[metric.Name]
		if !ok { // If any of the custom metrics is not defined during policy binding, it should fail
			pdb.logger.Info("unmatched-custom-metric-type", lager.Data{"metric": metric.Type})
			return false, errors.New("CustomMetric name does not match with metric defined in policy")
		}
	}
	return true, nil
}
