package scalingengine

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
	"github.com/google/uuid"
)

var memoryAdjustmentPattern = regexp.MustCompile(`^[+-][1-9][0-9]*%?$`)

// ComputeNewMemoryMB uses integer arithmetic, rounds percentages upward, then clamps.
// Arbitrarily large valid adjustments saturate at the policy bounds without overflow.
func ComputeNewMemoryMB(current, minimum, maximum int, adjustment string) (int, error) {
	if current <= 0 || minimum <= 0 || maximum < minimum {
		return 0, fmt.Errorf("invalid memory limits")
	}
	if !memoryAdjustmentPattern.MatchString(adjustment) {
		return 0, fmt.Errorf("invalid memory adjustment %q", adjustment)
	}
	step, ok := new(big.Int).SetString(strings.TrimSuffix(adjustment, "%"), 10)
	if !ok {
		return 0, fmt.Errorf("invalid memory adjustment %q", adjustment)
	}
	target := big.NewInt(int64(current))
	if strings.HasSuffix(adjustment, "%") {
		if step.Cmp(big.NewInt(-100)) < 0 {
			return 0, fmt.Errorf("percentage reduction exceeds 100 percent")
		}
		factor := new(big.Int).Add(big.NewInt(100), step)
		target.Mul(target, factor)
		target.Add(target, big.NewInt(99))
		target.Quo(target, big.NewInt(100))
	} else {
		target.Add(target, step)
	}
	if target.Cmp(big.NewInt(int64(minimum))) < 0 {
		return minimum, nil
	}
	if target.Cmp(big.NewInt(int64(maximum))) > 0 {
		return maximum, nil
	}
	return int(target.Int64()), nil
}

func scalingPriorityBlockReason(priority, dimension models.ScalingDimension, growing bool, memory, instances, minMemory, maxMemory, minInstances, maxInstances int) string {
	switch {
	case priority == models.ScalingDimensionVertical && dimension == models.ScalingDimensionHorizontal && growing && memory < maxMemory:
		return "priority_requires_max_memory"
	case priority == models.ScalingDimensionVertical && dimension == models.ScalingDimensionVertical && !growing && instances > minInstances:
		return "priority_requires_min_instances"
	case priority == models.ScalingDimensionHorizontal && dimension == models.ScalingDimensionVertical && growing && instances < maxInstances:
		return "priority_requires_max_instances"
	case priority == models.ScalingDimensionHorizontal && dimension == models.ScalingDimensionHorizontal && !growing && memory > minMemory:
		return "priority_requires_min_memory"
	default:
		return ""
	}
}

func (s *scalingEngine) deploymentBlockReason(ctx context.Context, appId string) (string, error) {
	active, err := s.cfClient.HasActiveDeployment(ctx, cf.Guid(appId))
	if err != nil {
		return "", err
	}
	if active {
		return "active_deployment", nil
	}
	open, err := s.scalingEngineDB.HasOpenScalingHistory(ctx, appId)
	if err != nil {
		return "", err
	}
	if open {
		return "unresolved_deployment", nil
	}
	return "", nil
}

// Do not commit a schedule change while its CAPI mutation is blocked. The periodic
// synchronizer will re-read the actual scheduler state instead of replaying this request.
func (s *scalingEngine) checkScheduleDeployment(ctx context.Context, appId, reason string) error {
	blocked, err := s.deploymentBlockReason(ctx, appId)
	if err == nil && blocked == "" {
		return nil
	}
	h := &models.AppScalingHistory{AppId: appId, Timestamp: s.clock.Now().UnixNano(),
		ScalingType: models.ScalingTypeSchedule, Status: models.ScalingStatusIgnored,
		OldInstances: -1, NewInstances: -1, Reason: reason, Message: blocked}
	if err != nil {
		h.Status = models.ScalingStatusFailed
		h.Error = "failed to check deployment state: " + err.Error()
	} else {
		err = fmt.Errorf("schedule adjustment deferred: %s", blocked)
	}
	if saveErr := s.scalingEngineDB.SaveScalingHistory(h); saveErr != nil {
		s.logger.Error("failed-to-save-deferred-schedule-history", saveErr)
	}
	return err
}

func (s *scalingEngine) scaleVertical(ctx context.Context, appId string, trigger *models.Trigger) (*models.AppScalingResult, error) {
	s.appLock.GetLock(appId).Lock()
	defer s.appLock.GetLock(appId).Unlock()
	h := &models.AppScalingHistory{AppId: appId, Timestamp: s.clock.Now().UnixNano(),
		ScalingType: models.ScalingTypeDynamic, ScalingDimension: models.ScalingDimensionVertical,
		Status: models.ScalingStatusIgnored, OldInstances: -1, NewInstances: -1,
		Reason: fmt.Sprintf("%s MiB memory adjustment because %s %s %d%s for %d seconds",
			trigger.Adjustment, trigger.MetricType, trigger.Operator, trigger.Threshold, trigger.MetricUnit, trigger.BreachDurationSeconds)}
	persisted := false
	defer func() {
		if !persisted {
			if err := s.scalingEngineDB.SaveScalingHistory(h); err != nil {
				s.logger.Error("failed-to-save-vertical-scaling-history", err)
			}
		}
	}()
	result := &models.AppScalingResult{AppId: appId, ScalingDimension: models.ScalingDimensionVertical}
	fail := func(message string, err error) (*models.AppScalingResult, error) {
		h.Status, h.Error = models.ScalingStatusFailed, message+": "+err.Error()
		s.logger.Error("vertical-scaling-failed", err, lager.Data{"appId": appId, "stage": message})
		return nil, err
	}
	ignore := func(reason string) (*models.AppScalingResult, error) {
		h.Message = reason
		result.Status = models.ScalingStatusIgnored
		result.Reason = reason
		return result, nil
	}
	app, err := s.cfClient.GetAppAndProcesses(ctx, cf.Guid(appId))
	if err != nil {
		return fail("failed to get app info", err)
	}
	if app == nil || app.App == nil {
		return fail("failed to get app info", fmt.Errorf("empty app response"))
	}
	instances, memory := app.Processes.GetInstances(), app.Processes.GetMemoryMB()
	h.OldInstances, h.NewInstances, h.OldMemoryMB, h.NewMemoryMB = instances, instances, memory, memory
	if strings.ToUpper(app.App.State) != models.AppStatusStarted {
		return ignore("app is not started")
	}
	if app.App.DisableAutoscaling != nil {
		return ignore("app has disable-autoscaling label: " + *app.App.DisableAutoscaling)
	}
	policy, err := s.policyDB.GetAppPolicy(ctx, appId)
	if err != nil {
		return fail("failed to get scaling policy", err)
	}
	if policy == nil {
		return ignore("app does not have policy set")
	}
	blocked, err := s.deploymentBlockReason(ctx, appId)
	if err != nil {
		return fail("failed to check deployment state", err)
	}
	if blocked != "" {
		return ignore(blocked)
	}
	canScale, expiresAt, err := s.scalingEngineDB.CanScaleApp(appId)
	if err != nil {
		return fail("failed to check app cooldown setting", err)
	}
	if !canScale {
		result.CooldownExpiredAt = expiresAt
		return ignore("app in cooldown period")
	}
	if err = validateVerticalTrigger(policy, trigger); err != nil {
		return fail("invalid vertical trigger", err)
	}
	if memory <= 0 || instances <= 0 {
		return fail("web process is not scalable", fmt.Errorf("web process has missing/ambiguous memory or zero instances"))
	}
	schedule, err := s.scalingEngineDB.GetActiveSchedule(appId)
	if err != nil {
		return fail("failed to get active schedule", err)
	}
	minInstances, maxInstances := policy.InstanceMin, policy.InstanceMax
	if schedule != nil {
		minInstances, maxInstances = schedule.InstanceMin, schedule.InstanceMax
	}
	target, err := ComputeNewMemoryMB(memory, policy.MinMemoryMB, policy.MaxMemoryMB, trigger.Adjustment)
	if err != nil {
		return fail("failed to compute new memory limit", err)
	}
	h.NewMemoryMB = target
	if target == memory {
		return ignore("unchanged_target")
	}
	if reason := scalingPriorityBlockReason(policy.ScalingPriority, models.ScalingDimensionVertical, target > memory,
		memory, instances, policy.MinMemoryMB, policy.MaxMemoryMB, minInstances, maxInstances); reason != "" {
		return ignore(reason)
	}
	cooldownSeconds := trigger.CoolDownSeconds
	if cooldownSeconds == 0 {
		cooldownSeconds = s.defaultCoolDownSecs
	}
	if cooldownSeconds <= 0 || int64(cooldownSeconds) > math.MaxInt32 {
		return fail("invalid cooldown duration", fmt.Errorf("cooldown must be positive and fit the database integer column"))
	}
	h.CorrelationID, h.CoolDownSeconds = uuid.NewString(), cooldownSeconds
	h.Status = models.ScalingStatusPending
	h.Message = "rolling deployment requested"
	if err = s.scalingEngineDB.SaveScalingHistory(h); err != nil {
		// Without a durable request no POST is safe. Avoid a second insert in the defer.
		persisted = true
		return nil, fmt.Errorf("failed to persist deployment request: %w", err)
	}
	persisted = true
	s.logger.Info("vertical-deployment-requested", lager.Data{"appId": appId, "correlationID": h.CorrelationID,
		"oldMemoryMB": memory, "newMemoryMB": target, "coolDownSeconds": h.CoolDownSeconds})
	deployment, err := s.cfClient.CreateRollingMemoryDeployment(ctx, cf.Guid(appId), target, h.CorrelationID)
	if err != nil {
		if cf.IsDeploymentRequestRejected(err) {
			_, completeErr := s.scalingEngineDB.CompleteScalingHistory(ctx, h.CorrelationID, models.ScalingStatusFailed,
				s.clock.Now().UnixNano(), "deployment request rejected", err.Error())
			if completeErr != nil {
				s.logger.Error("failed-to-complete-rejected-deployment", completeErr)
			}
			return nil, err
		}
		return s.unknownDeploymentResult(ctx, result, h.CorrelationID, err)
	}
	if deployment == nil || deployment.GUID == "" {
		return s.unknownDeploymentResult(ctx, result, h.CorrelationID, fmt.Errorf("deployment response has no GUID"))
	}
	result.Status, result.DeploymentGUID = models.ScalingStatusPending, deployment.GUID
	result.Adjustment = target - memory
	s.logger.Info("vertical-deployment-accepted", lager.Data{"appId": appId, "deploymentGUID": deployment.GUID,
		"correlationID": h.CorrelationID, "oldMemoryMB": memory, "newMemoryMB": target})
	if err = s.scalingEngineDB.SetScalingDeployment(ctx, h.CorrelationID, deployment.GUID); err != nil {
		// The accepted deployment can still be recovered by its correlation metadata.
		s.logger.Error("failed-to-attach-deployment-to-history", err)
	}
	return result, nil
}

func (s *scalingEngine) unknownDeploymentResult(ctx context.Context, result *models.AppScalingResult, correlationID string, requestErr error) (*models.AppScalingResult, error) {
	if err := s.scalingEngineDB.MarkScalingRequestUnknown(ctx, correlationID, requestErr.Error()); err != nil {
		s.logger.Error("failed-to-mark-unknown-deployment-request", err)
	}
	s.logger.Error("deployment-request-outcome-unknown", requestErr, lager.Data{"appId": result.AppId, "correlationID": correlationID})
	result.Status = models.ScalingStatusUnknown
	return result, nil
}

func validateVerticalTrigger(policy *models.PolicyDefinition, trigger *models.Trigger) error {
	if trigger.MetricType != models.MetricNameEntitlementUtil || trigger.Threshold < 0 || trigger.BreachDurationSeconds <= 0 || trigger.CoolDownSeconds < 0 {
		return fmt.Errorf("invalid entitlement trigger fields")
	}
	for _, rule := range policy.VerticalScalingRules {
		if rule != nil && rule.MetricType == trigger.MetricType && rule.Threshold == trigger.Threshold &&
			rule.Operator == trigger.Operator && rule.Adjustment == trigger.Adjustment &&
			(rule.BreachDurationSeconds <= 0 || rule.BreachDurationSeconds == trigger.BreachDurationSeconds) &&
			(rule.CoolDownSeconds <= 0 || rule.CoolDownSeconds == trigger.CoolDownSeconds) {
			return nil
		}
	}
	return fmt.Errorf("trigger no longer matches a current vertical rule")
}
