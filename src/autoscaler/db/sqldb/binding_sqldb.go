package sqldb

import (
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

func (bdb *BindingSQLDB) CreateServiceInstance(serviceInstanceId string, orgId string, spaceId string) error {
	query := "SELECT FROM service_instance WHERE service_instance_id = $1"
	rows, err := bdb.sqldb.Query(query, serviceInstanceId)
	if err != nil {
		bdb.logger.Error("create-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId, "orgid": orgId, "spaceid": spaceId})
		return err
	}

	if rows.Next() {
		rows.Close()
		return db.ErrAlreadyExists
	}
	rows.Close()

	query = "INSERT INTO service_instance" +
		"(service_instance_id, org_id, space_id) " +
		" VALUES($1, $2, $3)"
	_, err = bdb.sqldb.Exec(query, serviceInstanceId, orgId, spaceId)

	if err != nil {
		bdb.logger.Error("create-service-instance", err, lager.Data{"query": query, "serviceinstanceid": serviceInstanceId, "orgid": orgId, "spaceid": spaceId})
	}
	return err
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

	return db.ErrDoesNotExist
}
func (bdb *BindingSQLDB) CheckServiceBinding(appId string) bool {
	var count int
	query := "SELECT COUNT(*) FROM credentials WHERE id=$1"
	bdb.sqldb.QueryRow(query, appId).Scan(&count)
	return count > 0
}
func (bdb *BindingSQLDB) GetDBStatus() sql.DBStats {
	return bdb.sqldb.Stats()
}
