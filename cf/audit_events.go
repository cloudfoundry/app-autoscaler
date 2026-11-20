package cf

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudfoundry/go-cfclient/v3/client"
)

// AuditEventSource is separate from CFClient so consumers of ordinary CF
// operations do not depend on the event journal.
type AuditEventSource interface {
	ListAuditEvents(context.Context, time.Time, time.Time) ([]AuditEvent, error)
}

type AuditEvent struct {
	GUID       string
	CreatedAt  time.Time
	TargetGUID string
	TargetType string
}

func (w *CFClientWrapper) ListAuditEvents(ctx context.Context, from, until time.Time) ([]AuditEvent, error) {
	opts := client.NewAuditEventListOptions()
	opts.PerPage = 1000
	opts.OrderBy = "created_at"
	opts.CreatedAts.AfterOrEqualTo(from)
	opts.CreatedAts.BeforeOrEqualTo(until)
	opts.Types = client.Filter{Values: []string{
		"audit.app.process.scale", "audit.app.process.update", "audit.app.process.create", "audit.app.process.delete",
		"audit.app.start", "audit.app.stop", "audit.app.restart", "audit.app.update", "audit.app.delete-request",
		"audit.app.map-route", "audit.app.unmap-route", "audit.route.update", "audit.route.delete-request",
	}}
	var result []AuditEvent
	// A bounded window and page cap prevent unbounded memory use. A failed
	// window is retried without advancing the checkpoint.
	for range 20 {
		events, pager, err := w.cfClient.AuditEvents.List(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("list audit events: %w", MapCFClientError(err))
		}
		for _, event := range events {
			result = append(result, AuditEvent{GUID: event.GUID, CreatedAt: event.CreatedAt,
				TargetGUID: event.Target.GUID, TargetType: event.Target.Type})
		}
		if !pager.HasNextPage() {
			return result, nil
		}
		pager.NextPage(opts)
	}
	return nil, fmt.Errorf("audit event window exceeds 20000 events; reduce observer window")
}
