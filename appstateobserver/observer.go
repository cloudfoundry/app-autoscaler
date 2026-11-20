package appstateobserver

import (
	"context"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstate"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/lager/v3"
)

type Membership interface {
	GetAppIds(context.Context) (map[string]bool, error)
}
type CF interface {
	cf.AuditEventSource
	GetAppAndProcesses(context.Context, cf.Guid) (*cf.AppAndProcesses, error)
	GetAppRoutes(context.Context, cf.Guid) ([]cf.Route, error)
}

type Observer struct {
	Store             *Store
	CF                CF
	Membership        Membership
	Logger            lager.Logger
	ReconcileInterval time.Duration
	Window            time.Duration
}

func (o *Observer) Poll(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return o.Store.Poll(ctx, func(checkpoint time.Time) (Window, error) {
		until := time.Now().UTC().Add(-2 * time.Second).Truncate(time.Second)
		if max := checkpoint.Add(o.Window); until.After(max) {
			until = max
		}
		if until.Before(checkpoint) {
			until = checkpoint
		}
		managed, err := o.Membership.GetAppIds(ctx)
		if err != nil {
			return Window{}, err
		}
		events, err := o.CF.ListAuditEvents(ctx, checkpoint.Add(-time.Minute), until)
		return Window{Until: until, Events: events, Managed: managed}, err
	})
}

func (o *Observer) Refresh(ctx context.Context) error {
	items, err := o.Store.Pending(ctx, time.Now().Add(-o.ReconcileInterval))
	if err != nil {
		return err
	}
	// Bounded parallel CF reads, independent of journal polling.
	jobs := make(chan appstate.Observation, len(items))
	for _, item := range items {
		jobs <- item
	}
	close(jobs)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for item := range jobs {
				if err := o.refresh(ctx, item); err != nil {
					o.Logger.Error("refresh-app-state", err, lager.Data{"appID": item.AppID})
					_ = o.Store.Retry(ctx, item)
				}
			}
		})
	}
	wg.Wait()
	return nil
}

func (o *Observer) refresh(ctx context.Context, item appstate.Observation) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	expected := item.Revision
	started := time.Now().UTC()
	app, err := o.CF.GetAppAndProcesses(ctx, cf.Guid(item.AppID))
	if cf.IsNotFound(err) {
		item.Deleted = true
		item.Dirty = false
		item.VerifiedAt = started
		item.Routes = nil
		return o.Store.Save(ctx, expected, item)
	}
	if err != nil {
		return err
	}
	routes, err := o.CF.GetAppRoutes(ctx, cf.Guid(item.AppID))
	if err != nil {
		return err
	}
	item.State = app.App.State
	item.Instances = app.Processes.GetInstances()
	item.Disabled = app.App.DisableAutoscaling != nil
	item.Routes = nil
	for _, route := range routes {
		item.Routes = append(item.Routes, route.URL)
	}
	item.Deleted = false
	item.Dirty = false
	item.VerifiedAt = started
	return o.Store.Save(ctx, expected, item)
}
