package app_test

import (
	"acceptance"
	"acceptance/helpers"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/generator"
	cfh "github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const verticalInitialMemoryMB = 512

var _ = Describe("AutoScaler vertical scaling", Label(acceptance.LabelVerticalScaling), func() {
	BeforeEach(func() {
		if !cfg.VerticalScaling.Enabled {
			Skip("vertical_scaling_test.enabled is false; explicitly opt in on a disposable test foundation")
		}
		if !cfg.VerticalScaling.DeploymentScalingSupported {
			Skip("foundation support for deployment memory options and Diego HTTP readiness has not been confirmed")
		}
	})

	DescribeTable("grows and shrinks through rolling deployments without route outages", func(load string, threshold int64) {
		v := newVerticalFixture()
		v.setPolicy(threshold, ">=", "+512")
		minutes := max(cfg.MinLoadDuration(), int(cfg.ScaleEventTimeout()/time.Minute)+4)
		By("sustaining real " + load + " load until entitlementutil breaches")
		if load == "memory" {
			v.appRequest(fmt.Sprintf("/memory/320/%d", minutes))
		} else {
			v.appRequest(fmt.Sprintf("/cpu/%d/%d", max(1, cfg.CPUUtilScalingPolicyTest.AppCPUEntitlement), minutes))
		}
		Eventually(func() (float64, error) { return v.latestMetric(loadMetric(load)) }, cfg.ScaleEventTimeout(), 2*time.Second).
			Should(BeNumerically(">=", threshold))
		if load == "cpu" {
			// This case must exercise the CPU arm of max(memoryutil,cpuutil),
			// not an accidental memory breach.
			memory, err := v.latestMetric("memoryutil")
			Expect(err).NotTo(HaveOccurred())
			Expect(memory).To(BeNumerically("<", threshold))
		}
		Eventually(func() (float64, error) { return v.latestMetric("entitlementutil") }, cfg.ScaleEventTimeout(), 2*time.Second).
			Should(BeNumerically(">=", threshold))

		By("observing grow: Pending, overlapping old/new containers, then completed Success")
		grown := v.waitForPending(verticalInitialMemoryMB, 1024)
		v.waitForOverlap(verticalInitialMemoryMB, 1024)
		v.waitForSuccess(grown, 1024)
		grown = v.historyFor(grown)

		// Synthetic load belonged to the old container. The verified replacement
		// is idle; a separate one-direction rule prevents shrink during grow.
		By("attaching an idle-load shrink rule; the completed grow cooldown still applies")
		v.setPolicy(40, "<", "-50%")
		shrunk := v.waitForPending(1024, verticalInitialMemoryMB)
		Expect(shrunk.DeploymentGUID).NotTo(Equal(grown.DeploymentGUID))
		Expect(shrunk.Timestamp).To(BeNumerically(">=", *grown.CompletedAt+int64(time.Duration(cfg.VerticalScaling.CoolDownSeconds)*time.Second)))
		v.waitForOverlap(1024, verticalInitialMemoryMB)
		v.waitForSuccess(shrunk, verticalInitialMemoryMB)
		v.assertAvailable()
	},
		Entry("memory pressure with absolute grow and percentage shrink", "memory", int64(55)),
		Entry("CPU-entitlement pressure with absolute grow and percentage shrink", "cpu", int64(80)),
	)

	Context("isolated rollout safety scenarios", Label(acceptance.LabelVerticalScalingSafety), func() {
		BeforeEach(func() {
			if !cfg.VerticalScaling.SafetyScenariosEnabled {
				Skip("vertical_scaling_test.safety_scenarios_enabled is false; these tests deliberately stall their own app")
			}
		})
		DescribeTable("keeps old instances available and does not claim rollout success", func(undersized bool) {
			v := newVerticalFixture()
			target, adjustment := 1024, "+512"
			if undersized {
				// Reject an insufficient memory budget at readiness, not by causing
				// an OOM or changing any foundation quota. Existing containers retain
				// their startup environment and remain ready.
				v.setEnvironment(map[string]string{"VERTICAL_TEST_MIN_MEMORY_MB": "384"})
				target, adjustment = 256, "-50%"
			} else {
				v.setEnvironment(map[string]string{"VERTICAL_TEST_READINESS_BLOCK": "true"})
			}
			v.setPolicy(100, "<", adjustment)
			pending := v.waitForPending(verticalInitialMemoryMB, target)
			v.waitForOverlap(verticalInitialMemoryMB, target)
			// Includes several 15-second completion checks and another breach
			// evaluation. A stalled deployment must not be automatically retried,
			// canceled, or marked successful after a timeout.
			observation := max(45*time.Second, time.Duration(cfg.VerticalScaling.BreachDurationSeconds+cfg.AggregateInterval+15)*time.Second)
			Consistently(func() bool {
				v.assertAvailable()
				v.assertFixedInstances()
				deployment := v.deployment(pending.DeploymentGUID)
				history := v.historyFor(pending)
				Expect(history.Status).NotTo(BeNil())
				Expect(*history.Status).To(Equal(helpers.VerticalHistoryPending))
				Expect(history.CompletedAt == nil || *history.CompletedAt == 0).To(BeTrue())
				var active struct{ Resources []verticalDeployment `json:"resources"` }
				v.capi(http.MethodGet, "/v3/deployments?app_guids="+v.guid+"&status_values=ACTIVE", nil, &active)
				Expect(active.Resources).To(HaveLen(1))
				Expect(active.Resources[0].GUID).To(Equal(pending.DeploymentGUID))
				return deployment.Status.Value == "ACTIVE"
			}, observation, 2*time.Second).Should(BeTrue())
		},
			Entry("target below the application's declared readiness memory requirement", true),
			Entry("replacement container that never becomes ready", false),
		)
	})
})

type verticalFixture struct {
	name, guid, token, dropletGUID string
	httpClient                    *http.Client
	probe                         *helpers.ContinuousHTTPProbe
}

type verticalProcess struct {
	GUID         string `json:"guid"`
	Type         string `json:"type"`
	Instances    int    `json:"instances"`
	MemoryMB     int    `json:"memory_in_mb"`
	DiskMB       int    `json:"disk_in_mb"`
	HealthCheck  *struct{ Type string `json:"type"` } `json:"health_check"`
	Readiness    *struct{ Type string `json:"type"` } `json:"readiness_health_check"`
}

type verticalStat struct {
	State    string `json:"state"`
	Memory   int64  `json:"mem_quota"`
	Routable *bool  `json:"routable"`
}

type verticalDeployment struct {
	GUID     string `json:"guid"`
	Strategy string `json:"strategy"`
	Status   struct {
		Value  string `json:"value"`
		Reason string `json:"reason"`
	} `json:"status"`
	Options struct {
		MemoryMB    int `json:"memory_in_mb"`
		MaxInFlight int `json:"max_in_flight"`
	} `json:"options"`
	Droplet struct{ GUID string `json:"guid"` } `json:"droplet"`
}

func newVerticalFixture() *verticalFixture {
	GinkgoHelper()
	v := &verticalFixture{name: generator.PrefixedRandomName(cfg.Prefix, "vertical"), token: helpers.OauthToken(cfg), httpClient: helpers.GetHTTPClient(cfg)}
	v.httpClient.Timeout = 5 * time.Second
	v.httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	// Mandatory cleanup is deliberately independent of SKIP_TEARDOWN. These
	// tests may leave an ACTIVE deployment; only the app they created is deleted.
	DeferCleanup(func() { helpers.DeleteTestApp(v.name, cfg.DefaultTimeoutDuration()) })
	appConfig := cfg.Clone()
	appConfig.NodeMemoryLimit = verticalInitialMemoryMB
	Expect(helpers.CreateTestAppFromDropletByName(appConfig, dropletPath, v.name, 1)).To(Succeed())
	var err error
	v.guid, err = helpers.GetAppGuid(cfg, v.name)
	Expect(err).NotTo(HaveOccurred())
	process := v.currentProcess()
	if process.Readiness == nil {
		Skip("CAPI does not expose readiness health checks")
	}
	v.setEnvironment(map[string]string{"VERTICAL_TEST_ENABLED": "true", "VERTICAL_TEST_READINESS_DELAY_SECONDS": "45"})
	health := map[string]any{"health_check": map[string]any{"type": "http", "data": map[string]any{
		"endpoint": "/vertical/ready", "timeout": 120, "invocation_timeout": 2,
	}}, "readiness_health_check": map[string]any{"type": "http", "data": map[string]any{
		"endpoint": "/vertical/ready", "invocation_timeout": 2, "interval": 1,
	}}}
	v.capi(http.MethodPatch, "/v3/processes/"+process.GUID, health, &process)
	Expect(process.HealthCheck.Type).To(Equal("http"))
	Expect(process.Readiness.Type).To(Equal("http"))
	service := helpers.CreateService(cfg)
	DeferCleanup(func() { helpers.DeleteService(cfg, service, v.name) })
	helpers.BindServiceToApp(cfg, v.name, service)
	helpers.StartApp(v.name, cfg.CfPushTimeoutDuration())
	stats := v.stats(v.currentProcess().GUID)
	Expect(stats).To(HaveLen(1))
	if stats[0].Routable == nil {
		Skip("Diego does not report readiness/routability; no unsupported fallback is tested")
	}
	Expect(*stats[0].Routable).To(BeTrue())
	var droplet struct{ GUID string `json:"guid"` }
	v.capi(http.MethodGet, "/v3/apps/"+v.guid+"/droplets/current", nil, &droplet)
	v.dropletGUID = droplet.GUID
	probeClient := helpers.GetHTTPClient(cfg)
	probeClient.Timeout = 2 * time.Second
	probeClient.CheckRedirect = v.httpClient.CheckRedirect
	v.probe = helpers.StartContinuousHTTPProbe(probeClient, cfh.AppUri(v.name, "", cfg), 100*time.Millisecond)
	DeferCleanup(func() {
		v.probe.Stop()
		v.assertAvailable()
	})
	Eventually(func() int { n, _ := v.probe.Snapshot(); return n }, 10*time.Second, 100*time.Millisecond).Should(BeNumerically(">=", 4))
	v.assertAvailable()
	return v
}

func (v *verticalFixture) setEnvironment(values map[string]string) {
	v.capi(http.MethodPatch, "/v3/apps/"+v.guid+"/environment_variables", map[string]any{"var": values}, nil)
}

func (v *verticalFixture) setPolicy(threshold int64, operator, adjustment string) {
	GinkgoHelper()
	policy, err := helpers.GenerateVerticalScalingPolicy(cfg.VerticalScaling, 1, threshold, operator, adjustment)
	Expect(err).NotTo(HaveOccurred())
	var response helpers.VerticalScalingPolicy
	v.request(http.MethodPut, cfg.ASApiEndpoint+"/v1/apps/"+v.guid+"/policy", json.RawMessage(policy), &response)
	v.request(http.MethodGet, cfg.ASApiEndpoint+"/v1/apps/"+v.guid+"/policy", nil, &response)
	Expect(response.MinMemoryMB).To(Equal(256))
	Expect(response.MaxMemoryMB).To(Equal(1024))
	Expect(response.VerticalScalingRules).To(HaveLen(1))
	Expect(response.VerticalScalingRules[0].MetricType).To(Equal("entitlementutil"))
	Expect(response.VerticalScalingRules[0].Adjustment).To(Equal(adjustment))
}

func (v *verticalFixture) appRequest(path string) {
	v.request(http.MethodGet, cfh.AppUri(v.name, path, cfg), nil, nil)
}

func (v *verticalFixture) request(method, target string, body, output any) {
	GinkgoHelper()
	var input io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		Expect(err).NotTo(HaveOccurred())
		input = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, target, input)
	Expect(err).NotTo(HaveOccurred())
	// App routes receive no credentials; CAPI/public Autoscaler requests do.
	if strings.HasPrefix(target, cfg.ASApiEndpoint+"/") || strings.HasPrefix(target, v.capiBase()+"/") {
		req.Header.Set("Authorization", v.token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.httpClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(BeNumerically(">=", 200), "%s %s: %s", method, target, data)
	Expect(resp.StatusCode).To(BeNumerically("<", 300), "%s %s: %s", method, target, data)
	if output != nil {
		Expect(json.Unmarshal(data, output)).To(Succeed())
	}
}

func (v *verticalFixture) capiBase() string {
	base := strings.TrimSuffix(cfg.GetApiEndpoint(), "/")
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = cfg.Protocol() + base
	}
	return base
}

func (v *verticalFixture) capi(method, path string, body, output any) {
	v.request(method, v.capiBase()+path, body, output)
}

func (v *verticalFixture) currentProcess() verticalProcess {
	var process verticalProcess
	v.capi(http.MethodGet, "/v3/apps/"+v.guid+"/processes/web", nil, &process)
	return process
}

func (v *verticalFixture) stats(processGUID string) []verticalStat {
	var result struct{ Resources []verticalStat `json:"resources"` }
	v.capi(http.MethodGet, "/v3/processes/"+processGUID+"/stats", nil, &result)
	return result.Resources
}

func (v *verticalFixture) deployment(guid string) verticalDeployment {
	var deployment verticalDeployment
	v.capi(http.MethodGet, "/v3/deployments/"+guid, nil, &deployment)
	return deployment
}

func (v *verticalFixture) histories() []helpers.VerticalHistoryEntry {
	var entries []helpers.VerticalHistoryEntry
	for page, pages := 1, 1; page <= pages; page++ {
		var result struct {
			TotalPages int `json:"total_pages"`
			Resources []helpers.VerticalHistoryEntry `json:"resources"`
		}
		v.request(http.MethodGet, fmt.Sprintf("%s/v1/apps/%s/scaling_histories?results-per-page=100&order-direction=desc&page=%d", cfg.ASApiEndpoint, v.guid, page), nil, &result)
		entries = append(entries, result.Resources...)
		pages = result.TotalPages
	}
	return entries
}

func (v *verticalFixture) historyFor(original helpers.VerticalHistoryEntry) helpers.VerticalHistoryEntry {
	for _, entry := range v.histories() {
		if entry.DeploymentGUID == original.DeploymentGUID && entry.Timestamp == original.Timestamp {
			return entry
		}
	}
	Fail("correlated vertical scaling history disappeared")
	return helpers.VerticalHistoryEntry{}
}

func (v *verticalFixture) waitForPending(oldMemory, newMemory int) helpers.VerticalHistoryEntry {
	GinkgoHelper()
	var pending helpers.VerticalHistoryEntry
	Eventually(func() bool {
		v.assertAvailable()
		v.assertFixedInstances()
		for _, entry := range v.histories() {
			if entry.ScalingDimension != "vertical" || entry.OldMemoryMB == nil || entry.NewMemoryMB == nil ||
				*entry.OldMemoryMB != oldMemory || *entry.NewMemoryMB != newMemory {
				continue
			}
			Expect(entry.Status).NotTo(BeNil())
			if *entry.Status == helpers.VerticalHistoryIgnored {
				continue
			}
			Expect(*entry.Status).To(Equal(helpers.VerticalHistoryPending), "expected an observable Pending request, not completed/failed history: %+v", entry)
			if entry.DeploymentGUID == "" {
				continue
			}
			Expect(entry.AppID).To(Equal(v.guid))
			Expect(entry.CompletedAt == nil || *entry.CompletedAt == 0).To(BeTrue())
			deployment := v.deployment(entry.DeploymentGUID)
			Expect(deployment.Strategy).To(Equal("rolling"))
			Expect(deployment.Status.Value).To(Equal("ACTIVE"))
			Expect(deployment.Options.MaxInFlight).To(Equal(1))
			Expect(deployment.Options.MemoryMB).To(Equal(newMemory))
			Expect(deployment.Droplet.GUID).To(Equal(v.dropletGUID), "vertical scaling must reuse the current droplet")
			pending = entry
			return true
		}
		return false
	}, cfg.ScaleEventTimeout()+time.Duration(cfg.VerticalScaling.CoolDownSeconds)*time.Second, time.Second).Should(BeTrue())
	return pending
}

func (v *verticalFixture) waitForOverlap(oldMemory, newMemory int) {
	GinkgoHelper()
	Eventually(func() bool {
		v.assertAvailable()
		v.assertFixedInstances()
		var processes struct{ Resources []verticalProcess `json:"resources"` }
		v.capi(http.MethodGet, "/v3/apps/"+v.guid+"/processes?per_page=100", nil, &processes)
		oldReady, newUnready := false, false
		for _, process := range processes.Resources {
			if process.Type != "web" {
				continue
			}
			for _, stat := range v.stats(process.GUID) {
				if stat.Memory == int64(oldMemory)*1024*1024 && stat.State == "RUNNING" && stat.Routable != nil && *stat.Routable {
					oldReady = true
				}
				if stat.Memory == int64(newMemory)*1024*1024 && (stat.State == "STARTING" || (stat.Routable != nil && !*stat.Routable)) {
					newUnready = true
				}
			}
		}
		return oldReady && newUnready
	}, cfg.CfPushTimeoutDuration(), time.Second).Should(BeTrue(), "must observe an old routable container alongside an unready replacement with the target memory quota")
}

func (v *verticalFixture) waitForSuccess(pending helpers.VerticalHistoryEntry, memory int) {
	GinkgoHelper()
	Eventually(func() bool {
		v.assertAvailable()
		v.assertFixedInstances()
		history := v.historyFor(pending)
		Expect(history.Status).NotTo(BeNil())
		if *history.Status == helpers.VerticalHistoryPending {
			Expect(history.CompletedAt == nil || *history.CompletedAt == 0).To(BeTrue())
			return false
		}
		Expect(*history.Status).To(Equal(helpers.VerticalHistorySucceeded))
		// Read CAPI after history to avoid declaring a race at the precise
		// completion boundary (ACTIVE response followed by completed history).
		deployment := v.deployment(pending.DeploymentGUID)
		Expect(deployment.Status.Value).To(Equal("FINALIZED"))
		Expect(deployment.Status.Reason).To(Equal("DEPLOYED"))
		Expect(history.CompletedAt).NotTo(BeNil())
		Expect(*history.CompletedAt).To(BeNumerically(">", history.Timestamp))
		return true
	}, cfg.CfPushTimeoutDuration()+30*time.Second, time.Second).Should(BeTrue())
	Eventually(func() bool {
		v.assertAvailable()
		process := v.currentProcess()
		Expect(process.Instances).To(Equal(1))
		Expect(process.MemoryMB).To(Equal(memory))
		Expect(process.DiskMB).To(Equal(32), "vertical scaling must not change disk")
		stats := v.stats(process.GUID)
		return len(stats) == 1 && stats[0].State == "RUNNING" && stats[0].Memory == int64(memory)*1024*1024 && stats[0].Routable != nil && *stats[0].Routable
	}, cfg.DefaultTimeoutDuration(), time.Second).Should(BeTrue())
}

func (v *verticalFixture) assertFixedInstances() {
	GinkgoHelper()
	Expect(v.currentProcess().Instances).To(Equal(1), "desired web instance count must remain unchanged; temporary rolling overlap is allowed")
}

func (v *verticalFixture) assertAvailable() {
	GinkgoHelper()
	n, err := v.probe.Snapshot()
	Expect(err).NotTo(HaveOccurred(), "HTTP route outage after %d continuous requests", n)
	Expect(n).To(BeNumerically(">", 0))
}

func loadMetric(load string) string {
	if load == "cpu" {
		return "cpuutil"
	}
	return "memoryutil"
}

func (v *verticalFixture) latestMetric(metric string) (float64, error) {
	var result struct{ Resources []struct{ Value string `json:"value"` } `json:"resources"` }
	v.request(http.MethodGet, cfg.ASApiEndpoint+"/v1/apps/"+v.guid+"/aggregated_metric_histories/"+metric+"?order-direction=desc&results-per-page=1", nil, &result)
	if len(result.Resources) == 0 {
		return 0, fmt.Errorf("no %s metric yet", metric)
	}
	return strconv.ParseFloat(result.Resources[0].Value, 64)
}
