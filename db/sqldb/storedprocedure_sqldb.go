package sqldb

import (
	"context"
	"database/sql"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"code.cloudfoundry.org/lager"
)

var _ db.StoredProcedureDB = &StoredProcedureSQLDb{}

type StoredProcedureSQLDb struct {
	config   models.StoredProcedureConfig
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *pgxpool.Pool
}

func (sdb *StoredProcedureSQLDb) Ping() error {
	return sdb.sqldb.Ping(context.Background())
}

func NewStoredProcedureSQLDb(config models.StoredProcedureConfig, dbConfig db.DatabaseConfig, logger lager.Logger) (*StoredProcedureSQLDb, error) {
	poolConfig, err := pgxpool.ParseConfig(dbConfig.URL)
	if err != nil {
		logger.Error("parse-procedure-db-url", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	poolConfig.MaxConnLifetime = dbConfig.ConnectionMaxLifetime
	poolConfig.MaxConns = dbConfig.MaxOpenConnections
	poolConfig.MaxConnIdleTime = dbConfig.ConnectionMaxIdleTime

	sqldb, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Error("open-stored-procedure-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping(context.Background())
	if err != nil {
		sqldb.Close()
		logger.Error("ping-stored-procedure-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	return &StoredProcedureSQLDb{
		config:   config,
		dbConfig: dbConfig,
		logger:   logger,
		sqldb:    sqldb,
	}, nil
}

func (sdb *StoredProcedureSQLDb) Close() error {
	sdb.sqldb.Close()
	return nil
}

func (sdb *StoredProcedureSQLDb) CreateCredentials(ctx context.Context, credOptions models.CredentialsOptions) (*models.Credential, error) {
	credentials := &models.Credential{}
	procedureIdentifier := pgx.Identifier{sdb.config.SchemaName, sdb.config.CreateBindingCredentialProcedureName}
	query := fmt.Sprintf("SELECT * from %s($1,$2)", procedureIdentifier.Sanitize())
	sdb.logger.Info(query)
	err := sdb.sqldb.QueryRow(ctx, query, credOptions.InstanceId, credOptions.BindingId).Scan(&credentials.Username, &credentials.Password)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		sdb.logger.Error("create-stored-procedure-credentials", err, lager.Data{"query": query, "credOptions": credOptions})
		return nil, err
	}
	return credentials, nil
}

func (sdb *StoredProcedureSQLDb) DeleteCredentials(ctx context.Context, credOptions models.CredentialsOptions) error {
	var count int
	procedureIdentifier := pgx.Identifier{sdb.config.SchemaName, sdb.config.DropBindingCredentialProcedureName}
	query := fmt.Sprintf("SELECT * from %s($1,$2)", procedureIdentifier.Sanitize())
	err := sdb.sqldb.QueryRow(ctx, query, credOptions.InstanceId, credOptions.BindingId).Scan(&count)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		sdb.logger.Error("delete-stored-procedure-credentials", err, lager.Data{"query": query, "credOptions": credOptions})
		return err
	}
	return nil
}

func (sdb *StoredProcedureSQLDb) DeleteAllInstanceCredentials(ctx context.Context, instanceId string) error {
	var count int
	procedureIdentifier := pgx.Identifier{sdb.config.SchemaName, sdb.config.DropAllBindingCredentialProcedureName}
	query := fmt.Sprintf("SELECT * from %s($1)", procedureIdentifier.Sanitize())
	err := sdb.sqldb.QueryRow(ctx, query, instanceId).Scan(&count)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		sdb.logger.Error("delete-all-instance-stored-procedure-credentials", err, lager.Data{"query": query, "instanceId": instanceId})
		return err
	}
	return nil
}

func (sdb *StoredProcedureSQLDb) ValidateCredentials(ctx context.Context, creds models.Credential) (*models.CredentialsOptions, error) {
	credOptions := &models.CredentialsOptions{}
	procedureIdentifier := pgx.Identifier{sdb.config.SchemaName, sdb.config.ValidateBindingCredentialProcedureName}
	query := fmt.Sprintf("SELECT * from %s($1,$2)", procedureIdentifier.Sanitize())
	err := sdb.sqldb.QueryRow(ctx, query, creds.Username, creds.Password).Scan(&credOptions.InstanceId, &credOptions.BindingId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		sdb.logger.Error("validate-stored-procedure-credentials", err, lager.Data{"query": query, "creds": creds})
		return nil, err
	}
	return credOptions, nil
}
