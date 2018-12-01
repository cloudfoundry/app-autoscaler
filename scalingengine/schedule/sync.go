package schedule

import (
	"sync"

	"code.cloudfoundry.org/lager"

	"autoscaler/db"
	"autoscaler/models"
	"autoscaler/scalingengine"
)

type ActiveScheduleSychronizer interface {
	Sync()
}

type activeScheduleSychronizer struct {
	logger      lager.Logger
	schedulerDB db.SchedulerDB
	engineDB    db.ScalingEngineDB
	engine      scalingengine.ScalingEngine
}

func NewActiveScheduleSychronizer(logger lager.Logger, schedulerDB db.SchedulerDB, engineDB db.ScalingEngineDB, engine scalingengine.ScalingEngine) *activeScheduleSychronizer {
	return &activeScheduleSychronizer{
		logger:      logger,
		schedulerDB: schedulerDB,
		engineDB:    engineDB,
		engine:      engine,
	}
}

func (ss *activeScheduleSychronizer) Sync() {
	ss.logger.Info("synchronizing-active-schedules")

	asScheduler, err := ss.schedulerDB.GetActiveSchedules()
	if err != nil {
		ss.logger.Error("failed-synchronize-active-schedules-get-schedules-from-schedulerDB", err)
		return
	}

	asEngine, err := ss.engineDB.GetActiveSchedules()
	if err != nil {
		ss.logger.Error("failed-synchronize-active-schedules-get-schedules-from-engineDB", err)
		return
	}

	wg := &sync.WaitGroup{}

	for appId, schedule := range asScheduler {
		scheduleId := asEngine[appId]
		if scheduleId == "" || scheduleId != schedule.ScheduleId {
			ss.logger.Info("synchronize-active-schedules-find-missing-active-schedule-start", lager.Data{"appId": appId, "schedule": schedule})
			wg.Add(1)
			go func(aid string, as *models.ActiveSchedule) {
				defer wg.Done()
				ss.engine.SetActiveSchedule(aid, as)
			}(appId, schedule)
		}
	}

	for appId, scheduleId := range asEngine {
		if asScheduler[appId] == nil {
			ss.logger.Info("synchronize-active-schedules-find-missing-active-schedule-end", lager.Data{"appId": appId, "scheduleId": scheduleId})
			wg.Add(1)
			go func(aid, sid string) {
				defer wg.Done()
				ss.engine.RemoveActiveSchedule(aid, sid)
			}(appId, scheduleId)
		}
	}
	wg.Wait()
}
