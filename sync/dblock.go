package sync

import (
	"context"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/runner"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3"
)

const (
	LockStatusHeld = iota
	LockStatusLost
)

type DatabaseLock struct {
	logger     lager.Logger
	lockStatus int32
}

func NewDatabaseLock(logger lager.Logger) *DatabaseLock {
	return &DatabaseLock{
		logger:     logger,
		lockStatus: LockStatusLost,
	}
}

func (dblock *DatabaseLock) InitDBLockRunner(retryInterval time.Duration, ttl time.Duration, owner string, lockDB db.LockDB, callbackOnAcquiredLock func(), callbackOnLostLock func()) runner.Runner {
	return runner.RunFunc(func(ctx context.Context, ready chan<- struct{}) error {
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
			dblock.lockStatus = LockStatusHeld
			dblock.logger.Info("lock-acquired-in-first-attempt", lager.Data{"owner": owner, "isLockAcquired": isLockAcquired})
			callbackOnAcquiredLock()
			close(ready)
		}
		for {
			select {
			case <-ctx.Done():
				dblock.logger.Info("received-interrupt-signal", lager.Data{"owner": owner})
				lockTicker.Stop()
				releaseErr := lockDB.Release(owner)
				if releaseErr != nil {
					dblock.logger.Error("failed-to-release-lock ", releaseErr)
				} else {
					dblock.lockStatus = LockStatusLost
					dblock.logger.Debug("successfully-released-lock", lager.Data{"owner": owner})
				}
				return nil

			case <-lockTicker.C:
				readyToAcquireLock = dblock.handleLockTick(owner, ttl, lockDB, callbackOnAcquiredLock, callbackOnLostLock, readyToAcquireLock, ready)
			}
		}
	})
}

func (dblock *DatabaseLock) handleLockTick(owner string, ttl time.Duration, lockDB db.LockDB, callbackOnAcquiredLock func(), callbackOnLostLock func(), readyToAcquireLock bool, ready chan<- struct{}) bool {
	dblock.logger.Debug("retry-acquiring-lock", lager.Data{"owner": owner})
	lock := &models.Lock{Owner: owner, Ttl: ttl}
	isLockAcquired, lockErr := lockDB.Lock(lock)
	if lockErr != nil {
		dblock.logger.Error("failed-to-acquire-lock", lockErr)
		releaseErr := lockDB.Release(owner)
		if releaseErr != nil {
			dblock.logger.Error("failed-to-release-lock ", releaseErr)
		} else {
			dblock.lockStatus = LockStatusLost
			dblock.logger.Info("successfully-released-lock", lager.Data{"owner": owner})
		}
		callbackOnLostLock()
	}
	if !isLockAcquired && dblock.lockStatus == LockStatusHeld {
		dblock.logger.Info("lock-has-been-acquired-by-competitor")
		dblock.lockStatus = LockStatusLost
		callbackOnLostLock()
	}
	if isLockAcquired && readyToAcquireLock {
		dblock.logger.Info("successfully-acquired-lock", lager.Data{"owner": owner})
		dblock.lockStatus = LockStatusHeld
		callbackOnAcquiredLock()
		close(ready)
		return false
	}
	return readyToAcquireLock
}
