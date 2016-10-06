package schedule

import (
	"autoscaler/cf"
	"autoscaler/db"

	"code.cloudfoundry.org/lager"
	"sync"
)

type ActiveSchedule struct {
	ScheduleId      string
	InstanceMin     int `json:"instance_min_count"`
	InstanceMax     int `json:"instance_max_count"`
	InstanceInitial int `json:"initial_min_instance_count"`
}

type AppSchedules struct {
	schedules map[string]ActiveSchedule
	lock      *sync.Mutex
	cfClient  cf.CfClient
	logger    lager.Logger
	policyDB  db.PolicyDB
}

func NewAppSchedules(logger lager.Logger, cfClient cf.CfClient, policyDB db.PolicyDB) *AppSchedules {
	return &AppSchedules{
		schedules: make(map[string]ActiveSchedule),
		lock:      &sync.Mutex{},
		cfClient:  cfClient,
		policyDB:  policyDB,
		logger:    logger,
	}
}

func (as *AppSchedules) GetActiveSchedule(appId string) (ActiveSchedule, bool) {
	as.lock.Lock()
	schedule, exists := as.schedules[appId]
	as.lock.Unlock()
	return schedule, exists
}

func (as *AppSchedules) SetActiveSchedule(appId string, schedule ActiveSchedule) error {
	as.lock.Lock()
	currentSchedule, exists := as.schedules[appId]
	as.schedules[appId] = schedule
	as.lock.Unlock()

	logger := as.logger.WithData(lager.Data{"appId": appId})

	if exists {
		logger.Info("set-active-schedule", lager.Data{"message": "active schedule exists", "currentSchedule": currentSchedule})
	}

	err := as.cfClient.SetAppInstances(appId, schedule.InstanceInitial)
	if err != nil {
		logger.Error("failed-set-active-schedule-set-instance-initial", err, lager.Data{"schedule": schedule})
		return err
	}
	return nil
}

func (as *AppSchedules) RemoveActiveSchedule(appId string, scheduleId string) error {
	as.lock.Lock()
	schedule, exists := as.schedules[appId]
	delete(as.schedules, appId)
	as.lock.Unlock()

	logger := as.logger.WithData(lager.Data{"appId": appId, "scheduleId": scheduleId})

	if !exists {
		logger.Info("remove-active-schedule", lager.Data{"message": "active schedule does not exist"})
	} else if schedule.ScheduleId != scheduleId {
		logger.Info("remove-active-schedule", lager.Data{"message": "schedule id does not match", "currrentScheduleId": schedule.ScheduleId})
	}

	policy, err := as.policyDB.GetAppPolicy(appId)
	if err != nil {
		logger.Error("failed-remove-active-schedule-get-app-policy", err)
		return err
	}

	instances, err := as.cfClient.GetAppInstances(appId)
	if err != nil {
		logger.Error("failed-remove-active-schedule-get-app-instances", err)
		return err
	}

	if (instances < policy.InstanceMin) || (instances > policy.InstanceMax) {
		if instances < policy.InstanceMin {
			instances = policy.InstanceMin
		} else {
			instances = policy.InstanceMax
		}

		err = as.cfClient.SetAppInstances(appId, instances)
		if err != nil {
			logger.Error("failed-remove-active-schedule-set-app-instances", err)
			return err
		}
	}

	return nil
}
