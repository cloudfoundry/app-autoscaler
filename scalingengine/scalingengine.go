package scalingengine

import (
	"context"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"fmt"
	"strconv"
	"strings"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

type ScalingEngine interface {
	Scale(appId string, trigger *models.Trigger) (*models.AppScalingResult, error)
	ComputeNewInstances(currentInstances int, adjustment string) (int, error)
	SetActiveSchedule(appId string, schedule *models.ActiveSchedule) error
	RemoveActiveSchedule(appId string, scheduleId string) error
}

type scalingEngine struct {
	logger              lager.Logger
	cfClient            cf.CFClient
	policyDB            db.PolicyDB
	scalingEngineDB     db.ScalingEngineDB
	appLock             *StripedLock
	clock               clock.Clock
	defaultCoolDownSecs int
}

type ActiveScheduleNotFoundError struct {
}
type AppNotFoundError struct {
}

func (ase *ActiveScheduleNotFoundError) Error() string {
	return "active schedule not found"
}

func NewScalingEngine(logger lager.Logger, cfClient cf.CFClient, policyDB db.PolicyDB, scalingEngineDB db.ScalingEngineDB, clock clock.Clock, defaultCoolDownSecs int, lockSize int) ScalingEngine {
	return &scalingEngine{
		logger:              logger.Session("scalingEngine"),
		cfClient:            cfClient,
		policyDB:            policyDB,
		scalingEngineDB:     scalingEngineDB,
		appLock:             NewStripedLock(lockSize),
		clock:               clock,
		defaultCoolDownSecs: defaultCoolDownSecs,
	}
}

func (s *scalingEngine) Scale(appId string, trigger *models.Trigger) (*models.AppScalingResult, error) {
	logger := s.logger.WithData(lager.Data{"appId": appId})

	s.appLock.GetLock(appId).Lock()
	defer s.appLock.GetLock(appId).Unlock()

	now := s.clock.Now()
	history := &models.AppScalingHistory{
		AppId:        appId,
		Timestamp:    now.UnixNano(),
		ScalingType:  models.ScalingTypeDynamic,
		OldInstances: -1,
		NewInstances: -1,
		Reason:       getDynamicScalingReason(trigger),
	}

	defer func() {
		err := s.scalingEngineDB.SaveScalingHistory(history)
		if err != nil {
			s.logger.Error("Scale failed to save history", err)
		}
	}()

	result := &models.AppScalingResult{
		AppId:             appId,
		Adjustment:        0,
		CooldownExpiredAt: 0,
	}

	appAndProcesses, err := s.cfClient.GetAppAndProcesses(cf.Guid(appId))
	if err != nil {
		logger.Error("failed-to-get-app-info", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get app info: " + err.Error()
		return nil, err
	}
	instances := appAndProcesses.Processes.GetInstances()
	history.OldInstances = instances

	if strings.ToUpper(appAndProcesses.App.State) != models.AppStatusStarted {
		logger.Info("check-app-state", lager.Data{"message": "ignore scaling since app is not started"})
		history.Status = models.ScalingStatusIgnored
		history.NewInstances = instances
		history.Message = "app is not started"
		result.Status = history.Status
		return result, nil
	}

	if disableAutoscaling := appAndProcesses.App.DisableAutoscaling; disableAutoscaling != nil {
		logger.Info("check-app-label", lager.Data{"message": "ignore scaling since app has the label app-autoscaler.cloudfoundry.org/disable-autoscaling set", "label-value": *disableAutoscaling})
		history.Status = models.ScalingStatusIgnored
		history.NewInstances = instances
		appNotScaledDueTolabel := "The application was not scaled as the label " +
			"\"app-autoscaler.cloudfoundry.org/disable-autoscaling\" " +
			"was set on the app."
		if *disableAutoscaling == "" {
			history.Message = appNotScaledDueTolabel
		} else {
			history.Message = fmt.Sprintf(appNotScaledDueTolabel+
				" The content of the label might give a hint on why the label was set: \"%s\"",
				*disableAutoscaling)
		}
		result.Status = history.Status
		return result, nil
	}

	ok, expiredAt, err := s.scalingEngineDB.CanScaleApp(appId)
	if err != nil {
		logger.Error("failed-to-check-cooldown", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to check app cooldown setting"
		return nil, err
	}
	result.CooldownExpiredAt = expiredAt
	if !ok {
		logger.Info("scaling ignored: App in cooldown")
		history.Status = models.ScalingStatusIgnored
		history.NewInstances = instances
		history.Message = "app in cooldown period"
		result.Status = history.Status
		return result, nil
	}

	newInstances, err := s.ComputeNewInstances(instances, trigger.Adjustment)
	if err != nil {
		logger.Error("failed-to-compute-new-instance", err, lager.Data{"instances": instances, "adjustment": trigger.Adjustment})
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to compute new app instances"
		return nil, err
	}

	schedule, err := s.scalingEngineDB.GetActiveSchedule(appId)
	if err != nil {
		logger.Error("failed-to-get-active-schedule", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get active schedule"
		return nil, err
	}

	var instanceMin, instanceMax int

	if schedule != nil {
		instanceMin = schedule.InstanceMin
		instanceMax = schedule.InstanceMax
	} else {
		policy, err := s.policyDB.GetAppPolicy(context.TODO(), appId)
		if err != nil {
			logger.Error("failed-to-get-app-policy", err)
			history.Status = models.ScalingStatusFailed
			history.Error = "failed to get scaling policy"
			return nil, err
		}
		if policy == nil {
			logger.Info("check-get-app-policy", lager.Data{"message": "ignore scaling since app does not have scaling policy"})
			history.Status = models.ScalingStatusIgnored
			history.NewInstances = instances
			history.Message = "app does not have policy set"
			result.Status = history.Status
			result.CooldownExpiredAt = 0
			return result, nil
		}
		instanceMin = policy.InstanceMin
		instanceMax = policy.InstanceMax
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
		logger.Info(fmt.Sprintf("ignoring scale app:%s already has %d instances", appId, newInstances))
		history.Status = models.ScalingStatusIgnored
		result.Status = history.Status
		result.Adjustment = 0
		result.CooldownExpiredAt = 0
		return result, nil
	}

	err = s.cfClient.ScaleAppWebProcess(cf.Guid(appId), newInstances)
	if err != nil {
		logger.Error("failed-to-set-app-instances", err, lager.Data{"newInstances": newInstances})
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to set app instances: " + err.Error()
		return nil, err
	}

	history.Status = models.ScalingStatusSucceeded
	result.Status = history.Status
	result.Adjustment = newInstances - instances
	result.CooldownExpiredAt = now.Add(trigger.CoolDown(s.defaultCoolDownSecs)).UnixNano()
	err = s.scalingEngineDB.UpdateScalingCooldownExpireTime(appId, result.CooldownExpiredAt)
	if err != nil {
		logger.Error("failed-to-update-scaling-cool-down-expire-time", err, lager.Data{"newInstances": newInstances})
	}
	return result, nil
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

		if newInstances == currentInstances {
			if percentage > 0 {
				newInstances = currentInstances + 1
			} else if percentage < 0 {
				newInstances = currentInstances - 1
			}
		}
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

	s.appLock.GetLock(appId).Lock()
	defer s.appLock.GetLock(appId).Unlock()

	currentSchedule, err := s.scalingEngineDB.GetActiveSchedule(appId)
	if err != nil {
		logger.Error("failed-to-get-existing-active-schedule-from-database", err)
		return err
	}

	if currentSchedule != nil {
		if schedule.ScheduleId == currentSchedule.ScheduleId {
			logger.Info("set-active-schedule", lager.Data{"message": "duplicate request to set active schedule"})
			return nil
		}
		logger.Info("set-active-schedule", lager.Data{"message": "an active schedule exists in database", "currentSchedule": currentSchedule})
	}

	err = s.scalingEngineDB.SetActiveSchedule(appId, schedule)
	if err != nil {
		logger.Error("failed-to-set-active-schedule-in-database", err)
		return err
	}

	now := s.clock.Now()
	history := &models.AppScalingHistory{
		AppId:        appId,
		Timestamp:    now.UnixNano(),
		ScalingType:  models.ScalingTypeSchedule,
		OldInstances: -1,
		NewInstances: -1,
		Reason:       getScheduledScalingReason(schedule),
	}
	defer func() {
		err := s.scalingEngineDB.SaveScalingHistory(history)
		if err != nil {
			s.logger.Error("SetActiveSchedule failed to save history", err)
		}
	}()

	processes, err := s.cfClient.GetAppProcesses(cf.Guid(appId), cf.ProcessTypeWeb)
	if err != nil {
		logger.Error("failed-to-get-app-info", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get app info: " + err.Error()
		return err
	}
	instances := processes.GetInstances()
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
		history.Message = fmt.Sprintf("limited by max instances %d", schedule.InstanceMax)
	}

	history.NewInstances = newInstances

	if newInstances == instances {
		history.Status = models.ScalingStatusIgnored
		return nil
	}

	err = s.cfClient.ScaleAppWebProcess(cf.Guid(appId), newInstances)
	if err != nil {
		logger.Error("failed-to-set-app-instances", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to set app instances: " + err.Error()
		return err
	}
	history.Status = models.ScalingStatusSucceeded
	return nil
}

func (s *scalingEngine) RemoveActiveSchedule(appId string, scheduleId string) error {
	logger := s.logger.WithData(lager.Data{"appId": appId, "scheduleId": scheduleId})

	s.appLock.GetLock(appId).Lock()
	defer s.appLock.GetLock(appId).Unlock()

	currentSchedule, err := s.scalingEngineDB.GetActiveSchedule(appId)
	if err != nil {
		logger.Error("failed-to-get-existing-active-schedule-from-database", err)
		return err
	}

	if (currentSchedule == nil) || (currentSchedule.ScheduleId != scheduleId) {
		logger.Info("active-schedule-not-found", lager.Data{"appId": appId, "scheduleId": scheduleId})
		return nil
	}

	err = s.scalingEngineDB.RemoveActiveSchedule(appId)
	if err != nil {
		logger.Error("failed-to-remove-active-schedule-from-database", err)
		return err
	}

	now := s.clock.Now()
	history := &models.AppScalingHistory{
		AppId:        appId,
		Timestamp:    now.UnixNano(),
		ScalingType:  models.ScalingTypeSchedule,
		OldInstances: -1,
		NewInstances: -1,
		Reason:       "schedule ends",
	}
	defer func() {
		err := s.scalingEngineDB.SaveScalingHistory(history)
		if err != nil {
			s.logger.Error("RemoveActiveSchedule failed to save history", err)
		}
	}()

	processes, err := s.cfClient.GetAppProcesses(cf.Guid(appId), cf.ProcessTypeWeb)
	if err != nil {
		if cf.IsNotFound(err) {
			s.logger.Info(fmt.Sprintf("app(%s) missing ignoring scale request", appId))
			history.Status = models.ScalingStatusIgnored
			return nil
		}
		s.logger.Error(fmt.Sprintf("failed getting %s process info", appId), err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get app info: " + err.Error()
		return err
	}
	instances := processes.GetInstances()

	history.OldInstances = instances

	policy, err := s.policyDB.GetAppPolicy(context.TODO(), appId)
	if err != nil {
		logger.Error("failed-to-get-app-policy", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get app policy"
		return err
	}

	if policy == nil {
		s.logger.Info(fmt.Sprintf("app(%s) has no policy ignoring scale", appId))
		history.Status = models.ScalingStatusIgnored
		return nil
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
		s.logger.Info(fmt.Sprintf("ignoring scale app:%s already has %d instances", appId, newInstances))
		history.Status = models.ScalingStatusIgnored
		return nil
	}

	err = s.cfClient.ScaleAppWebProcess(cf.Guid(appId), newInstances)
	if err != nil {
		logger.Error("failed-to-set-app-instances", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to set app instances: " + err.Error()
		return err
	}
	history.Status = models.ScalingStatusSucceeded
	return nil
}

func getDynamicScalingReason(trigger *models.Trigger) string {
	return fmt.Sprintf("%s instance(s) because %s %s %d%s for %d seconds",
		trigger.Adjustment,
		trigger.MetricType,
		trigger.Operator,
		trigger.Threshold,
		trigger.MetricUnit,
		trigger.BreachDurationSeconds)
}

func getScheduledScalingReason(schedule *models.ActiveSchedule) string {
	return fmt.Sprintf("schedule starts with instance min %d, instance max %d and instance min initial %d",
		schedule.InstanceMin, schedule.InstanceMax, schedule.InstanceMinInitial)
}
