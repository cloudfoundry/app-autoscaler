package syncer

import (
	"autoscaler/db"
	"code.cloudfoundry.org/lager"
	"errors"
)

type ActiveScheduleSyncer struct {
	policyDb    db.PolicyDB
	schedulerDb db.SchedulerDB
	logger      lager.Logger
}

func NewActiveScheduleSyncer(policyDb db.PolicyDB, schedulerDb db.SchedulerDB, logger lager.Logger) *ActiveScheduleSyncer {

	return &ActiveScheduleSyncer{
		policyDb:    policyDb,
		schedulerDb: schedulerDb,
		logger:      logger,
	}
}

func (ass *ActiveScheduleSyncer) Synchronize() error {
	ass.logger.Debug("Synchronize active schedules with policies start")
	appIdMap, err := ass.policyDb.GetAppIds()
	if err != nil {
		ass.logger.Error("Failed to get app ids from policy database", err)
		return errors.New("Failed to get app ids from policy database")
	}
	if len(appIdMap) == 0 {
		ass.logger.Debug("No application found in policy database")
		return nil
	}
	err1 := ass.schedulerDb.SynchronizeActiveSchedules(appIdMap)
	if err1 != nil {
		ass.logger.Error("Failed to synchronize active schedule with policies", err1)
		return errors.New("Failed to synchronize active schedule with policies")
	} else {
		ass.logger.Debug("Synchronize active schedules with policies successfully")
		return nil
	}

}
