package app_test

import (
	"acceptance"
	"acceptance/helpers"
	"fmt"
	"net/http"
	"os"
	"time"

	cfh "github.com/cloudfoundry/cf-test-helpers/v2/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Scale-to-zero / scale-from-zero acceptance test.
//
// Run just this test against a deployed autoscaler with:
//
//	SUITES=app GINKGO_OPTS='--focus=scaletozero' make acceptance-tests
//
// or by label: --label-filter='scaletozero'.
//
// It requires the activator to be deployed and the app's route bound to the
// activator route-service on scale-to-zero (see docs/design/scale-to-zero.md).
var _ = Describe("AutoScaler scaletozero", func() {
	var policy string

	JustBeforeEach(func() {
		appToScaleName = helpers.CreateTestAppFromDroplet(cfg, dropletPath, "scaletozero", initialInstanceCount)
		var err error
		appToScaleGUID, err = helpers.GetAppGuid(cfg, appToScaleName)
		Expect(err).NotTo(HaveOccurred())
		helpers.StartApp(appToScaleName, cfg.CfPushTimeoutDuration())
		instanceName = helpers.CreatePolicy(cfg, appToScaleName, appToScaleGUID, policy)
	})

	AfterEach(func() {
		if os.Getenv("SKIP_TEARDOWN") == "true" {
			fmt.Println("Skipping Teardown...")
			return
		}
		AppAfterEach()
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
}, Label(acceptance.LabelScaleToZero))

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
