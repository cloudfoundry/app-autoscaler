package sync

import (
	"autoscaler/db"
	"autoscaler/models"

	"os"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/tedsuo/ifrit"
)

const (
	OwnLock = iota
	LostLock
)

type DatabaseLock struct {
	logger lager.Logger
	// isHoldLock represents the lock status of the last acquiring.
	isHoldLock int32
}

func NewDatabaseLock(logger lager.Logger) *DatabaseLock {
	return &DatabaseLock{
		logger:     logger,
		isHoldLock: LostLock,
	}
}

func (dblock *DatabaseLock) InitDBLockRunner(retryInterval time.Duration, ttl time.Duration, owner string, lockDB db.LockDB, callbackOnAcquiredLock func(), callbackOnLostLock func()) ifrit.Runner {
	dbLockMaintainer := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		lockTicker := time.NewTicker(retryInterval)
		readyToAcquireLock := true
		if owner == "" {
			dblock.logger.Info("failed-to-get-owner-details")
			callbackOnLostLock()
		}
		lock := &models.Lock{Owner: owner, Ttl: ttl}
		isLockAcquired, lockErr := lockDB.Lock(lock)
		if lockErr != nil {
			dblock.logger.Error("failed-to-acquire-lock-in-first-attempt", lockErr)
		}
		if isLockAcquired {
			readyToAcquireLock = false
			atomic.StoreInt32(&dblock.isHoldLock, OwnLock)
			dblock.logger.Info("lock-acquired-in-first-attempt", lager.Data{"owner": owner, "isLockAcquired": isLockAcquired})
			callbackOnAcquiredLock()
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
					atomic.StoreInt32(&dblock.isHoldLock, LostLock)
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
						atomic.StoreInt32(&dblock.isHoldLock, LostLock)
						dblock.logger.Info("successfully-released-lock", lager.Data{"owner": owner})
					}
					callbackOnLostLock()
				}
				if !isLockAcquired {
					previousLockStatus := atomic.LoadInt32(&dblock.isHoldLock)
					if previousLockStatus == OwnLock {
						dblock.logger.Info("lock-has-been-acquired-by-competitor")
						atomic.StoreInt32(&dblock.isHoldLock, LostLock)
						callbackOnLostLock()
					}
				}
				if isLockAcquired && readyToAcquireLock {
					readyToAcquireLock = false
					dblock.logger.Info("successfully-acquired-lock", lager.Data{"owner": owner})
					atomic.StoreInt32(&dblock.isHoldLock, OwnLock)
					callbackOnAcquiredLock()
					close(ready)
				}
			}
		}
	})
	return dbLockMaintainer
}
