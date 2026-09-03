package activator

import (
	"testing"
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
