package sync

import (
	"autoscaler/db"
	"autoscaler/models"
	"os"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/tedsuo/ifrit"
)

type DatabaseLock struct {
	logger lager.Logger
}

func NewDatabaseLock(logger lager.Logger) *DatabaseLock {
	return &DatabaseLock{
		logger: logger,
	}
}

func (dblock *DatabaseLock) InitDBLockRunner(retryInterval time.Duration, ttl time.Duration, owner string, lockDB db.LockDB) ifrit.Runner {
	dbLockMaintainer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		lockTicker := time.NewTicker(retryInterval)
		readyToAcquireLock := true
		if owner == "" {
			dblock.logger.Info("failed-to-get-owner-details")
			os.Exit(1)
		}
		lock := &models.Lock{Owner: owner, Ttl: ttl}
		isLockAcquired, lockErr := lockDB.Lock(lock)
		if lockErr != nil {
			dblock.logger.Error("failed-to-acquire-lock-in-first-attempt", lockErr)
		}
		if isLockAcquired {
			readyToAcquireLock = false
			dblock.logger.Info("lock-acquired-in-first-attempt", lager.Data{"owner": owner, "isLockAcquired": isLockAcquired})
			close(ready)
		}
		for {
			select {
			case <-signals:
				dblock.logger.Info("received-interrupt-signal", lager.Data{"owner": owner})
				lockTicker.Stop()
				releaseErr := lockDB.Release(owner)
				if releaseErr != nil {
					dblock.logger.Error("failed-to-release-lock ", releaseErr)
				} else {
					dblock.logger.Debug("successfully-released-lock", lager.Data{"owner": owner})
				}
				readyToAcquireLock = true
				return nil

			case <-lockTicker.C:
				dblock.logger.Debug("retry-acquiring-lock", lager.Data{"owner": owner})
				lock := &models.Lock{Owner: owner, Ttl: ttl}
				isLockAcquired, lockErr := lockDB.Lock(lock)
				if lockErr != nil {
					dblock.logger.Error("failed-to-acquire-lock", lockErr)
					releaseErr := lockDB.Release(owner)
					if releaseErr != nil {
						dblock.logger.Error("failed-to-release-lock ", releaseErr)
					} else {
						dblock.logger.Info("successfully-released-lock", lager.Data{"owner": owner})
					}
					os.Exit(1)
				}
				if isLockAcquired && readyToAcquireLock {
					readyToAcquireLock = false
					dblock.logger.Info("successfully-acquired-lock", lager.Data{"owner": owner})
					close(ready)
				}
			}
		}
	})
	return dbLockMaintainer
}
