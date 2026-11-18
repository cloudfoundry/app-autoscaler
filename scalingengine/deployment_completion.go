package scalingengine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/tedsuo/ifrit"
)

// CheckDeployments recovers every open history entry, including after restart.
// A missing resource or failed GET never implies successful deployment completion.
func CheckDeployments(ctx context.Context, engine ScalingEngine) error {
	s, ok := engine.(*scalingEngine)
	if !ok {
		return fmt.Errorf("engine does not support deployment completion")
	}
	histories, err := s.scalingEngineDB.GetOpenScalingHistories(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, h := range histories {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if h == nil || h.ScalingDimension != models.ScalingDimensionVertical {
			failures = append(failures, fmt.Errorf("invalid open scaling history"))
			continue
		}
		s.appLock.GetLock(h.AppId).Lock()
		err := s.checkDeployment(ctx, h)
		s.appLock.GetLock(h.AppId).Unlock()
		if err != nil {
			s.logger.Error("failed-to-check-deployment-completion", err,
				lager.Data{"appId": h.AppId, "deploymentGUID": h.DeploymentGUID, "correlationID": h.CorrelationID})
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (s *scalingEngine) checkDeployment(ctx context.Context, h *models.AppScalingHistory) error {
	var deployment *cf.Deployment
	var err error
	if h.DeploymentGUID == "" {
		deployment, err = s.cfClient.FindDeployment(ctx, cf.Guid(h.AppId), h.CorrelationID)
		if err != nil {
			return err
		}
		if deployment == nil || deployment.GUID == "" {
			err = fmt.Errorf("deployment request remains unresolved; no correlated deployment found")
			if saveErr := s.scalingEngineDB.MarkScalingRequestUnknown(ctx, h.CorrelationID, err.Error()); saveErr != nil {
				return errors.Join(err, saveErr)
			}
			return err
		}
		if err = s.scalingEngineDB.SetScalingDeployment(ctx, h.CorrelationID, deployment.GUID); err != nil {
			return err
		}
	} else {
		deployment, err = s.cfClient.GetDeployment(ctx, h.DeploymentGUID)
		if err != nil {
			return err
		}
	}
	if deployment == nil {
		return fmt.Errorf("deployment response is empty")
	}
	if strings.ToUpper(deployment.StatusValue) == "ACTIVE" {
		return nil
	}
	if strings.ToUpper(deployment.StatusValue) != "FINALIZED" {
		return fmt.Errorf("unrecognized deployment state %q", deployment.StatusValue)
	}
	var status models.ScalingStatus
	switch strings.ToUpper(deployment.StatusReason) {
	case "DEPLOYED":
		status = models.ScalingStatusSucceeded
	case "SUPERSEDED":
		status = models.ScalingStatusSuperseded
	case "CANCELED", "CANCELLED":
		status = models.ScalingStatusAborted
	default:
		return fmt.Errorf("unrecognized finalized deployment reason %q", deployment.StatusReason)
	}
	completed, err := s.scalingEngineDB.CompleteScalingHistory(ctx, h.CorrelationID, status, s.clock.Now().UnixNano(),
		"deployment finalized: "+deployment.StatusReason, "")
	if err != nil {
		return err
	}
	if completed {
		s.logger.Info("vertical-deployment-completed", lager.Data{"appId": h.AppId, "deploymentGUID": deployment.GUID,
			"correlationID": h.CorrelationID, "oldMemoryMB": h.OldMemoryMB, "newMemoryMB": h.NewMemoryMB, "status": status})
	}
	return nil
}

// NewDeploymentCompletionRunner is an in-process worker, not a separate service.
// Schedule synchronization runs against fresh scheduler state even for foreign
// deployments, which have no autoscaler history entry to complete.
func NewDeploymentCompletionRunner(logger lager.Logger, engine ScalingEngine, clk clock.Clock, interval time.Duration, synchronizeSchedules func()) ifrit.Runner {
	return ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		if interval <= 0 {
			return fmt.Errorf("deployment check interval must be positive")
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			select {
			case <-signals:
				cancel()
			case <-ctx.Done():
			}
		}()
		ticker := clk.NewTicker(interval)
		defer ticker.Stop()
		close(ready)
		for {
			if err := CheckDeployments(ctx, engine); err != nil && ctx.Err() == nil {
				logger.Error("deployment-completion-check-failed", err)
			}
			if ctx.Err() != nil {
				return nil
			}
			if synchronizeSchedules != nil {
				synchronizeSchedules()
			}
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C():
			}
		}
	})
}
