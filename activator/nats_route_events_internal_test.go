package activator

import (
	"context"
	"errors"
	"testing"
	"time"

	"code.cloudfoundry.org/lager/v3/lagertest"
	"github.com/nats-io/nats.go"
)

// TestDecodeReadinessEvents covers the router.register filtering logic that
// drives Loop B readiness: a real app's backend registration yields one Upsert
// per URI, while the activator's own registration (matched by instance GUID or
// host+tls_port) yields none. Kept as a plain Go test (no NATS server) so the
// pure decode/self-filter logic is exercised directly.
func TestDecodeReadinessEvents(t *testing.T) {
	self := SelfBackend{
		Host:                "10.0.1.24",
		TLSPort:             61032,
		ServerCertDomainSAN: "activator-guid",
		InstanceGUID:        "activator-guid",
		AppGUID:             "activator-app-guid",
	}

	cases := []struct {
		name     string
		payload  string
		wantURIs []string // nil = expect no events; error cases handled separately
		wantErr  bool
	}{
		{
			name:     "real app backend -> one Upsert",
			payload:  `{"uris":["app-1.example.com"],"host":"10.0.1.99","tls_port":61099,"private_instance_id":"real-app"}`,
			wantURIs: []string{"app-1.example.com"},
		},
		{
			name:     "multi-route registration -> one Upsert per URI",
			payload:  `{"uris":["a.example.com","b.example.com"],"host":"10.0.1.99","tls_port":61099,"private_instance_id":"real-app"}`,
			wantURIs: []string{"a.example.com", "b.example.com"},
		},
		{
			name:    "peer activator registration -> dropped",
			payload: `{"uris":["app-1.example.com"],"host":"10.0.1.25","tls_port":61033,"app":"activator-app-guid","private_instance_id":"peer-guid"}`,
		},
		{
			name:     "self by instance GUID -> dropped",
			payload:  `{"uris":["app-1.example.com"],"host":"10.0.1.24","tls_port":61032,"private_instance_id":"activator-guid"}`,
			wantURIs: nil,
		},
		{
			name:     "self by server_cert_domain_san -> dropped",
			payload:  `{"uris":["app-1.example.com"],"host":"1.2.3.4","tls_port":1,"server_cert_domain_san":"activator-guid"}`,
			wantURIs: nil,
		},
		{
			name:     "self by host+tls_port fallback (no instance id) -> dropped",
			payload:  `{"uris":["app-1.example.com"],"host":"10.0.1.24","tls_port":61032}`,
			wantURIs: nil,
		},
		{
			name:    "malformed JSON -> error",
			payload: `{not json`,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events, err := decodeReadinessEvents([]byte(tc.payload), self)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) != len(tc.wantURIs) {
				t.Fatalf("got %d events %v, want %d %v", len(events), events, len(tc.wantURIs), tc.wantURIs)
			}
			for i, e := range events {
				if e.Action != actionUpsert {
					t.Errorf("event %d action = %q, want %q", i, e.Action, actionUpsert)
				}
				if e.RouteURL != tc.wantURIs[i] {
					t.Errorf("event %d URL = %q, want %q", i, e.RouteURL, tc.wantURIs[i])
				}
			}
		})
	}
}

type guardParker struct {
	calls int
	err   error
}

func (*guardParker) Park(context.Context, string) error     { return nil }
func (p *guardParker) Unpark(context.Context, string) error { p.calls++; return p.err }

func TestReadinessGuardAndUnparkFailure(t *testing.T) {
	registry := NewInMemoryRegistry()
	registry.MarkParked("app", []string{"app.example.com"})
	parker := &guardParker{}
	ready := false
	watcher := NewReadinessWatcher(lagertest.NewTestLogger("guard"), nil, registry, parker, func(context.Context, string) (bool, error) { return ready, nil }).(*readinessWatcher)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waiter := watcher.WaitForReady(ctx, "app")
	watcher.handleUpsert(ctx, "app.example.com")
	if parker.calls != 0 {
		t.Fatal("retiring backend unparked a zero-desired app")
	}
	ready = true
	parker.err = errors.New("registration failure")
	watcher.handleUpsert(ctx, "app.example.com")
	select {
	case err := <-waiter:
		t.Fatalf("failed unpark released a held request: %v", err)
	default:
	}
	parker.err = nil
	watcher.handleUpsert(ctx, "app.example.com")
	select {
	case err := <-waiter:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("manual scale-up readiness did not release waiter")
	}
}

func TestRouteStreamCloseUnblocksNext(t *testing.T) {
	stream := &natsRouteEventStream{events: make(chan RouteEvent), errs: make(chan error), done: make(chan struct{})}
	done := make(chan error, 1)
	go func() { _, err := stream.Next(); done <- err }()
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, nats.ErrConnectionClosed) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream Next remained blocked after shutdown")
	}
}
