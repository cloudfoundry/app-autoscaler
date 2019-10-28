package sqldb

import (
	"autoscaler/models"
	"database/sql"
	"time"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"

	"autoscaler/db"
)

type BindingSQLDB struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sql.DB
}

func NewBindingSQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*BindingSQLDB, error) {
	sqldb, err := sql.Open(db.PostgresDriverName, dbConfig.URL)
	if err != nil {
		logger.Error("open-binding-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("ping-binding-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(dbConfig.MaxIdleConnections)
	sqldb.SetMaxOpenConns(dbConfig.MaxOpenConnections)

	return &BindingSQLDB{
		dbConfig: dbConfig,
		logger:   logger,
		sqldb:    sqldb,
	}, nil
}

func (bdb *BindingSQLDB) Close() error {
	err := bdb.sqldb.Close()
	if err != nil {
		bdb.logger.Error("Close-binding-db", err, lager.Data{"dbConfig": bdb.dbConfig})
		return err
	}
	return nil
}

func nullableString(s string) interface{} {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return s
}

func (bdb *BindingSQLDB) CreateServiceInstance(serviceInstance models.ServiceInstance) error {
	existingInstance, err := bdb.GetServiceInstance(serviceInstance.ServiceInstanceId)
	if err != nil {
		bdb.logger.Error("create-service-instance-get-existing", err, lager.Data{"serviceinstance": serviceInstance})
		return err
	}

	if existingInstance != nil {
		if existingInstance.OrgId == serviceInstance.OrgId && existingInstance.SpaceId == serviceInstance.SpaceId && existingInstance.DefaultPolicy == serviceInstance.DefaultPolicy && existingInstance.DefaultPolicyGuid == serviceInstance.DefaultPolicyGuid {
			return db.ErrAlreadyExists
		} else {
			return db.ErrConflict
		}

	}

	query := "INSERT INTO service_instance" +
		"(service_instance_id, org_id, space_id, default_policy, default_policy_guid) " +
		" VALUES($1, $2, $3, $4, $5)"
	_, err = bdb.sqldb.Exec(query, serviceInstance.ServiceInstanceId, serviceInstance.OrgId, serviceInstance.SpaceId, nullableString(serviceInstance.DefaultPolicy), nullableString(serviceInstance.DefaultPolicyGuid))

	if err != nil {
		bdb.logger.Error("create-service-instance", err, lager.Data{"query": query, "serviceinstance": serviceInstance})
	}
	return err
}

func (bdb *BindingSQLDB) GetServiceInstance(serviceInstanceId string) (*models.ServiceInstance, error) {
	query := "SELECT org_id, space_id, default_policy, default_policy_guid FROM service_instance WHERE service_instance_id = $1"

	rows, err := bdb.sqldb.Query(query, serviceInstanceId)
	if err != nil {
		bdb.logger.Error("get-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId})
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var (
			existingOrgId             string
			existingSpaceId           string
			existingDefaultPolicy     sql.NullString
			existingDefaultPolicyGuid sql.NullString
		)
		if err := rows.Scan(&existingOrgId, &existingSpaceId, &existingDefaultPolicy, &existingDefaultPolicyGuid); err != nil {
			bdb.logger.Error("get-default-policy", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId})
			return nil, err
		}
		return &models.ServiceInstance{
			ServiceInstanceId: serviceInstanceId,
			OrgId:             existingOrgId,
			SpaceId:           existingSpaceId,
			DefaultPolicy:     existingDefaultPolicy.String,
			DefaultPolicyGuid: existingDefaultPolicyGuid.String,
		}, nil
	}
	return nil, nil
}

func (bdb *BindingSQLDB) UpdateServiceInstance(serviceInstance models.ServiceInstance) error {
	query := "UPDATE service_instance SET default_policy = $2, default_policy_guid = $3 WHERE service_instance_id = $1"

	result, err := bdb.sqldb.Exec(query, serviceInstance.ServiceInstanceId, nullableString(serviceInstance.DefaultPolicy), nullableString(serviceInstance.DefaultPolicyGuid))
	if err != nil {
		bdb.logger.Error("update-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstance.ServiceInstanceId})
		return err
	}

	if rowsAffected, err := result.RowsAffected(); err != nil || rowsAffected == 0 {
		bdb.logger.Error("update-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstance.ServiceInstanceId, "rowsAffected": rowsAffected})
		return db.ErrDoesNotExist
	}

	return nil
}

func (bdb *BindingSQLDB) DeleteServiceInstance(serviceInstanceId string) error {
	query := "SELECT FROM service_instance WHERE service_instance_id = $1"
	rows, err := bdb.sqldb.Query(query, serviceInstanceId)
	if err != nil {
		bdb.logger.Error("delete-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId})
		return err
	}

	if rows.Next() {
		rows.Close()
		query = "DELETE FROM service_instance WHERE service_instance_id = $1"
		_, err = bdb.sqldb.Exec(query, serviceInstanceId)

		if err != nil {
			bdb.logger.Error("delete-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId})
		}
		return err
	}
	rows.Close()
	return db.ErrDoesNotExist
}

func (bdb *BindingSQLDB) CreateServiceBinding(bindingId string, serviceInstanceId string, appId string) error {
	query := "SELECT FROM binding WHERE app_id = $1"
	rows, err := bdb.sqldb.Query(query, appId)
	if err != nil {
		bdb.logger.Error("create-service-binding", err, lager.Data{"query": query, "appId": appId, "serviceId": serviceInstanceId, "bindingId": bindingId})
		return err
	}

	if rows.Next() {
		rows.Close()
		return db.ErrAlreadyExists
	}
	rows.Close()

	query = "INSERT INTO binding" +
		"(binding_id, service_instance_id, app_id, created_at) " +
		"VALUES($1, $2, $3, $4)"
	_, err = bdb.sqldb.Exec(query, bindingId, serviceInstanceId, appId, time.Now())

	if err != nil {
		bdb.logger.Error("create-service-binding", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId, "bindingid": bindingId, "appid": appId})
	}
	return err
}
func (bdb *BindingSQLDB) DeleteServiceBinding(bindingId string) error {
	query := "SELECT FROM binding WHERE binding_id = $1"
	rows, err := bdb.sqldb.Query(query, bindingId)
	if err != nil {
		bdb.logger.Error("delete-service-binding", err, lager.Data{"query": query, "bindingId": bindingId})
		return err
	}

	if rows.Next() {
		rows.Close()
		query = "DELETE FROM binding WHERE binding_id = $1"
		_, err = bdb.sqldb.Exec(query, bindingId)

		if err != nil {
			bdb.logger.Error("delete-service-binding", err, lager.Data{"query": query, "bindingid": bindingId})
		}
		return err
	}
	rows.Close()

	return db.ErrDoesNotExist
}
func (bdb *BindingSQLDB) DeleteServiceBindingByAppId(appId string) error {
	query := "DELETE FROM binding WHERE app_id = $1"
	_, err := bdb.sqldb.Exec(query, appId)

	if err != nil {
		bdb.logger.Error("delete-service-binding-by-appid", err, lager.Data{"query": query, "appId": appId})
		return err
	}
	return nil
}
func (bdb *BindingSQLDB) CheckServiceBinding(appId string) bool {
	var count int
	query := "SELECT COUNT(*) FROM binding WHERE app_id=$1"
	bdb.sqldb.QueryRow(query, appId).Scan(&count)
	return count > 0
}
func (bdb *BindingSQLDB) GetDBStatus() sql.DBStats {
	return bdb.sqldb.Stats()
}
func (bdb *BindingSQLDB) GetAppIdByBindingId(bindingId string) (string, error) {
	var appId string
	query := "SELECT app_id FROM binding WHERE binding_id=$1"
	err := bdb.sqldb.QueryRow(query, bindingId).Scan(&appId)
	if err != nil {
		bdb.logger.Error("get-appid-from-binding-table", err, lager.Data{"query": query, "bindingId": bindingId})
		return "", err
	}
	return appId, nil
}

func (bdb *BindingSQLDB) GetAppIdsByInstanceId(instanceId string) ([]string, error) {
	var appIds []string
	query := "SELECT app_id FROM binding WHERE service_instance_id=$1"
	rows, err := bdb.sqldb.Query(query, instanceId)
	if err != nil {
		bdb.logger.Error("get-appids-from-binding-table", err, lager.Data{"query": query, "instanceId": instanceId})
		return appIds, err
	}
	defer rows.Close()

	var appId string
	for rows.Next() {
		if err = rows.Scan(&appId); err != nil {
			bdb.logger.Error("scan-appids-from-binding-table", err)
			return nil, err
		}
		appIds = append(appIds, appId)
	}

	return appIds, nil
}

func (bdb *BindingSQLDB) CountServiceInstancesInOrg(orgId string) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM service_instance WHERE org_id=$1"
	err := bdb.sqldb.QueryRow(query, orgId).Scan(&count)
	if err != nil {
		bdb.logger.Error("count-service-instances-in-org", err, lager.Data{"query": query, "orgId": orgId})
		return 0, err
	}
	return count, nil
}
