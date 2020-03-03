package sqldb

import (
	"database/sql"
	"time"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"autoscaler/db"
)

type BindingSQLDB struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sqlx.DB
}

func NewBindingSQLDB(dbConfig db.DatabaseConfig, logger lager.Logger) (*BindingSQLDB, error) {
	database, err := db.GetConnection(dbConfig.URL)
	if err != nil {
		return nil, err
	}

	sqldb, err := sqlx.Open(database.DriverName, database.DSN)
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

func (bdb *BindingSQLDB) CreateServiceInstance(serviceInstanceId string, orgId string, spaceId string) error {
	query := bdb.sqldb.Rebind("SELECT org_id, space_id FROM service_instance WHERE service_instance_id =?")
	rows, err := bdb.sqldb.Query(query, serviceInstanceId)
	if err != nil {
		bdb.logger.Error("create-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId, "orgid": orgId, "spaceid": spaceId})
		return err
	}

	if rows.Next() {
		var (
			existingOrgId   string
			existingSpaceId string
		)
		if err := rows.Scan(&existingOrgId, &existingSpaceId); err != nil {
			bdb.logger.Error("create-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId, "orgid": orgId, "spaceid": spaceId})
		}
		rows.Close()
		if existingOrgId == orgId && existingSpaceId == spaceId {
			return db.ErrAlreadyExists
		} else {
			return db.ErrConflict
		}

	}
	rows.Close()

	query = bdb.sqldb.Rebind("INSERT INTO service_instance" +
		"(service_instance_id, org_id, space_id) " +
		" VALUES(?, ?, ?)")
	_, err = bdb.sqldb.Exec(query, serviceInstanceId, orgId, spaceId)

	if err != nil {
		bdb.logger.Error("create-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId, "orgid": orgId, "spaceid": spaceId})
	}
	return err
}

func (bdb *BindingSQLDB) DeleteServiceInstance(serviceInstanceId string) error {
	query := bdb.sqldb.Rebind("SELECT * FROM service_instance WHERE service_instance_id =?")
	rows, err := bdb.sqldb.Query(query, serviceInstanceId)
	if err != nil {
		bdb.logger.Error("delete-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId})
		return err
	}

	if rows.Next() {
		rows.Close()
		query = bdb.sqldb.Rebind("DELETE FROM service_instance WHERE service_instance_id =?")
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
	query := bdb.sqldb.Rebind("SELECT * FROM binding WHERE app_id =?")
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

	query = bdb.sqldb.Rebind("INSERT INTO binding" +
		"(binding_id, service_instance_id, app_id, created_at) " +
		"VALUES(?, ?, ?, ?)")
	_, err = bdb.sqldb.Exec(query, bindingId, serviceInstanceId, appId, time.Now())

	if err != nil {
		bdb.logger.Error("create-service-binding", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId, "bindingid": bindingId, "appid": appId})
	}
	return err
}
func (bdb *BindingSQLDB) DeleteServiceBinding(bindingId string) error {
	query := bdb.sqldb.Rebind("SELECT * FROM binding WHERE binding_id =?")
	rows, err := bdb.sqldb.Query(query, bindingId)
	if err != nil {
		bdb.logger.Error("delete-service-binding", err, lager.Data{"query": query, "bindingId": bindingId})
		return err
	}

	if rows.Next() {
		rows.Close()
		query = bdb.sqldb.Rebind("DELETE FROM binding WHERE binding_id =?")
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
	query := bdb.sqldb.Rebind("DELETE FROM binding WHERE app_id =?")
	_, err := bdb.sqldb.Exec(query, appId)

	if err != nil {
		bdb.logger.Error("delete-service-binding-by-appid", err, lager.Data{"query": query, "appId": appId})
		return err
	}
	return nil
}
func (bdb *BindingSQLDB) CheckServiceBinding(appId string) bool {
	var count int
	query := bdb.sqldb.Rebind("SELECT COUNT(*) FROM binding WHERE app_id=?")
	bdb.sqldb.QueryRow(query, appId).Scan(&count)
	return count > 0
}
func (bdb *BindingSQLDB) GetDBStatus() sql.DBStats {
	return bdb.sqldb.Stats()
}
func (bdb *BindingSQLDB) GetAppIdByBindingId(bindingId string) (string, error) {
	var appId string
	query := bdb.sqldb.Rebind("SELECT app_id FROM binding WHERE binding_id=?")
	err := bdb.sqldb.QueryRow(query, bindingId).Scan(&appId)
	if err != nil {
		bdb.logger.Error("get-appid-from-binding-table", err, lager.Data{"query": query, "bindingId": bindingId})
		return "", err
	}
	return appId, nil
}
