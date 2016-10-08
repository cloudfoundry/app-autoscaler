package schedule

import (
	"autoscaler/cf"
	"autoscaler/db"

	"code.cloudfoundry.org/lager"
)

type ActiveSchedule struct {
	ScheduleId         string
	InstanceMin        int `json:"instance_min_count"`
	InstanceMax        int `json:"instance_max_count"`
	InstanceMinInitial int `json:"initial_min_instance_count"`
}

type AppSchedules struct {
	cfClient cf.CfClient
	logger   lager.Logger
	policyDB db.PolicyDB
}

func NewAppSchedules(logger lager.Logger, cfClient cf.CfClient, policyDB db.PolicyDB) *AppSchedules {
	return &AppSchedules{
		cfClient: cfClient,
		policyDB: policyDB,
		logger:   logger,
	}
}

func (as *AppSchedules) SetActiveSchedule(appId string, schedule *ActiveSchedule) error {
	logger := as.logger.WithData(lager.Data{"appId": appId, "schedule": schedule})

	instances, err := as.cfClient.GetAppInstances(appId)
	if err != nil {
		logger.Error("failed-set-active-schedule-get-app-instances", err)
		return err
	}

	instanceMin := schedule.InstanceMinInitial
	if instanceMin <= 0 {
		instanceMin = schedule.InstanceMin
	}

	newInstances := instances
	if newInstances < instanceMin {
		newInstances = instanceMin
	} else if newInstances > schedule.InstanceMax {
		newInstances = schedule.InstanceMax
	}

	if newInstances != instances {
		err = as.cfClient.SetAppInstances(appId, newInstances)
		if err != nil {
			logger.Error("failed-set-active-schedule-set-app-instances", err)
			return err
		}
	}
	return nil
}

func (as *AppSchedules) RemoveActiveSchedule(appId string, scheduleId string) error {
	logger := as.logger.WithData(lager.Data{"appId": appId, "scheduleId": scheduleId})

	instances, err := as.cfClient.GetAppInstances(appId)
	if err != nil {
		logger.Error("failed-remove-active-schedule-get-app-instances", err)
		return err
	}

	policy, err := as.policyDB.GetAppPolicy(appId)
	if err != nil {
		logger.Error("failed-remove-active-schedule-get-app-policy", err)
		return err
	}

	newInstances := instances
	if newInstances < policy.InstanceMin {
		newInstances = policy.InstanceMin
	} else if newInstances > policy.InstanceMax {
		newInstances = policy.InstanceMax
	}

	if newInstances != instances {
		err = as.cfClient.SetAppInstances(appId, newInstances)
		if err != nil {
			logger.Error("failed-remove-active-schedule-set-app-instances", err)
			return err
		}
	}
	return nil
}
