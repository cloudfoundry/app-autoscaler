package plancheck

import (
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3"
)

type planChecker struct {
	conf   *config.PlanCheckConfig
	logger lager.Logger
}
type PlanChecker interface {
	CheckPlan(policy *models.ScalingPolicy, planID string) (bool, string, error)
	IsPlanUpdatable(planID string) (bool, error)
}

func NewPlanChecker(config *config.PlanCheckConfig, logger lager.Logger) *planChecker {
	return &planChecker{
		conf:   config,
		logger: logger.Session("plan-checker"),
	}
}

func (pc planChecker) CheckPlan(policy *models.ScalingPolicy, planID string) (bool, string, error) {
	if pc.conf == nil {
		pc.logger.Info("plan-checker-not-configured-allowing-all")
		return true, "", nil
	}
	if policy == nil {
		pc.logger.Info("No policy")
		return true, "", nil
	}
	definition, ok := pc.conf.PlanDefinitions[planID]
	if !ok {
		return false, "", fmt.Errorf(`unknown plan id "%s"`, planID)
	}
	if !definition.PlanCheckEnabled {
		return true, "", nil
	}
	validationResult := ""

	if policy.Schedules != nil {
		numSchedules := len(policy.Schedules.RecurringSchedules) + len(policy.Schedules.SpecificDateSchedules)
		if numSchedules > definition.SchedulesCount {
			validationResult += fmt.Sprintf("Too many schedules: Found %d schedules, but a maximum of %d schedules are allowed for this service plan. ", numSchedules, definition.SchedulesCount)
		}
	}

	numScalingRules := len(policy.ScalingRules)
	if numScalingRules > definition.ScalingRulesCount {
		validationResult += fmt.Sprintf("Too many scaling rules: Found %d scaling rules, but a maximum of %d scaling rules are allowed for this service plan. ", numScalingRules, definition.SchedulesCount)
	}

	if validationResult == "" {
		return true, "", nil
	} else {
		return false, validationResult, nil
	}
}

func (pc planChecker) IsPlanUpdatable(planID string) (bool, error) {
	if pc.conf == nil {
		return true, nil
	}

	definition, exists := pc.conf.PlanDefinitions[planID]
	if !exists {
		return false, fmt.Errorf(`unknown plan id "%s"`, planID)
	}

	return definition.PlanUpdateable, nil
}
