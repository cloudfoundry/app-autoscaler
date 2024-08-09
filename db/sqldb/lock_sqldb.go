package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"time"

	"code.cloudfoundry.org/lager/v3"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"github.com/uptrace/opentelemetry-go-extra/otelsqlx"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type LockSQLDB struct {
	dbConfig db.DatabaseConfig
	logger   lager.Logger
	table    string
	sqldb    *sqlx.DB
}

func NewLockSQLDB(dbConfig db.DatabaseConfig, table string, logger lager.Logger) (*LockSQLDB, error) {
	database, err := db.GetConnection(dbConfig.URL)
	if err != nil {
		return nil, err
	}

	sqldb, err := otelsqlx.Open(database.DriverName, database.DataSourceName, otelsql.WithAttributes(database.OTELAttribute))
	if err != nil {
		logger.Error("open-lock-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		_ = sqldb.Close()
		logger.Error("ping-lock-db", err, lager.Data{"dbConfig": dbConfig})
		return nil, err
	}

	sqldb.SetConnMaxLifetime(dbConfig.ConnectionMaxLifetime)
	sqldb.SetMaxIdleConns(int(dbConfig.MaxIdleConnections))
	sqldb.SetMaxOpenConns(int(dbConfig.MaxOpenConnections))
	sqldb.SetConnMaxIdleTime(dbConfig.ConnectionMaxIdleTime)

	return &LockSQLDB{
		dbConfig: dbConfig,
		logger:   logger,
		sqldb:    sqldb,
		table:    table,
	}, nil
}

func (ldb *LockSQLDB) Close() error {
	err := ldb.sqldb.Close()
	if err != nil {
		ldb.logger.Error("close-lock-db", err, lager.Data{"dbConfig": ldb.dbConfig})
		return err
	}
	return nil
}

//nolint:gosec // #nosec G202 -- string comes from safe source and parametrized table names are not supported.
func (ldb *LockSQLDB) fetch(tx *sql.Tx) (*models.Lock, error) {
	ldb.logger.Debug("fetching-lock")
	var (
		owner     string
		timestamp time.Time
		ttl       time.Duration
	)

	if ldb.sqldb.DriverName() == "pgx" {
		tquery := "LOCK TABLE " + ldb.table + " IN ACCESS EXCLUSIVE MODE"
		if _, err := tx.Exec(tquery); err != nil {
			ldb.logger.Error("failed-to-set-table-level-lock", err)
			return &models.Lock{}, err
		}
	}

	query := "SELECT owner,lock_timestamp,ttl FROM " + ldb.table + " LIMIT 1 FOR UPDATE"
	if ldb.sqldb.DriverName() == "pgx" {
		query = query + " NOWAIT "
	}
	row := tx.QueryRow(query)
	err := row.Scan(&owner, &timestamp, &ttl)
	if err != nil {
		if err == sql.ErrNoRows {
			ldb.logger.Error("no-lock-found", err)
			return nil, nil
		}
		ldb.logger.Error("failed-to-fetch-lock-details", err)
		return &models.Lock{}, err
	}
	fetchedLock := &models.Lock{Owner: owner, LastModifiedTimestamp: timestamp, Ttl: ttl}
	return fetchedLock, nil
}

func (ldb *LockSQLDB) remove(owner string, tx *sql.Tx) error {
	ldb.logger.Debug("removing-lock", lager.Data{"Owner": owner})
	query := ldb.sqldb.Rebind("DELETE FROM " + ldb.table + " WHERE owner = ?")
	if _, err := tx.Exec(query, owner); err != nil {
		ldb.logger.Error("failed-to-delete-lock-details-during-release-lock", err)
		return err
	}
	return nil
}

func (ldb *LockSQLDB) insert(lockDetails *models.Lock, tx *sql.Tx) error {
	ldb.logger.Info("inserting-the-lock-details", lager.Data{"Owner": lockDetails.Owner, "LastModifiedTimestamp": lockDetails.LastModifiedTimestamp, "Ttl": lockDetails.Ttl})
	currentTimestamp, err := ldb.getDatabaseTimestamp(tx)
	if err != nil {
		ldb.logger.Error("error-getting-timestamp-while-inserting-lock-details", err)
		return err
	}
	query := ldb.sqldb.Rebind("INSERT INTO " + ldb.table + " (owner,lock_timestamp,ttl) VALUES (?,?,?)")
	if _, err = tx.Exec(query, lockDetails.Owner, currentTimestamp, int64(lockDetails.Ttl/time.Second)); err != nil {
		ldb.logger.Error("failed-to-insert-lock-details", err)
		return err
	}
	return err
}

func (ldb *LockSQLDB) renew(owner string, tx *sql.Tx) error {
	ldb.logger.Debug("renewing-lock", lager.Data{"Owner": owner})
	currentTimestamp, err := ldb.getDatabaseTimestamp(tx)
	if err != nil {
		ldb.logger.Error("error-getting-timestamp-while-renewing-lock", err)
		return err
	}
	updatequery := ldb.sqldb.Rebind("UPDATE " + ldb.table + " SET lock_timestamp=? where owner=?")
	if _, err = tx.Exec(updatequery, currentTimestamp, owner); err != nil {
		ldb.logger.Error("failed-to-update-lock-details-during-lock-renewal", err)
		return err
	}
	return err
}

func (ldb *LockSQLDB) Release(owner string) error {
	ldb.logger.Debug("releasing-lock", lager.Data{"Owner": owner})
	err := ldb.transact(ldb.sqldb, func(tx *sql.Tx) error {
		query := ldb.sqldb.Rebind("DELETE FROM " + ldb.table + " WHERE owner = ?")
		if _, err := tx.Exec(query, owner); err != nil {
			ldb.logger.Error("failed-to-delete-lock-details-during-release-lock", err)
			return err
		}
		return nil
	})
	return err
}

func (ldb *LockSQLDB) Lock(lock *models.Lock) (bool, error) {
	ldb.logger.Debug("acquiring-lock", lager.Data{"Owner": lock.Owner})
	isLockAcquired := true
	err := ldb.transact(ldb.sqldb, func(tx *sql.Tx) error {
		newLock := false
		fetchedLock, err := ldb.fetch(tx)
		if err == nil && fetchedLock == nil {
			ldb.logger.Debug("no-one-holds-the-lock")
			newLock = true
		} else if err != nil {
			ldb.logger.Error("failed-to-fetch-lock", err)
			isLockAcquired = false
			return err
		} else if fetchedLock.Owner != lock.Owner && fetchedLock.Owner != "" {
			ldb.logger.Debug("someone-else-owns-lock", lager.Data{"Owner": fetchedLock.Owner})
			lastUpdatedTimestamp := fetchedLock.LastModifiedTimestamp
			currentTimestamp, err := ldb.getDatabaseTimestamp(tx)
			if err != nil {
				ldb.logger.Error("error-getting-timestamp-while-fetching-lock-details", err)
				isLockAcquired = false
				return err
			}
			if lastUpdatedTimestamp.Add(time.Second * fetchedLock.Ttl).Before(currentTimestamp) {
				ldb.logger.Info("lock-expired", lager.Data{"Owner": fetchedLock.Owner})
				err = ldb.remove(fetchedLock.Owner, tx)
				if err != nil {
					ldb.logger.Error("failed-to-release-existing-lock", err)
					isLockAcquired = false
					return err
				}
				newLock = true
			} else {
				ldb.logger.Debug("lock-still-valid", lager.Data{"Owner": fetchedLock.Owner})
				isLockAcquired = false
				return nil
			}
		}

		if newLock {
			err = ldb.insert(lock, tx)
			if err != nil {
				ldb.logger.Error("failed-to-insert-lock", err)
				isLockAcquired = false
				return err
			}
		} else {
			err = ldb.renew(lock.Owner, tx)
			if err != nil {
				ldb.logger.Error("failed-to-renew-lock", err)
				isLockAcquired = false
				return err
			}
		}

		if newLock {
			ldb.logger.Info("acquired-lock-successfully")
		} else {
			ldb.logger.Debug("renewed-lock-successfully")
		}
		return nil
	})

	return isLockAcquired, err
}

func (ldb *LockSQLDB) getDatabaseTimestamp(tx *sql.Tx) (time.Time, error) {
	var currentTimestamp time.Time
	var query string
	switch ldb.sqldb.DriverName() {
	case "pgx":
		query = "SELECT NOW() AT TIME ZONE 'utc'"
	case "mysql":
		query = "SELECT UTC_TIMESTAMP()"
	default:
		return time.Time{}, nil
	}

	err := tx.QueryRow(query).Scan(&currentTimestamp)
	if err != nil {
		ldb.logger.Error("failed-fetching-timestamp", err)
		return time.Time{}, err
	}
	return currentTimestamp, nil
}

func (ldb *LockSQLDB) transact(db *sqlx.DB, f func(tx *sql.Tx) error) error {
	var err error
	for attempts := 0; attempts < 3; attempts++ {
		err = func() error {
			tx, err := db.Begin()
			if err != nil {
				ldb.logger.Error("failed-starting-transaction", err)
				return err
			}
			defer func() {
				_ = tx.Rollback()
			}()

			err = f(tx)
			if err != nil {
				return err
			}

			err = tx.Commit()
			if err != nil {
				ldb.logger.Error("failed-committing-transaction", err)
			}
			return err
		}()

		// golang sql package does not always retry query on ErrBadConn
		if attempts >= 2 || !errors.Is(err, driver.ErrBadConn) {
			break
		} else {
			ldb.logger.Debug("wait-before-retry-for-transaction", lager.Data{"attempts": attempts})
			time.Sleep(500 * time.Millisecond)
		}
	}

	return err
}
