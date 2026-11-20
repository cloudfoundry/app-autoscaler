package scaletozero_test

import (
	"acceptance"
	"acceptance/helpers"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	cfh "github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

// Scale-to-zero / scale-from-zero acceptance test.
//
// Run just this test against a deployed autoscaler with:
//
//	SUITES=scaletozero make mta-acceptance-tests
//
// or by label: --label-filter='scaletozero'.
//
// Requires the activator, appstateobserver and routing NATS connectivity.
var _ = Describe("AutoScaler scaletozero", func() {
	var policy string

	JustBeforeEach(func() {
		appToScaleName = helpers.CreateTestAppFromDroplet(cfg, dropletPath, "scaletozero", initialInstanceCount)
		var err error
		appToScaleGUID, err = helpers.GetAppGuid(cfg, appToScaleName)
		Expect(err).NotTo(HaveOccurred())
		helpers.StartApp(appToScaleName, cfg.CfPushTimeoutDuration())
		instanceName, err = helpers.CreatePolicyWithErr(cfg, appToScaleName, policy)
		Expect(err).NotTo(HaveOccurred(), "the scale-to-zero policy must be bound before scaling")
	})

	Context("when scaling to and from zero by cpu", func() {
		BeforeEach(func() {
			// instance_min_count 0 enables scale-to-zero and is only accepted by
			// the v0.1 policy schema, so the policy must carry schema-version
			// "0.1" (a policy without it defaults to the legacy schema, which
			// still enforces a minimum of 1). A low-cpu scale-in rule drives the
			// app down to zero when idle.
			scaleInThreshold := int64(float64(cfg.CPUUpperThreshold) * 0.2)
			scaleOutThreshold := int64(float64(cfg.CPUUpperThreshold) * 0.4)
			policy = fmt.Sprintf(`{
	"schema-version": "0.1",
	"instance_min_count": 0,
	"instance_max_count": 2,
	"scaling_rules": [
		{ "metric_type": "cpu", "threshold": %d, "operator": ">=", "adjustment": "+1" },
		{ "metric_type": "cpu", "threshold": %d, "operator": "<", "adjustment": "-1" }
	]
}`, scaleOutThreshold, scaleInThreshold)
			initialInstanceCount = 1
		})

		It("scaletozero: scales down to zero when idle and back up on request", func() {
			By("scaling to zero once cpu is idle")
			helpers.WaitForNInstancesRunning(appToScaleGUID, 0, cfg.ScaleEventTimeout(),
				"expected the app to scale to zero instances when idle")

			By("waking the app from zero via an HTTP request to its route")
			// The request is forwarded by Gorouter to the activator (route
			// service), which scales the app back up and forwards the request.
			statusCode := cfh.CurlAppWithStatusCode(cfg, appToScaleName, "/health")
			// The wake path may return 503 + Retry-After on the very first hit
			// while the app is still starting; the app must become reachable.
			GinkgoWriter.Printf("initial wake status: %s\n", statusCode)

			By("scaling back up to at least one instance")
			helpers.WaitForNInstancesRunning(appToScaleGUID, 1, cfg.ScaleEventTimeout(),
				"expected the app to scale from zero to one instance after a request")

			By("serving the request once awake")
			Eventually(func() int {
				return getHealthStatus(appToScaleName)
			}).WithTimeout(cfg.ScaleEventTimeout()).WithPolling(5*time.Second).Should(Equal(http.StatusOK),
				"expected the woken app to serve /health with 200")
		})
	})

	Context("when instance counts are changed manually", func() {
		BeforeEach(func() {
			// The schema requires a rule or schedule. This metric is never
			// emitted by the test app, so only the CLI and a wake request can
			// change its instance count.
			policy = `{"schema-version":"0.1","instance_min_count":0,"instance_max_count":3,
				"scaling_rules":[{"metric_type":"manual_scaletozero_unused","threshold":1,"operator":">","adjustment":"+1"}]}`
			initialInstanceCount = 1
		})

		It("scaletozero: parks a manually zero-scaled app and wakes it on request", func() {
			By("manually scaling the web process to zero")
			Expect(cf.Cf("scale", appToScaleName, "-i", "0").Wait(cfg.DefaultTimeoutDuration())).To(Exit(0))
			helpers.WaitForNInstancesRunning(appToScaleGUID, 0, cfg.ScaleEventTimeout())
			waitForActivatorEvent("cf-parker.parked")

			By("waking the manually parked app through its unchanged route")
			Eventually(func() int { return getHealthStatus(appToScaleName) }).
				WithTimeout(cfg.ScaleEventTimeout()).WithPolling(5 * time.Second).Should(Equal(http.StatusOK))
			helpers.WaitForNInstancesRunning(appToScaleGUID, 1, cfg.ScaleEventTimeout())
			waitForActivatorEvent("cf-parker.unparked")
		})

		It("scaletozero: unparks after manual scale-back-up without a wake request", func() {
			By("manually scaling to zero and waiting for parking without HTTP traffic")
			Expect(cf.Cf("scale", appToScaleName, "-i", "0").Wait(cfg.DefaultTimeoutDuration())).To(Exit(0))
			helpers.WaitForNInstancesRunning(appToScaleGUID, 0, cfg.ScaleEventTimeout())
			waitForActivatorEvent("cf-parker.parked")

			By("manually scaling back up to two instances")
			Expect(cf.Cf("scale", appToScaleName, "-i", "2").Wait(cfg.DefaultTimeoutDuration())).To(Exit(0))
			helpers.WaitForNInstancesRunning(appToScaleGUID, 2, cfg.ScaleEventTimeout())
			// This must happen before any HTTP request: the routing readiness
			// event, rather than request-driven activation, causes un-parking.
			waitForActivatorEvent("cf-parker.unparked")
			Eventually(func() int { return getHealthStatus(appToScaleName) }).
				WithTimeout(cfg.ScaleEventTimeout()).WithPolling(5 * time.Second).Should(Equal(http.StatusOK))
		})
	})
}, Label(acceptance.LabelScaleToZero))

// The parking registry is deliberately an authenticated internal surface.
// Foundation-admin access to the activator's existing structured logs lets the
// acceptance test prove parking/unparking without issuing a wake request or
// adding an unauthenticated test-only application endpoint.
func waitForActivatorEvent(event string) {
	GinkgoHelper()
	workflowhelpers.AsUser(setup.AdminUserContext(), cfg.DefaultTimeoutDuration(), func() {
		name := cfg.ActivatorAppName
		if name == "" {
			name = "activator"
		}
		apps := cf.Cf("curl", "/v3/apps?names="+url.QueryEscape(name)+"&include=space.organization").Wait(cfg.DefaultTimeoutDuration())
		Expect(apps).To(Exit(0))
		var response struct {
			Included struct {
				Spaces []struct {
					Name string `json:"name"`
				} `json:"spaces"`
				Organizations []struct {
					Name string `json:"name"`
				} `json:"organizations"`
			} `json:"included"`
		}
		Expect(json.Unmarshal(apps.Out.Contents(), &response)).To(Succeed())
		Expect(response.Included.Spaces).To(HaveLen(1))
		Expect(response.Included.Organizations).To(HaveLen(1))
		Expect(cf.Cf("target", "-o", response.Included.Organizations[0].Name, "-s", response.Included.Spaces[0].Name).
			Wait(cfg.DefaultTimeoutDuration())).To(Exit(0))
		Eventually(func() bool {
			logs := cf.Cf("logs", name, "--recent").Wait(cfg.DefaultTimeoutDuration())
			if logs.ExitCode() != 0 {
				return false
			}
			for _, line := range strings.Split(string(logs.Out.Contents()), "\n") {
				if strings.Contains(line, event) && strings.Contains(line, appToScaleGUID) {
					return true
				}
			}
			return false
		}).WithTimeout(cfg.ScaleEventTimeout()).WithPolling(5*time.Second).Should(BeTrue(),
			"expected activator event %s for app %s", event, appToScaleGUID)
	})
}

// getHealthStatus GETs the app's /health route and returns the HTTP status
// code, retrying the wake path until the app is reachable.
func getHealthStatus(appName string) int {
	uri := cfh.AppUri(appName, "/health", cfg)
	resp, err := client.Get(uri)
	if err != nil {
		return 0
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}
