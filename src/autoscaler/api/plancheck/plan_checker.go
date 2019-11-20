package plancheck

import (
	"autoscaler/api/config"
	"autoscaler/models"
	"fmt"

	"code.cloudfoundry.org/lager"
)

type PlanChecker struct {
	conf   *config.PlanCheckConfig
	logger lager.Logger
}

func NewPlanChecker(config *config.PlanCheckConfig, logger lager.Logger) *PlanChecker {
	return &PlanChecker{
		conf:   config,
		logger: logger.Session("plan-checker"),
	}
}

func (pc PlanChecker) CheckPlan(policy models.ScalingPolicy, planID string) (bool, string, error) {
	if pc.conf == nil {
		pc.logger.Info("plan-checker-not-configured-allowing-all")
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
