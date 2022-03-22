package sqldb

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager"
)

type StoredProcedureSQLDb struct {
	config   models.StoredProcedureConfig
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sql.DB
}

func NewStoredProcedureSQLDb(config models.StoredProcedureConfig, dbConfig db.DatabaseConfig, logger lager.Logger) (*StoredProcedureSQLDb, error) {
	sqldb, err := sql.Open(db.PostgresDriverName, dbConfig.URL)
	if err != nil {
		logger.Error("open-stored-procedure-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("ping-stored-procedure-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(dbConfig.MaxIdleConnections)
	sqldb.SetMaxOpenConns(dbConfig.MaxOpenConnections)
	sqldb.SetConnMaxIdleTime(dbConfig.ConnectionMaxIdleTime)

	return &StoredProcedureSQLDb{
		config:   config,
		dbConfig: dbConfig,
		logger:   logger,
		sqldb:    sqldb,
	}, nil
}

func (sdb *StoredProcedureSQLDb) Close() error {
	err := sdb.sqldb.Close()
	if err != nil {
		sdb.logger.Error("close-stored-procedure-db", err, lager.Data{"dbConfig": sdb.dbConfig})
		return err
	}
	return nil
}

func (sdb *StoredProcedureSQLDb) CreateCredentials(credOptions models.CredentialsOptions) (*models.Credential, error) {
	credentials := &models.Credential{}
	query := fmt.Sprintf("SELECT * from %s.%s($1,$2)", pq.QuoteIdentifier(sdb.config.SchemaName), pq.QuoteIdentifier(sdb.config.CreateBindingCredentialProcedureName))
	sdb.logger.Info(query)
	err := sdb.sqldb.QueryRow(query, credOptions.InstanceId, credOptions.BindingId).Scan(&credentials.Username, &credentials.Password)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		sdb.logger.Error("create-stored-procedure-credentials", err, lager.Data{"query": query, "credOptions": credOptions})
		return nil, err
	}
	return credentials, nil
}

func (sdb *StoredProcedureSQLDb) DeleteCredentials(credOptions models.CredentialsOptions) error {
	var count int
	query := fmt.Sprintf("SELECT * from %s.%s($1,$2)", pq.QuoteIdentifier(sdb.config.SchemaName), pq.QuoteIdentifier(sdb.config.DropBindingCredentialProcedureName))
	err := sdb.sqldb.QueryRow(query, credOptions.InstanceId, credOptions.BindingId).Scan(&count)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		sdb.logger.Error("delete-stored-procedure-credentials", err, lager.Data{"query": query, "credOptions": credOptions})
		return err
	}
	return nil
}

func (sdb *StoredProcedureSQLDb) DeleteAllInstanceCredentials(instanceId string) error {
	var count int
	query := fmt.Sprintf("SELECT * from %s.%s($1)", pq.QuoteIdentifier(sdb.config.SchemaName), pq.QuoteIdentifier(sdb.config.DropAllBindingCredentialProcedureName))
	err := sdb.sqldb.QueryRow(query, instanceId).Scan(&count)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		sdb.logger.Error("delete-all-instance-stored-procedure-credentials", err, lager.Data{"query": query, "instanceId": instanceId})
		return err
	}
	return nil
}

func (sdb *StoredProcedureSQLDb) ValidateCredentials(creds models.Credential) (*models.CredentialsOptions, error) {
	credOptions := &models.CredentialsOptions{}
	query := fmt.Sprintf("SELECT * from %s.%s($1,$2)", pq.QuoteIdentifier(sdb.config.SchemaName), pq.QuoteIdentifier(sdb.config.ValidateBindingCredentialProcedureName))
	err := sdb.sqldb.QueryRow(query, creds.Username, creds.Password).Scan(&credOptions.InstanceId, &credOptions.BindingId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		sdb.logger.Error("validate-stored-procedure-credentials", err, lager.Data{"query": query, "creds": creds})
		return nil, err
	}
	return credOptions, nil
}
