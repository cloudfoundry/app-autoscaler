package scalingengine

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/models"

	"fmt"
	"strconv"
	"strings"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type ScalingEngine interface {
	Scale(appId string, trigger *models.Trigger) (int, error)
	ComputeNewInstances(currentInstances int, adjustment string) (int, error)
	SetActiveSchedule(appId string, schedule *models.ActiveSchedule) error
	RemoveActiveSchedule(appId string, scheduleId string) error
}

type scalingEngine struct {
	logger     lager.Logger
	cfClient   cf.CfClient
	policyDB   db.PolicyDB
	historyDB  db.HistoryDB
	scheduleDB db.ScheduleDB
	clock      clock.Clock
}

func NewScalingEngine(logger lager.Logger, cfClient cf.CfClient, policyDB db.PolicyDB, historyDB db.HistoryDB, scheduleDB db.ScheduleDB, clock clock.Clock) ScalingEngine {
	return &scalingEngine{
		logger:     logger.Session("scale"),
		cfClient:   cfClient,
		policyDB:   policyDB,
		historyDB:  historyDB,
		scheduleDB: scheduleDB,
		clock:      clock,
	}
}

func (s *scalingEngine) Scale(appId string, trigger *models.Trigger) (int, error) {
	logger := s.logger.WithData(lager.Data{"appId": appId})

	now := s.clock.Now()
	history := &models.AppScalingHistory{
		AppId:        appId,
		Timestamp:    now.UnixNano(),
		ScalingType:  models.ScalingTypeDynamic,
		OldInstances: -1,
		NewInstances: -1,
		Reason:       getDynamicScalingReason(trigger),
	}

	defer s.historyDB.SaveScalingHistory(history)

	instances, err := s.cfClient.GetAppInstances(appId)
	if err != nil {
		logger.Error("failed-to-get-app-instances", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get app instances"
		return -1, err
	}
	history.OldInstances = instances

	ok, err := s.historyDB.CanScaleApp(appId)
	if err != nil {
		logger.Error("failed-check-cooldown", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to check app cooldown setting"
		return -1, err
	}
	if !ok {
		history.Status = models.ScalingStatusIgnored
		history.NewInstances = instances
		history.Message = "app in cooldown period"
		return instances, nil
	}

	newInstances, err := s.ComputeNewInstances(instances, trigger.Adjustment)
	if err != nil {
		logger.Error("failed-compute-new-instance", err, lager.Data{"instances": instances, "adjustment": trigger.Adjustment})
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to compute new app instances"
		return -1, err
	}

	schedule, err := s.scheduleDB.GetActiveSchedule(appId)
	if err != nil {
		logger.Error("failed-get-active-schedule", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get active schedule"
		return -1, err
	}

	var instanceMin, instanceMax int

	if schedule != nil {
		instanceMin = schedule.InstanceMin
		instanceMax = schedule.InstanceMax
	} else {
		var policy *models.ScalingPolicy
		policy, err = s.policyDB.GetAppPolicy(appId)
		if err != nil {
			logger.Error("failed-get-app-policy", err)
			history.Status = models.ScalingStatusFailed
			history.Error = "failed to get scaling policy"
			return -1, err
		} else {
			instanceMin = policy.InstanceMin
			instanceMax = policy.InstanceMax
		}
	}

	if newInstances < instanceMin {
		newInstances = instanceMin
		history.Message = fmt.Sprintf("limited by min instances %d", instanceMin)
	} else if newInstances > instanceMax {
		newInstances = instanceMax
		history.Message = fmt.Sprintf("limited by max instances %d", instanceMax)
	}
	history.NewInstances = newInstances

	if newInstances == instances {
		history.Status = models.ScalingStatusIgnored
		return newInstances, nil
	}

	err = s.cfClient.SetAppInstances(appId, newInstances)
	if err != nil {
		logger.Error("failed-to-set-app-instances", err, lager.Data{"newInstances": newInstances})
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to set app instances"
		return -1, err
	}

	history.Status = models.ScalingStatusSucceeded

	err = s.historyDB.UpdateScalingCooldownExpireTime(appId, now.Add(trigger.CoolDown()).UnixNano())
	if err != nil {
		logger.Error("failed-to-update-scaling-cool-down-expire-time", err, lager.Data{"newInstances": newInstances})
	}

	return newInstances, nil
}

func (s *scalingEngine) ComputeNewInstances(currentInstances int, adjustment string) (int, error) {
	var newInstances int
	if strings.HasSuffix(adjustment, "%") {
		percentage, err := strconv.ParseFloat(strings.TrimSuffix(adjustment, "%"), 32)
		if err != nil {
			s.logger.Error("failed-to-parse-percentage", err, lager.Data{"adjustment": adjustment})
			return -1, err
		}
		newInstances = int(float64(currentInstances)*(1+percentage/100) + 0.5)
	} else {
		step, err := strconv.ParseInt(adjustment, 10, 32)
		if err != nil {
			s.logger.Error("failed-to-parse-step-adjustment", err, lager.Data{"adjustment": adjustment})
			return -1, err
		}
		newInstances = int(step) + currentInstances
	}

	return newInstances, nil
}

func (s *scalingEngine) SetActiveSchedule(appId string, schedule *models.ActiveSchedule) error {
	logger := s.logger.WithData(lager.Data{"appId": appId, "schedule": schedule})

	now := s.clock.Now()
	history := &models.AppScalingHistory{
		AppId:        appId,
		Timestamp:    now.UnixNano(),
		ScalingType:  models.ScalingTypeSchedule,
		OldInstances: -1,
		NewInstances: -1,
		Reason:       getScheduledScalingReason(schedule),
	}
	defer s.historyDB.SaveScalingHistory(history)

	instances, err := s.cfClient.GetAppInstances(appId)
	if err != nil {
		logger.Error("failed-to-get-app-instances", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get app instances"
		return err
	}
	history.OldInstances = instances

	instanceMin := schedule.InstanceMinInitial
	if schedule.InstanceMin > instanceMin {
		instanceMin = schedule.InstanceMin
	}

	newInstances := instances
	if newInstances < instanceMin {
		newInstances = instanceMin
		history.Message = fmt.Sprintf("limited by min instances %d", instanceMin)
	} else if newInstances > schedule.InstanceMax {
		newInstances = schedule.InstanceMax
		history.Message = fmt.Sprintf("limited by max instances %d", instanceMin)
	}

	history.NewInstances = newInstances

	if newInstances == instances {
		history.Status = models.ScalingStatusIgnored
		return nil
	}

	err = s.cfClient.SetAppInstances(appId, newInstances)
	if err != nil {
		logger.Error("failed-to-set-app-instances", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to set app instances"
		return err
	}
	history.Status = models.ScalingStatusSucceeded
	return nil
}

func (s *scalingEngine) RemoveActiveSchedule(appId string, scheduleId string) error {
	logger := s.logger.WithData(lager.Data{"appId": appId, "scheduleId": scheduleId})

	now := s.clock.Now()
	history := &models.AppScalingHistory{
		AppId:        appId,
		Timestamp:    now.UnixNano(),
		ScalingType:  models.ScalingTypeSchedule,
		OldInstances: -1,
		NewInstances: -1,
		Reason:       "schedule ends",
	}
	defer s.historyDB.SaveScalingHistory(history)

	instances, err := s.cfClient.GetAppInstances(appId)
	if err != nil {
		logger.Error("failed-to-get-app-instances", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get app instances"
		return err
	}
	history.OldInstances = instances

	policy, err := s.policyDB.GetAppPolicy(appId)
	if err != nil {
		logger.Error("failed-to-get-app-policy", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get app policy"
		return err
	}

	newInstances := instances
	if newInstances < policy.InstanceMin {
		newInstances = policy.InstanceMin
		history.Message = fmt.Sprintf("limited by min instances %d", policy.InstanceMin)
	} else if newInstances > policy.InstanceMax {
		newInstances = policy.InstanceMax
		history.Message = fmt.Sprintf("limited by max instances %d", policy.InstanceMax)
	}

	history.NewInstances = newInstances

	if newInstances == instances {
		history.Status = models.ScalingStatusIgnored
		return nil
	}

	err = s.cfClient.SetAppInstances(appId, newInstances)
	if err != nil {
		logger.Error("failed-to-set-app-instances", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to set app instances"
		return err
	}
	history.Status = models.ScalingStatusSucceeded
	return nil
}

func getDynamicScalingReason(trigger *models.Trigger) string {
	return fmt.Sprintf("%s instance(s) because %s %s %d for %d seconds",
		trigger.Adjustment,
		trigger.MetricType,
		trigger.Operator,
		trigger.Threshold,
		trigger.BreachDurationSeconds)
}

func getScheduledScalingReason(schedule *models.ActiveSchedule) string {
	return fmt.Sprintf("schedule starts with instance min %d, instance max %d and instance min initial %d",
		schedule.InstanceMin, schedule.InstanceMax, schedule.InstanceMinInitial)
}
