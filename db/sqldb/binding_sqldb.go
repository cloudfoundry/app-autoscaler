package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"github.com/uptrace/opentelemetry-go-extra/otelsqlx"

	"code.cloudfoundry.org/lager/v3"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
)

type BindingSQLDB struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sqlx.DB
}

var _ db.BindingDB = &BindingSQLDB{}

func NewBindingSQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*BindingSQLDB, error) {
	database, err := db.GetConnection(dbConfig.URL)
	if err != nil {
		return nil, err
	}

	sqldb, err := otelsqlx.Open(database.DriverName, database.DataSourceName, otelsql.WithAttributes(database.OTELAttribute))
	if err != nil {
		logger.Error("open-binding-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		_ = sqldb.Close()
		logger.Error("ping-binding-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(int(dbConfig.MaxIdleConnections))
	sqldb.SetMaxOpenConns(int(dbConfig.MaxOpenConnections))
	sqldb.SetConnMaxIdleTime(dbConfig.ConnectionMaxIdleTime)

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

func nullableString(s string) *string {
	if s == "" {
		return nil
	} else {
		return &s
	}
}

func (bdb *BindingSQLDB) CreateServiceInstance(ctx context.Context, serviceInstance models.ServiceInstance) error {
	existingInstance, err := bdb.GetServiceInstance(context.Background(), serviceInstance.ServiceInstanceId)
	if err != nil && !errors.Is(err, db.ErrDoesNotExist) {
		bdb.logger.Error("create-service-instance-get-existing", err, lager.Data{"serviceinstance": serviceInstance})
		return err
	}

	if existingInstance != nil {
		if *existingInstance == serviceInstance {
			return db.ErrAlreadyExists
		} else {
			return db.ErrConflict
		}
	}

	query := bdb.sqldb.Rebind("INSERT INTO service_instance" +
		"(service_instance_id, org_id, space_id, default_policy, default_policy_guid) " +
		" VALUES(?, ?, ?, ?, ?)")

	_, err = bdb.sqldb.ExecContext(ctx, query, serviceInstance.ServiceInstanceId, serviceInstance.OrgId, serviceInstance.SpaceId, nullableString(serviceInstance.DefaultPolicy), nullableString(serviceInstance.DefaultPolicyGuid))

	if err != nil {
		bdb.logger.Error("create-service-instance", err, lager.Data{"query": query, "serviceinstance": serviceInstance})
	}
	return err
}

type bdServiceInstance struct {
	ServiceInstanceId string         `db:"service_instance_id"`
	OrgId             string         `db:"org_id"`
	SpaceId           string         `db:"space_id"`
	DefaultPolicy     sql.NullString `db:"default_policy"`
	DefaultPolicyGuid sql.NullString `db:"default_policy_guid"`
}

func (bdb *BindingSQLDB) GetServiceInstance(ctx context.Context, serviceInstanceId string) (*models.ServiceInstance, error) {
	query := bdb.sqldb.Rebind("SELECT * FROM service_instance WHERE service_instance_id = ?")
	result := bdServiceInstance{}
	err := bdb.sqldb.GetContext(ctx, &result, query, serviceInstanceId)
	if err != nil {
		bdb.logger.Error("get-service-instance", err, lager.Data{"query": query, "serviceInstanceId": serviceInstanceId})
		if errors.Is(err, sql.ErrNoRows) {
			return nil, db.ErrDoesNotExist
		}
		return nil, err
	}

	return &models.ServiceInstance{
		ServiceInstanceId: result.ServiceInstanceId,
		OrgId:             result.OrgId,
		SpaceId:           result.SpaceId,
		DefaultPolicy:     result.DefaultPolicy.String,
		DefaultPolicyGuid: result.DefaultPolicyGuid.String,
	}, nil
}

func (bdb *BindingSQLDB) GetServiceInstanceByAppId(appId string) (*models.ServiceInstance, error) {
	serviceInstanceId, err := bdb.GetServiceInstanceIdByAppId(appId)
	if err != nil {
		bdb.logger.Error("get-service-instance-for-app-id", err, lager.Data{"appId": appId})
		return nil, err
	}
	return bdb.GetServiceInstance(context.Background(), serviceInstanceId)
}

func (bdb *BindingSQLDB) GetServiceInstanceIdByAppId(appId string) (string, error) {
	query := bdb.sqldb.Rebind("SELECT service_instance_id FROM binding WHERE app_id = ?")

	serviceInstanceId := ""
	err := bdb.sqldb.Get(&serviceInstanceId, query, appId)
	if err != nil {
		bdb.logger.Error("get-service-instance-for-app-id", err, lager.Data{"query": query, "appId": appId})
		if errors.Is(err, sql.ErrNoRows) {
			return serviceInstanceId, db.ErrDoesNotExist
		}
		return serviceInstanceId, err
	}
	return serviceInstanceId, nil
}

func (bdb *BindingSQLDB) UpdateServiceInstance(ctx context.Context, serviceInstance models.ServiceInstance) error {
	query := bdb.sqldb.Rebind("UPDATE service_instance SET default_policy = ?, default_policy_guid = ? WHERE service_instance_id = ?")

	result, err := bdb.sqldb.ExecContext(ctx, query, nullableString(serviceInstance.DefaultPolicy), nullableString(serviceInstance.DefaultPolicyGuid), serviceInstance.ServiceInstanceId)
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

func (bdb *BindingSQLDB) DeleteServiceInstance(ctx context.Context, serviceInstanceId string) error {
	query := bdb.sqldb.Rebind("SELECT * FROM service_instance WHERE service_instance_id =?")
	rows, err := bdb.sqldb.Query(query, serviceInstanceId)
	if err != nil {
		bdb.logger.Error("delete-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId})
		return err
	}

	defer func() { _ = rows.Close() }()

	if rows.Next() {
		query = bdb.sqldb.Rebind("DELETE FROM service_instance WHERE service_instance_id =?")
		_, err = bdb.sqldb.Exec(query, serviceInstanceId)

		if err != nil {
			bdb.logger.Error("delete-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId})
		}
		return err
	}

	err = rows.Err()
	if err != nil {
		bdb.logger.Error("delete-service-instance-row", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId})
		return err
	}

	return db.ErrDoesNotExist
}

func (bdb *BindingSQLDB) CreateServiceBinding(ctx context.Context, bindingId string, serviceInstanceId string, appId string, customMetricsStrategy string) error {
	err := bdb.isBindingExists(ctx, bindingId, serviceInstanceId, appId)
	if err != nil {
		return err
	}
	query := bdb.sqldb.Rebind("INSERT INTO binding" +
		"(binding_id, service_instance_id, app_id, created_at, custom_metrics_strategy) " +
		"VALUES(?, ?, ?, ?,?)")
	_, err = bdb.sqldb.ExecContext(ctx, query, bindingId, serviceInstanceId, appId, time.Now(), nullableString(customMetricsStrategy))

	if err != nil {
		bdb.logger.Error("create-service-binding", err, lager.Data{"query": query, "serviceInstanceId": serviceInstanceId, "bindingId": bindingId, "appId": appId, "customMetricsStrategy": customMetricsStrategy})
		return err
	}
	return nil
}

func (bdb *BindingSQLDB) isBindingExists(ctx context.Context, bindingId string, serviceInstanceId string, appId string) error {
	query := bdb.sqldb.Rebind("SELECT * FROM binding WHERE app_id =?")
	rows, err := bdb.sqldb.QueryContext(ctx, query, appId)
	if err != nil {
		bdb.logger.Error("is-binding-already-exists", err, lager.Data{"query": query, "appId": appId, "serviceId": serviceInstanceId, "bindingId": bindingId})
		return err
	}

	defer func() { _ = rows.Close() }()

	if rows.Next() {
		return db.ErrAlreadyExists
	}

	err = rows.Err()
	if err != nil {
		bdb.logger.Error("is-binding-already-exists", err, lager.Data{"query": query, "appId": appId, "serviceId": serviceInstanceId, "bindingId": bindingId})
		return err
	}
	return nil
}

type dbServiceBinding struct {
	ServiceBindingID      string         `db:"binding_id"`
	ServiceInstanceID     string         `db:"service_instance_id"`
	AppID                 string         `db:"app_id"`
	CustomMetricsStrategy sql.NullString `db:"custom_metrics_strategy"`
}

func (bdb *BindingSQLDB) GetServiceBinding(ctx context.Context, serviceBindingId string) (*models.ServiceBinding, error) {
	logger := bdb.logger.Session("get-service-binding", lager.Data{"serviceBindingId": serviceBindingId})

	dbServiceBinding := &dbServiceBinding{}
	query := bdb.sqldb.Rebind("SELECT binding_id, service_instance_id, app_id, custom_metrics_strategy FROM binding WHERE binding_id =?")

	err := bdb.sqldb.GetContext(ctx, dbServiceBinding, query, serviceBindingId)
	if err != nil {
		logger.Error("query", err, lager.Data{"query": query})
		if errors.Is(err, sql.ErrNoRows) {
			return nil, db.ErrDoesNotExist
		}
		return nil, err
	}
	return &models.ServiceBinding{
		ServiceBindingID:      dbServiceBinding.ServiceBindingID,
		ServiceInstanceID:     dbServiceBinding.ServiceInstanceID,
		AppID:                 dbServiceBinding.AppID,
		CustomMetricsStrategy: dbServiceBinding.CustomMetricsStrategy.String,
	}, nil
}

func (bdb *BindingSQLDB) DeleteServiceBinding(ctx context.Context, bindingId string) error {
	query := bdb.sqldb.Rebind("SELECT * FROM binding WHERE binding_id =?")
	rows, err := bdb.sqldb.QueryContext(ctx, query, bindingId)
	if err != nil {
		bdb.logger.Error("delete-service-binding", err, lager.Data{"query": query, "bindingId": bindingId})
		return err
	}

	defer func() { _ = rows.Close() }()

	if rows.Next() {
		query = bdb.sqldb.Rebind("DELETE FROM binding WHERE binding_id =?")
		_, err = bdb.sqldb.ExecContext(ctx, query, bindingId)

		if err != nil {
			bdb.logger.Error("delete-service-binding", err, lager.Data{"query": query, "bindingid": bindingId})
		}
		return err
	}

	err = rows.Err()
	if err != nil {
		bdb.logger.Error("delete-service-binding-row", err, lager.Data{"query": query, "bindingId": bindingId})
		return err
	}

	return db.ErrDoesNotExist
}
func (bdb *BindingSQLDB) DeleteServiceBindingByAppId(ctx context.Context, appId string) error {
	query := bdb.sqldb.Rebind("DELETE FROM binding WHERE app_id =?")
	_, err := bdb.sqldb.ExecContext(ctx, query, appId)

	if err != nil {
		bdb.logger.Error("delete-service-binding-by-appid", err, lager.Data{"query": query, "appId": appId})
		return err
	}
	return nil
}

func (bdb *BindingSQLDB) getServiceBindingByAppId(ctx context.Context, appId string) (*models.ServiceBinding, error) {
	dbServiceBinding := &dbServiceBinding{}
	query := bdb.sqldb.Rebind("SELECT  binding_id, service_instance_id, app_id, custom_metrics_strategy FROM binding WHERE app_id =?")
	err := bdb.sqldb.GetContext(ctx, dbServiceBinding, query, appId)
	if err != nil {
		bdb.logger.Debug("get-service-binding-by-appid", lager.Data{"query": query, "appId": appId, "error": err})
		if errors.Is(err, sql.ErrNoRows) {
			return nil, db.ErrDoesNotExist
		}
		return nil, err
	}
	return &models.ServiceBinding{
		ServiceBindingID:      dbServiceBinding.ServiceBindingID,
		ServiceInstanceID:     dbServiceBinding.ServiceInstanceID,
		AppID:                 dbServiceBinding.AppID,
		CustomMetricsStrategy: dbServiceBinding.CustomMetricsStrategy.String,
	}, nil
}
func (bdb *BindingSQLDB) CheckServiceBinding(appId string) bool {
	var count int
	query := bdb.sqldb.Rebind("SELECT COUNT(*) FROM binding WHERE app_id=?")
	err := bdb.sqldb.QueryRow(query, appId).Scan(&count)
	if err != nil {
		bdb.logger.Error("check-service-binding-by-appid", err, lager.Data{"query": query, "appId": appId})
	}
	return count > 0
}
func (bdb *BindingSQLDB) GetDBStatus() sql.DBStats {
	return bdb.sqldb.Stats()
}

func (bdb *BindingSQLDB) GetAppIdByBindingId(ctx context.Context, bindingId string) (string, error) {
	var appId string
	query := bdb.sqldb.Rebind("SELECT app_id FROM binding WHERE binding_id=?")
	err := bdb.sqldb.QueryRowContext(ctx, query, bindingId).Scan(&appId)
	if err != nil {
		bdb.logger.Error("get-appid-from-binding-table", err, lager.Data{"query": query, "bindingId": bindingId})
		return "", err
	}
	return appId, nil
}

func (bdb *BindingSQLDB) GetAppIdsByInstanceId(ctx context.Context, instanceId string) ([]string, error) {
	var appIds []string
	query := bdb.sqldb.Rebind("SELECT app_id FROM binding WHERE service_instance_id = ?")
	rows, err := bdb.sqldb.QueryContext(ctx, query, instanceId)
	if err != nil {
		bdb.logger.Error("get-appids-from-binding-table", err, lager.Data{"query": query, "instanceId": instanceId})
		return appIds, err
	}

	defer func() { _ = rows.Close() }()

	var appId string
	for rows.Next() {
		if err = rows.Scan(&appId); err != nil {
			bdb.logger.Error("scan-appids-from-binding-table", err)
			return nil, err
		}
		appIds = append(appIds, appId)
	}

	return appIds, rows.Err()
}

func (bdb *BindingSQLDB) CountServiceInstancesInOrg(orgId string) (int, error) {
	var count int
	query := bdb.sqldb.Rebind("SELECT COUNT(*) FROM service_instance WHERE org_id=?")
	err := bdb.sqldb.QueryRow(query, orgId).Scan(&count)
	if err != nil {
		bdb.logger.Error("count-service-instances-in-org", err, lager.Data{"query": query, "orgId": orgId})
		return 0, err
	}
	return count, nil
}

func (bdb *BindingSQLDB) GetBindingIdsByInstanceId(ctx context.Context, instanceId string) ([]string, error) {
	var bindingIds []string
	query := bdb.sqldb.Rebind("SELECT binding_id FROM binding WHERE service_instance_id=?")
	rows, err := bdb.sqldb.QueryContext(ctx, query, instanceId)
	if err != nil {
		bdb.logger.Error("get-appids-from-binding-table", err, lager.Data{"query": query, "instanceId": instanceId})
		return bindingIds, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var bindingId string
		if err = rows.Scan(&bindingId); err != nil {
			bdb.logger.Error("scan-bindingids-from-binding-table", err)
			return nil, err
		}
		bindingIds = append(bindingIds, bindingId)
	}

	return bindingIds, rows.Err()
}

func (bdb *BindingSQLDB) IsAppBoundToSameAutoscaler(ctx context.Context, metricSubmitterAppId string, appToScaleId string) (bool, error) {
	serviceInstanceId, err := bdb.GetServiceInstanceIdByAppId(metricSubmitterAppId)
	if err != nil {
		bdb.logger.Error("get-service-instance-by-appId", err, lager.Data{"appId": metricSubmitterAppId})
		return false, err
	}
	if serviceInstanceId == "" {
		bdb.logger.Error("no-service-instance-found-by-appId", err, lager.Data{"appId": metricSubmitterAppId, "serviceInstanceId": serviceInstanceId})
		return false, nil
	}
	// find all apps which are bound to the same service instance
	appIds, err := bdb.GetAppIdsByInstanceId(ctx, serviceInstanceId)
	if err != nil {
		bdb.logger.Error("get-apps-by-service-instance-id", err, lager.Data{"serviceInstanceId": serviceInstanceId})
		return false, err
	}

	if len(appIds) == 0 {
		bdb.logger.Error("no-apps-bounded-with-serviceInstance", err, lager.Data{"serviceInstanceId": serviceInstanceId})
		return false, nil
	}
	// check if the app to scale is in the list of apps bound to the same service instance and return true .otherwise return false
	for _, app := range appIds {
		if app == appToScaleId {
			return true, nil
		}
	}
	return false, nil
}

func (bdb *BindingSQLDB) GetCustomMetricStrategyByAppId(ctx context.Context, appId string) (string, error) {
	customMetricsStrategy, err := bdb.fetchCustomMetricStrategyByAppId(ctx, appId)
	if err != nil {
		return "", err
	}
	return customMetricsStrategy, nil
}

func (bdb *BindingSQLDB) SetOrUpdateCustomMetricStrategy(ctx context.Context, appId string, customMetricsStrategy string, actionName string) error {
	appBinding, err := bdb.getServiceBindingByAppId(ctx, appId)
	if err != nil {
		return err
	}
	query := bdb.sqldb.Rebind("UPDATE binding SET custom_metrics_strategy = ? WHERE binding_id = ?")
	result, err := bdb.sqldb.ExecContext(ctx, query, nullableString(customMetricsStrategy), appBinding.ServiceBindingID)
	if err != nil {
		bdb.logger.Error(fmt.Sprintf("failed to %s custom metric submission strategy", actionName), err,
			lager.Data{"query": query, "customMetricsStrategy": customMetricsStrategy, "bindingId": appBinding.ServiceInstanceID, "appId": appId})
		return err
	}
	if rowsAffected, err := result.RowsAffected(); err != nil || rowsAffected == 0 {
		if customMetricsStrategy == appBinding.CustomMetricsStrategy {
			bdb.logger.Info("custom metrics strategy already exists", lager.Data{"query": query, "customMetricsStrategy": customMetricsStrategy, "bindingId": appBinding, "appId": appId})
			return nil
		}
		bdb.logger.Error(fmt.Sprintf("failed to %s custom metric submission strategy", actionName), err,
			lager.Data{"query": query, "customMetricsStrategy": customMetricsStrategy, "bindingId": appBinding, "appId": appId})
		return errors.New("no rows affected")
	}
	return nil
}

func (bdb *BindingSQLDB) fetchCustomMetricStrategyByAppId(ctx context.Context, appId string) (string, error) {
	var customMetricsStrategy sql.NullString
	query := bdb.sqldb.Rebind("SELECT custom_metrics_strategy FROM binding WHERE app_id =?")
	rows, err := bdb.sqldb.QueryContext(ctx, query, appId)

	if err != nil {
		bdb.logger.Error("get-custom-metrics-strategy-by-appid", err, lager.Data{"query": query, "appId": appId})
		return "", err
	}
	defer func() { _ = rows.Close() }()

	if rows.Next() {
		if err = rows.Scan(&customMetricsStrategy); err != nil {
			bdb.logger.Error("error-finding-customMetricsStrategy-in-binding-table", err)
			return "", err
		}
	}
	err = rows.Err()
	if err != nil {
		bdb.logger.Error("error-finding-customMetricsStrategy-in-binding-table", err)
		return "", err
	}
	return customMetricsStrategy.String, nil
}
