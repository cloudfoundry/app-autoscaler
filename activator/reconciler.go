package activator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstate"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type AppReader interface {
	GetAppAndProcesses(context.Context, cf.Guid) (*cf.AppAndProcesses, error)
}

type PolicyReader interface {
	GetAppPolicy(context.Context, string) (*models.PolicyDefinition, error)
}

// Reconciler consumes state changes, but makes parking decisions only after a
// fresh CF read. It also rebuilds the local route registry after a restart.
// Each activator replica registers its own backend and consumes the whole feed.
type Reconciler struct {
	Source            appstate.Source
	CF                AppReader
	Policies          PolicyReader
	Parker            Parker
	Registry          Registry
	MaxAge            time.Duration
	ReconcileInterval time.Duration
	cursor            int64
	observed          map[string]appstate.Observation
	pending           map[string]appstate.Observation
	lastReconcile     time.Time
}

func (r *Reconciler) Poll(ctx context.Context) error {
	if r.observed == nil {
		r.observed = make(map[string]appstate.Observation)
		r.pending = make(map[string]appstate.Observation)
	}
	for {
		changes, err := r.Source.Changes(ctx, r.cursor)
		if err != nil {
			return err
		}
		if time.Since(changes.CheckedAt) > r.MaxAge {
			return fmt.Errorf("observer event reader is stale")
		}
		if changes.Reset {
			r.pending = make(map[string]appstate.Observation)
			r.observed = make(map[string]appstate.Observation)
			// Recheck locally parked apps even if a restored journal no longer
			// contains their tombstone. Policy/CF verification decides cleanup.
			for _, appID := range r.Registry.ParkedApps() {
				r.pending[appID] = appstate.Observation{AppID: appID, Managed: true}
			}
		}
		for _, item := range changes.Items {
			r.observed[item.AppID] = item
			if item.Dirty {
				delete(r.pending, item.AppID)
				continue
			}
			r.pending[item.AppID] = item
		}
		r.cursor = changes.Cursor
		if !changes.More {
			break
		}
	}
	interval := r.ReconcileInterval
	if interval <= 0 {
		interval = time.Minute
	}
	if time.Since(r.lastReconcile) >= interval {
		// A slower safety net catches policy changes and registry loss. Normal
		// journal polls never inspect every parked app.
		for appID, item := range r.observed {
			if !item.Dirty && (item.Instances == 0 || r.Registry.IsParked(appID)) {
				r.pending[appID] = item
			}
		}
		r.lastReconcile = time.Now()
	}
	var errs []error
	for appID, item := range r.pending {
		if err := r.reconcile(ctx, item); err != nil {
			errs = append(errs, fmt.Errorf("reconcile %s: %w", appID, err))
			continue
		}
		delete(r.pending, appID)
		if !item.Managed || item.Deleted {
			delete(r.observed, appID)
		}
	}
	return errors.Join(errs...)
}

func (r *Reconciler) reconcile(ctx context.Context, item appstate.Observation) error {
	if !item.Managed || item.Deleted {
		if r.Registry.IsParked(item.AppID) {
			return r.Parker.Unpark(ctx, item.AppID)
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	policy, err := r.Policies.GetAppPolicy(ctx, item.AppID)
	if err != nil {
		return err
	}
	if policy == nil || policy.InstanceMin != 0 {
		if r.Registry.IsParked(item.AppID) {
			return r.Parker.Unpark(ctx, item.AppID)
		}
		return nil
	}
	app, err := r.CF.GetAppAndProcesses(ctx, cf.Guid(item.AppID))
	if cf.IsNotFound(err) {
		if r.Registry.IsParked(item.AppID) {
			return r.Parker.Unpark(ctx, item.AppID)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if app.App.State != models.AppStatusStarted || app.App.DisableAutoscaling != nil {
		if r.Registry.IsParked(item.AppID) {
			return r.Parker.Unpark(ctx, item.AppID)
		}
		return nil
	}
	if app.Processes.GetInstances() == 0 {
		return r.Parker.Park(ctx, item.AppID)
	}
	// Desired instances > 0 is not readiness. NATS removes our backend only
	// when a real app endpoint is advertised (including manual scale-up).
	return nil
}
