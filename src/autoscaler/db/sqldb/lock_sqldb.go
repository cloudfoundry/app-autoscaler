package sqldb

import (
	"database/sql"
	"time"

	"code.cloudfoundry.org/lager"
	_ "github.com/lib/pq"

	"autoscaler/db"
	"autoscaler/models"
)

type LockSQLDB struct {
	url    string
	logger lager.Logger
	sqldb  *sql.DB
}

func NewLockSQLDB(url string, logger lager.Logger) (*LockSQLDB, error) {
	sqldb, err := sql.Open(db.PostgresDriverName, url)
	if err != nil {
		logger.Error("open-lock-db", err, lager.Data{"url": url})
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		sqldb.Close()
		logger.Error("ping-lock-db", err, lager.Data{"url": url})
		return nil, err
	}

	return &LockSQLDB{
		url:    url,
		logger: logger,
		sqldb:  sqldb,
	}, nil
}

func (ldb *LockSQLDB) Close() error {
	err := ldb.sqldb.Close()
	if err != nil {
		ldb.logger.Error("close-lock-db", err, lager.Data{"url": ldb.url})
		return err
	}
	return nil
}

func (ldb *LockSQLDB) Fetch() (lock *models.Lock, err error) {
	ldb.logger.Info("fetching locks ")
	var (
		owner     string
		timestamp time.Time
		ttl       time.Duration
	)
	tx, err := ldb.sqldb.Begin()
	if err != nil {
		ldb.logger.Error("error-fetching-lock-transaction", err)
		return nil, err
	}
	query := "SELECT * FROM locks"
	err = tx.QueryRow(query).Scan(&owner, &timestamp, &ttl)
	if err != nil {
		return &models.Lock{}, err
	}
	err = tx.Commit()
	if err != nil {
		ldb.logger.Error("failed-to-commit-fetching-lock-transaction", err)
		return &models.Lock{}, err
	}
	fetchedLock := &models.Lock{Owner: owner, LastModifiedTimestamp: timestamp, Ttl: ttl}
	return fetchedLock, nil
}

func (ldb *LockSQLDB) Acquire(lockDetails *models.Lock) error {
	ldb.logger.Info("acquiring-the-lock", lager.Data{"Owner": lockDetails.Owner, "LastModifiedTimestamp": lockDetails.LastModifiedTimestamp, "Ttl": lockDetails.Ttl})
	tx, err := ldb.sqldb.Begin()
	if err != nil {
		ldb.logger.Error("error-starting-acquire-lock-transaction", err)
		return err
	}
	defer func() {
		if err != nil {
			ldb.logger.Error("rolling-back-acquire-lock-transaction!", err)
			err = tx.Rollback()
			if err != nil {
				ldb.logger.Error("failed-to-rollback-acquire-lock-transaction", err)
			}
			return
		}
		err = tx.Commit()
		if err != nil {
			ldb.logger.Error("failed-to-commit-acquire-lock-transaction", err)
		}
	}()
	if _, err = tx.Exec("SELECT * FROM locks FOR UPDATE NOWAIT"); err != nil {
		ldb.logger.Error("failed-to-select-for-update", err)
		return err
	}
	currentTimestamp, err := ldb.GetDatabaseTimestamp()
	if err != nil {
		ldb.logger.Error("error-getting-timestamp-while-acquiring-lock", err)
		return err
	}
	query := "INSERT INTO locks (owner,lock_timestamp,ttl) VALUES ($1,$2,$3)"
	if _, err = tx.Exec(query, lockDetails.Owner, currentTimestamp, int64(lockDetails.Ttl/time.Second)); err != nil {
		ldb.logger.Error("failed-to-insert-lock-details-during-acquire-lock", err)
		return err
	}
	return err
}

func (ldb *LockSQLDB) Renew(owner string) error {
	ldb.logger.Debug("renewing-lock", lager.Data{"Owner": owner})
	tx, err := ldb.sqldb.Begin()
	if err != nil {
		ldb.logger.Error("error-starting-renew-lock-transaction", err)
		return err
	}
	defer func() {
		if err != nil {
			ldb.logger.Error("rolling-back-renew-lock-transaction", err)
			err = tx.Rollback()
			if err != nil {
				ldb.logger.Error("failed-to-rollback-renew-lock-transaction", err)
			}
			return
		}
		err = tx.Commit()
		if err != nil {
			ldb.logger.Error("failed-to-commit-renew-lock-transaction", err)
		}
	}()
	query := "SELECT * FROM locks where owner=$1 FOR UPDATE NOWAIT"
	if _, err = tx.Exec(query, owner); err != nil {
		ldb.logger.Error("failed-to-select-for-update", err)
		return err
	}
	currentTimestamp, err := ldb.GetDatabaseTimestamp()
	if err != nil {
		ldb.logger.Error("error-getting-timestamp-while-renewing-lock", err)
		return err
	}
	updatequery := "UPDATE locks SET lock_timestamp=$1 where owner=$2"
	if _, err = tx.Exec(updatequery, currentTimestamp, owner); err != nil {
		ldb.logger.Error("failed-to-update-lock-details-during-lock-renewal", err)
		return err
	}
	return err
}

func (ldb *LockSQLDB) Release(owner string) error {
	ldb.logger.Debug("releasing-lock", lager.Data{"Owner": owner})
	tx, err := ldb.sqldb.Begin()
	if err != nil {
		ldb.logger.Error("error-starting-release-lock-transaction", err)
		return err
	}
	defer func() {
		if err != nil {
			ldb.logger.Error("rolling-back-release-lock-transaction!", err)
			err = tx.Rollback()
			if err != nil {
				ldb.logger.Error("failed-to-rollback-release-lock-transaction", err)
			}
			return
		}
		err = tx.Commit()
		if err != nil {
			ldb.logger.Error("failed-to-commit-release-lock-transaction", err)
		}
	}()
	if _, err := tx.Exec("SELECT * FROM locks FOR UPDATE NOWAIT"); err != nil {
		ldb.logger.Error("failed-to-select-for-update", err)
		return err
	}
	query := "DELETE FROM locks WHERE owner = $1"
	if _, err := tx.Exec(query, owner); err != nil {
		ldb.logger.Error("failed-to-delete-lock-details-during-release-lock", err)
		return err
	}
	return nil
}

func (ldb *LockSQLDB) Lock(lock *models.Lock) (bool, error) {
	fetchedLock, err := ldb.Fetch()
	if err != nil && err == sql.ErrNoRows {
		ldb.logger.Info("no-one-holds-the-lock")
		err = ldb.Acquire(lock)
		if err != nil {
			ldb.logger.Error("failed-to-acquire-the-lock", err)
			return false, err
		}
	} else if err != nil && err != sql.ErrNoRows {
		ldb.logger.Error("failed-to-fetch-lock", err)
		return false, err
	} else {
		if fetchedLock.Owner == lock.Owner {
			err = ldb.Renew(lock.Owner)
			if err != nil {
				ldb.logger.Error("failed-to-renew-lock", err)
				return false, err
			}
		} else {
			ldb.logger.Debug("someone-else-owns-lock", lager.Data{"Owner": fetchedLock.Owner})
			lastUpdatedTimestamp := fetchedLock.LastModifiedTimestamp
			currentTimestamp, err := ldb.GetDatabaseTimestamp()
			if err != nil {
				ldb.logger.Error("error-getting-timestamp-while-getting-lock", err)
				return false, err
			}
			if lastUpdatedTimestamp.Add(time.Second * time.Duration(fetchedLock.Ttl)).Before(currentTimestamp) {
				ldb.logger.Info("lock-not-renewed-by-owner-forcefully-acquiring-the-lock", lager.Data{"Owner": fetchedLock.Owner})
				err = ldb.Release(fetchedLock.Owner)
				if err != nil {
					ldb.logger.Error("failed-to-release-existing-lock", err)
					return false, err
				}
				err = ldb.Acquire(lock)
				if err != nil {
					ldb.logger.Error("failed-to-acquire-lock", err)
					return false, err
				}
			} else {
				ldb.logger.Debug("lock-renewed-by-owner", lager.Data{"Owner": fetchedLock.Owner})
				return false, nil
			}
		}
	}
	return true, nil
}

func (ldb *LockSQLDB) GetDatabaseTimestamp() (time.Time, error) {
	var currentTimestamp time.Time
	err := ldb.sqldb.QueryRow("SELECT NOW() AT TIME ZONE 'utc'").Scan(&currentTimestamp)
	if err != nil {
		return time.Time{}, err
	}
	return currentTimestamp, nil
}
