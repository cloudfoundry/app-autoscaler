package cf_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3/lagertest"
)

func TestUserAgentInHTTPRequests(t *testing.T) {
	var capturedUserAgent string

	// Simple mock server to capture User-Agent header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")

		switch r.URL.Path {
		case "/":
			// CF API root endpoint
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"links": {"uaa": {"href": "http://` + r.Host + `"}}}`))
		case "/oauth/token":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"test-token","expires_in":3600}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	logger := lagertest.NewTestLogger("test")
	clock := fakeclock.NewFakeClock(time.Now())

	conf := &cf.Config{
		API:      server.URL,
		ClientID: "test-client",
		Secret:   "test-secret",
		ClientConfig: cf.ClientConfig{
			SkipSSLValidation: true,
		},
	}

	client := cf.NewCFClient(conf, logger, clock)

	// Test authentication request includes user agent
	err := client.Login()
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if capturedUserAgent == "" {
		t.Error("User-Agent header was not set")
	}

	if !containsString(capturedUserAgent, "app-autoscaler/") {
		t.Errorf("User-Agent does not contain 'app-autoscaler/': %s", capturedUserAgent)
	}

	if !containsString(capturedUserAgent, "Go/") {
		t.Errorf("User-Agent does not contain Go version: %s", capturedUserAgent)
	}

	t.Logf("Captured User-Agent: %s", capturedUserAgent)
}

func TestUserAgentInAPIRequests(t *testing.T) {
	var capturedUserAgent string

	// Mock server to capture User-Agent header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`{"links": {"uaa": {"href": "http://` + r.Host + `"}}}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	logger := lagertest.NewTestLogger("test")
	clock := fakeclock.NewFakeClock(time.Now())

	conf := &cf.Config{
		API:      server.URL,
		ClientID: "test-client",
		Secret:   "test-secret",
		ClientConfig: cf.ClientConfig{
			SkipSSLValidation: true,
		},
	}

	client := cf.NewCFClient(conf, logger, clock)

	// Test API request includes user agent
	ctx := context.Background()
	ctxClient := client.GetCtxClient()
	_, err := ctxClient.GetEndpoints(ctx)
	if err != nil {
		t.Fatalf("GetEndpoints failed: %v", err)
	}

	if capturedUserAgent == "" {
		t.Error("User-Agent header was not set")
	}

	if !containsString(capturedUserAgent, "app-autoscaler/") {
		t.Errorf("User-Agent does not contain 'app-autoscaler/': %s", capturedUserAgent)
	}

	t.Logf("Captured User-Agent: %s", capturedUserAgent)
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && contains(s, substr)))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
