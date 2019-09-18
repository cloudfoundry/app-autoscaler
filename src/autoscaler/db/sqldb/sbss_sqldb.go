package sqldb

import (
	"database/sql"

	"autoscaler/db"
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"
)

type SbssSQLDb struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	sqldb    *sql.DB
}

func NewSbssSQLDb(dbConfig db.DatabaseConfig, logger lager.Logger) (*SbssSQLDb, error) {
	sqldb, err := sql.Open(db.PostgresDriverName, dbConfig.URL)
	if err != nil {
		logger.Error("open-sbss-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("ping-sbss-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(dbConfig.MaxIdleConnections)
	sqldb.SetMaxOpenConns(dbConfig.MaxOpenConnections)

	return &SbssSQLDb{
		dbConfig: dbConfig,
		logger:   logger,
		sqldb:    sqldb,
	}, nil
}

func (sdb *SbssSQLDb) Close() error {
	err := sdb.sqldb.Close()
	if err != nil {
		sdb.logger.Error("Close-sbss-db", err, lager.Data{"dbConfig": sdb.dbConfig})
		return err
	}
	return nil
}

func (sdb *SbssSQLDb) CreateCredentials(credOptions models.CredentialsOptions) (*models.Credential, error) {
	credentials := &models.Credential{}
	query := "SELECT * from SYS_XS_SBSS.CREATE_BINDING_CREDENTIAL($1,$2)"
	err := sdb.sqldb.QueryRow(query, credOptions.InstanceId, credOptions.BindingId).Scan(&credentials.Username, &credentials.Password)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		sdb.logger.Error("create-sbss-credentials", err, lager.Data{"query": query, "credOptions": credOptions})
		return nil, err
	}
	return credentials, nil
}

func (sdb *SbssSQLDb) DeleteCredentials(credOptions models.CredentialsOptions) error {
	var count int
	query := "SELECT * from SYS_XS_SBSS.DROP_BINDING_CREDENTIAL($1,$2)"
	err := sdb.sqldb.QueryRow(query, credOptions.InstanceId, credOptions.BindingId).Scan(&count)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		sdb.logger.Error("delete-sbss-credentials", err, lager.Data{"query": query, "credOptions": credOptions})
		return err
	}
	return nil
}

func (sdb *SbssSQLDb) DeleteAllInstanceCredentials(instanceId string) error {
	var count int
	query := "SELECT * from SYS_XS_SBSS.DROP_ALL_BINDING_CREDENTIALS($1)"
	err := sdb.sqldb.QueryRow(query, instanceId).Scan(&count)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		sdb.logger.Error("delete-all-instance-sbss-credentials", err, lager.Data{"query": query, "instanceId": instanceId})
		return err
	}
	return nil
}

func (sdb *SbssSQLDb) ValidateCredentials(creds models.Credential) (*models.CredentialsOptions, error) {
	credOptions := &models.CredentialsOptions{}
	query := "SELECT * from SYS_XS_SBSS.VALIDATE_BINDING_CREDENTIAL($1,$2)"
	err := sdb.sqldb.QueryRow(query, creds.Username, creds.Password).Scan(&credOptions.InstanceId, &credOptions.BindingId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		sdb.logger.Error("validate-sbss-credentials", err, lager.Data{"query": query, "creds": creds})
		return nil, err
	}
	return credOptions, nil
}
