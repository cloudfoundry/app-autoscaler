package activator_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3/lagertest"
)

func TestWakeClientUsesConfiguredEngine(t *testing.T) {
	const appID = "550e8400-e29b-41d4-a716-446655440000"
	var received models.Trigger
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/apps/"+appID+"/scale" {
			t.Errorf("unexpected scale request: %s %s", r.Method, r.URL)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer engine.Close()
	client := activator.NewScalingEngineClient(lagertest.NewTestLogger("engine-test"), engine.Client(), engine.URL)
	if err := client.ScaleUp(context.Background(), appID); err != nil {
		t.Fatal(err)
	}
	if received.AppId != appID || received.Adjustment != "+1" || !received.BypassCooldown {
		t.Fatalf("unexpected wake trigger: %+v", received)
	}
}

func TestWakeClientRejectsInvalidAppIDs(t *testing.T) {
	transport := &recordingTransport{}
	client := activator.NewScalingEngineClient(lagertest.NewTestLogger("engine-test"), &http.Client{Transport: transport}, "https://engine.example.com")
	for _, appID := range []string{"", "../../other", "//169.254.169.254", "app-id?redirect=evil"} {
		if err := client.ScaleUp(context.Background(), appID); err == nil {
			t.Errorf("accepted invalid app ID %q", appID)
		}
	}
	if transport.request != nil {
		t.Fatal("sent a request for an invalid app ID")
	}
}

func TestWakeClientDoesNotFollowRedirects(t *testing.T) {
	redirected := false
	untrusted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected = true
		w.WriteHeader(http.StatusOK)
	}))
	defer untrusted.Close()
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, untrusted.URL, http.StatusTemporaryRedirect)
	}))
	defer engine.Close()
	client := activator.NewScalingEngineClient(lagertest.NewTestLogger("engine-test"), engine.Client(), engine.URL)
	if err := client.ScaleUp(context.Background(), "550e8400-e29b-41d4-a716-446655440000"); err == nil || redirected {
		t.Fatalf("redirect should fail without following it: %v, followed %t", err, redirected)
	}
}

func TestWakeClientRejectsInvalidEngineURLs(t *testing.T) {
	for _, engineURL := range []string{"", "/relative", "ftp://engine.example.com", "https://user:password@engine.example.com", "https://%"} {
		t.Run(engineURL, func(t *testing.T) {
			transport := &recordingTransport{}
			client := activator.NewScalingEngineClient(lagertest.NewTestLogger("engine-test"), &http.Client{Transport: transport}, engineURL)
			if err := client.ScaleUp(context.Background(), "550e8400-e29b-41d4-a716-446655440000"); err == nil || transport.request != nil {
				t.Fatalf("invalid engine URL was used: %q", engineURL)
			}
		})
	}
}
