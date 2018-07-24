package schedule

import (
	"os"
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"autoscaler/db"
	"autoscaler/models"
	"autoscaler/scalingengine"
)

type ActiveScheduleSychronizer struct {
	logger      lager.Logger
	schedulerDB db.SchedulerDB
	engineDB    db.ScalingEngineDB
	engine      scalingengine.ScalingEngine
	interval    time.Duration
	sClock      clock.Clock
}

func NewActiveScheduleSychronizer(logger lager.Logger, schedulerDB db.SchedulerDB, engineDB db.ScalingEngineDB, engine scalingengine.ScalingEngine, interval time.Duration, sClock clock.Clock) *ActiveScheduleSychronizer {
	return &ActiveScheduleSychronizer{
		logger:      logger,
		schedulerDB: schedulerDB,
		engineDB:    engineDB,
		engine:      engine,
		interval:    interval,
		sClock:      sClock,
	}
}

func (ss *ActiveScheduleSychronizer) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	ss.logger.Info("started", lager.Data{"interval": ss.interval})

	timer := ss.sClock.NewTimer(ss.interval)
	for {
		select {
		case <-signals:
			ss.logger.Info("stopped")
			return nil
		case <-timer.C():
			ss.synchronizeActiveSchedules()
			timer.Reset(ss.interval)
		}
	}
}

func (ss *ActiveScheduleSychronizer) synchronizeActiveSchedules() {
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
			ss.logger.Debug("synchronize-active-schedules-find-missing-active-schedule-start", lager.Data{"appId": appId, "schedule": schedule})
			wg.Add(1)
			go func(aid string, as *models.ActiveSchedule) {
				defer wg.Done()
				ss.engine.SetActiveSchedule(aid, as)
			}(appId, schedule)
		}
	}

	for appId, scheduleId := range asEngine {
		if asScheduler[appId] == nil {
			ss.logger.Debug("synchronize-active-schedules-find-missing-active-schedule-end", lager.Data{"appId": appId, "scheduleId": scheduleId})
			wg.Add(1)
			go func(aid, sid string) {
				defer wg.Done()
				ss.engine.RemoveActiveSchedule(aid, sid)
			}(appId, scheduleId)
		}
	}
	wg.Wait()
}
