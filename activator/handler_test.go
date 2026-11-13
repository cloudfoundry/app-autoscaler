package activator_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator"
	"code.cloudfoundry.org/lager/v3/lagertest"
)

type readyWatcher struct{}

func (readyWatcher) WaitForReady(_ context.Context, _ string) <-chan error {
	ready := make(chan error, 1)
	ready <- nil
	return ready
}

func (readyWatcher) Run(_ context.Context) error { return nil }

type wakeClient struct{ calls int }

func (c *wakeClient) ScaleUp(_ context.Context, _ string) error {
	c.calls++
	return nil
}

type recordingTransport struct{ request *http.Request }

func (t *recordingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.request = r
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("awake"))}, nil
}

func TestWakeForwardsOnlyToRegisteredRoute(t *testing.T) {
	for _, rawURL := range []string{
		"https://app.example.com/orders?item=1",
		"http://169.254.169.254/latest/meta-data",
		"https://app.example.com//169.254.169.254/path",
	} {
		t.Run(rawURL, func(t *testing.T) {
			registry := activator.NewInMemoryRegistry()
			registry.MarkParked("app-id", []string{"app.example.com"})
			transport := &recordingTransport{}
			client := &http.Client{Transport: transport}
			logger := lagertest.NewTestLogger("handler-test")
			handler := activator.NewHandler(logger, activator.NewNoopParker(logger), &wakeClient{}, readyWatcher{}, registry, client, time.Second, "https://api.cf.example.com")
			request := httptest.NewRequest(http.MethodPost, rawURL, strings.NewReader("request body"))
			request.Host = "app.example.com:8080"
			request.Header.Set("X-Forwarded-Proto", "http://169.254.169.254/")
			response := httptest.NewRecorder()
			handler.HandleRouteService(response, request)
			if response.Code != http.StatusOK || transport.request == nil {
				t.Fatalf("request was not forwarded: status %d", response.Code)
			}
			out := transport.request
			if out.URL.Scheme != "https" || out.URL.Host != "api.cf.example.com" || out.Host != "app.example.com" {
				t.Fatalf("untrusted destination: %s", out.URL)
			}
			if out.URL.Path != request.URL.Path || out.URL.RawQuery != request.URL.RawQuery || out.Method != request.Method {
				t.Fatalf("request semantics changed: %s %s", out.Method, out.URL)
			}
			body, err := io.ReadAll(out.Body)
			if err != nil || string(body) != "request body" {
				t.Fatalf("request body changed: %q, %v", body, err)
			}
		})
	}
}

func TestWakeRejectsUnregisteredDestinations(t *testing.T) {
	for _, host := range []string{"169.254.169.254", "app.example.com.attacker.test", "app.example.com@attacker.test"} {
		t.Run(host, func(t *testing.T) {
			registry := activator.NewInMemoryRegistry()
			registry.MarkParked("app-id", []string{"app.example.com"})
			transport := &recordingTransport{}
			engine := &wakeClient{}
			logger := lagertest.NewTestLogger("handler-test")
			handler := activator.NewHandler(logger, activator.NewNoopParker(logger), engine, readyWatcher{}, registry, &http.Client{Transport: transport}, time.Second, "https://api.cf.example.com")
			request := httptest.NewRequest(http.MethodGet, "https://app.example.com/", nil)
			request.Host = host
			response := httptest.NewRecorder()
			handler.HandleRouteService(response, request)
			if response.Code != http.StatusServiceUnavailable || engine.calls != 0 || transport.request != nil {
				t.Fatalf("unregistered host was used: status %d", response.Code)
			}
		})
	}
}

func TestWakeDoesNotResolveRegisteredCustomDomain(t *testing.T) {
	registry := activator.NewInMemoryRegistry()
	registry.MarkParked("app-id", []string{"169.254.169.254"})
	transport := &recordingTransport{}
	logger := lagertest.NewTestLogger("handler-test")
	handler := activator.NewHandler(logger, activator.NewNoopParker(logger), &wakeClient{}, readyWatcher{}, registry, &http.Client{Transport: transport}, time.Second, "https://api.cf.example.com")
	response := httptest.NewRecorder()
	handler.HandleRouteService(response, httptest.NewRequest(http.MethodGet, "https://169.254.169.254/latest/meta-data", nil))
	if transport.request == nil || transport.request.URL.Host != "api.cf.example.com" || transport.request.Host != "169.254.169.254" {
		t.Fatal("registered custom domain influenced the network destination")
	}
}

func TestWakeDoesNotFollowRedirects(t *testing.T) {
	redirected := false
	untrusted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected = true
		w.WriteHeader(http.StatusOK)
	}))
	defer untrusted.Close()
	app := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, untrusted.URL, http.StatusFound)
	}))
	defer app.Close()
	registry := activator.NewInMemoryRegistry()
	registry.MarkParked("app-id", []string{"app.example.com"})
	logger := lagertest.NewTestLogger("handler-test")
	client := app.Client()
	handler := activator.NewHandler(logger, activator.NewNoopParker(logger), &wakeClient{}, readyWatcher{}, registry, client, time.Second, app.URL)
	response := httptest.NewRecorder()
	handler.HandleRouteService(response, httptest.NewRequest(http.MethodGet, "https://app.example.com/", nil))
	if redirected || response.Code != http.StatusFound || response.Header().Get("Location") != untrusted.URL {
		t.Fatalf("redirect was not passed to the caller: status %d, followed %t", response.Code, redirected)
	}
	if client.CheckRedirect != nil {
		t.Fatal("constructor modified the shared HTTP client")
	}
}
