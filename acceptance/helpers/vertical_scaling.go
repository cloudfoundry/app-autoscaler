package helpers

import (
	"acceptance/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	VerticalHistorySucceeded  = 0
	VerticalHistoryFailed     = 1
	VerticalHistoryIgnored    = 2
	VerticalHistoryPending    = 3
	VerticalHistoryUnknown    = 4
	VerticalHistorySuperseded = 5
	VerticalHistoryAborted    = 6
)

type VerticalScalingPolicy struct {
	ScalingPolicy
	MinMemoryMB          int            `json:"min_memory_mb"`
	MaxMemoryMB          int            `json:"max_memory_mb"`
	VerticalScalingRules []*ScalingRule `json:"vertical_scaling_rules"`
}

// GenerateVerticalScalingPolicy fixes the desired instance count and omits
// horizontal rules and scaling_priority. Each phase uses a one-direction rule
// so replacement containers losing synthetic load cannot start an early shrink.
func GenerateVerticalScalingPolicy(c config.VerticalScalingConfig, instances int, threshold int64, operator, adjustment string) (string, error) {
	if c.BreachDurationSeconds < 60 || c.BreachDurationSeconds > 3600 || c.CoolDownSeconds < 60 || c.CoolDownSeconds > 3600 {
		return "", fmt.Errorf("vertical acceptance policy durations must be between 60 and 3600")
	}
	policy := VerticalScalingPolicy{
		ScalingPolicy: ScalingPolicy{InstanceMin: instances, InstanceMax: instances},
		MinMemoryMB:   256,
		MaxMemoryMB:   1024,
		VerticalScalingRules: []*ScalingRule{{
			MetricType: "entitlementutil", Threshold: threshold, Operator: operator, Adjustment: adjustment,
			BreachDurationSeconds: c.BreachDurationSeconds, CoolDownSeconds: c.CoolDownSeconds,
		}},
	}
	data, err := MarshalWithoutHTMLEscape(policy)
	return string(data), err
}

// Optional public fields use pointers where absence must not look like success.
type VerticalHistoryEntry struct {
	AppID            string `json:"app_id"`
	Timestamp        int64  `json:"timestamp"`
	Status           *int   `json:"status"`
	ScalingDimension string `json:"scaling_dimension"`
	OldInstances     int    `json:"old_instances"`
	NewInstances     int    `json:"new_instances"`
	OldMemoryMB      *int   `json:"old_memory_mb"`
	NewMemoryMB      *int   `json:"new_memory_mb"`
	DeploymentGUID   string `json:"deployment_guid"`
	CompletedAt      *int64 `json:"completed_at"`
	Reason           string `json:"reason"`
	Error            string `json:"error"`
	IgnoreReason     string `json:"ignore_reason"`
}

// ContinuousHTTPProbe retains every failure, including those followed by a
// successful request. No redirects, retries, sticky instance header or reused
// connections may hide a route outage. Stop waits for the in-flight request.
type ContinuousHTTPProbe struct {
	mu       sync.Mutex
	requests int
	failure  error
	stop     chan struct{}
	done     chan struct{}
	once     sync.Once
}

func StartContinuousHTTPProbe(client *http.Client, appURL string, interval time.Duration) *ContinuousHTTPProbe {
	probeClient := *client
	probeClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	probe := &ContinuousHTTPProbe{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(probe.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			for _, path := range []string{"/", "/vertical/ready"} {
				select {
				case <-probe.stop:
					return
				default:
				}
				err := probeHTTP(&probeClient, strings.TrimSuffix(appURL, "/")+path)
				probe.mu.Lock()
				probe.requests++
				if err != nil && probe.failure == nil {
					probe.failure = err
				}
				probe.mu.Unlock()
			}
			select {
			case <-probe.stop:
				return
			case <-ticker.C:
			}
		}
	}()
	return probe
}

func (p *ContinuousHTTPProbe) Snapshot() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.requests, p.failure
}

func (p *ContinuousHTTPProbe) Stop() {
	p.once.Do(func() { close(p.stop) })
	<-p.done
}

func probeHTTP(client *http.Client, target string) error {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Close = true
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP availability probe %s: %w", target, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP availability probe %s: status %d, read error %v, body %q", target, resp.StatusCode, err, body)
	}
	var response map[string]string
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("HTTP availability probe %s: invalid application response: %w", target, err)
	}
	if (req.URL.Path == "/" && response["name"] != "test-app") || (req.URL.Path == "/vertical/ready" && response["status"] != "ready") {
		return fmt.Errorf("HTTP availability probe %s: unexpected application response %q", target, body)
	}
	return nil
}
