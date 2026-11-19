package helpers_test

import (
	"acceptance/config"
	"acceptance/helpers"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestVerticalPolicyUsesEntitlementAndLeavesInstancesFixed(t *testing.T) {
	for _, adjustment := range []string{"+512", "-50%"} {
		policy, err := helpers.GenerateVerticalScalingPolicy(config.VerticalScalingConfig{
			BreachDurationSeconds: 60, CoolDownSeconds: 120,
		}, 1, 80, ">=", adjustment)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(policy), &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded["instance_min_count"] != float64(1) || decoded["instance_max_count"] != float64(1) ||
			decoded["min_memory_mb"] != float64(256) || decoded["max_memory_mb"] != float64(1024) {
			t.Fatalf("unexpected limits: %s", policy)
		}
		if _, found := decoded["scaling_rules"]; found {
			t.Fatalf("unexpected horizontal rules: %s", policy)
		}
		if _, found := decoded["scaling_priority"]; found {
			t.Fatalf("unexpected priority without horizontal rules: %s", policy)
		}
		rules := decoded["vertical_scaling_rules"].([]any)
		rule := rules[0].(map[string]any)
		if rule["metric_type"] != "entitlementutil" || rule["threshold"] != float64(80) || rule["adjustment"] != adjustment ||
			rule["breach_duration_secs"] != float64(60) || rule["cool_down_secs"] != float64(120) {
			t.Fatalf("unexpected vertical rule: %s", policy)
		}
	}
}

func TestVerticalPolicyRejectsShortGlobalDurations(t *testing.T) {
	_, err := helpers.GenerateVerticalScalingPolicy(config.VerticalScalingConfig{
		BreachDurationSeconds: 1, CoolDownSeconds: 60,
	}, 1, 80, ">=", "+512")
	if err == nil {
		t.Fatal("expected invalid duration to be rejected")
	}
}

func TestVerticalHistoryDistinguishesPendingFromSuccessAndMissingFields(t *testing.T) {
	for _, status := range []int{0, 1, 2, 3, 4, 5, 6} {
		entry := helpers.VerticalHistoryEntry{Status: &status}
		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		var decoded helpers.VerticalHistoryEntry
		if err := json.Unmarshal(data, &decoded); err != nil || decoded.Status == nil || *decoded.Status != status {
			t.Fatalf("history status %d did not round trip: %s, %v", status, data, err)
		}
	}
	var missing helpers.VerticalHistoryEntry
	if err := json.Unmarshal([]byte(`{"app_id":"test"}`), &missing); err != nil {
		t.Fatal(err)
	}
	if missing.Status != nil || missing.OldMemoryMB != nil || missing.NewMemoryMB != nil || missing.CompletedAt != nil {
		t.Fatal("missing optional history fields must remain distinguishable")
	}
}

func TestContinuousProbeRemembersBriefRouteOutages(t *testing.T) {
	var fail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/vertical/ready" {
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		} else {
			_, _ = w.Write([]byte(`{"name":"test-app"}`))
		}
	}))
	defer server.Close()
	probe := helpers.StartContinuousHTTPProbe(server.Client(), server.URL, time.Millisecond)
	defer probe.Stop()
	waitForProbe(t, probe, func(n int, err error) bool { return n >= 4 && err == nil })
	fail.Store(true)
	waitForProbe(t, probe, func(_ int, err error) bool { return err != nil })
	fail.Store(false)
	probe.Stop()
	n, err := probe.Snapshot()
	if n < 4 || err == nil {
		t.Fatalf("probe lost an outage: requests %d, error %v", n, err)
	}
}

func TestContinuousProbeRejectsInvalidSuccessfulBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"not-ready"}`))
	}))
	defer server.Close()
	probe := helpers.StartContinuousHTTPProbe(server.Client(), server.URL, time.Millisecond)
	defer probe.Stop()
	waitForProbe(t, probe, func(_ int, err error) bool { return err != nil })
}

func waitForProbe(t *testing.T, probe *helpers.ContinuousHTTPProbe, condition func(int, error) bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition(probe.Snapshot()) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("HTTP probe did not reach expected state")
}
