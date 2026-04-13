package app_test

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"

	"acceptance/config"
	. "acceptance/helpers"

	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	cfg      *config.Config
	setup    *workflowhelpers.ReproducibleTestSuiteSetup
	interval int
	client   *http.Client

	instanceName         string
	initialInstanceCount int

	appToScaleName string
	appToScaleGUID string

	metricProducerAppName string

	metricProducerAppGUID string

	// dropletPath holds the pre-built app droplet created once in BeforeSuite.
	// All tests reuse it via CreateTestAppFromDropletByName instead of a full cf push.
	dropletPath string

	// Derived from cfg at suite-setup time; constant for all tests in this suite.
	holdMinutes        int
	maxHeapLimitMb     int
	reportedMiB        float64
	memoryUtilScaleOut int64
)

const componentName = "Application Scale Suite"

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, componentName)
}

var _ = BeforeSuite(func() {
	cfg = config.LoadConfig(config.DefaultTerminateSuite)

	if cfg.GetArtifactsDirectory() != "" {
		helpers.EnableCFTrace(cfg, componentName)
	}

	setup = workflowhelpers.NewTestSuiteSetup(cfg)
	setup.Setup()
	EnableServiceAccess(setup, cfg, setup.GetOrganizationName())
	CheckServiceExists(cfg, setup.TestSpace.SpaceName(), cfg.ServiceName)
	interval = cfg.AggregateInterval
	client = GetHTTPClient(cfg)
	dropletPath = CreateDroplet(cfg)

	const (
		minimalMemoryUsage     = 35
		heapFractionOfLimit    = 0.80
		decimalMBToMiB         = 1_000_000.0 / 1_048_576.0
		memoryUtilSafetyFactor = 0.90
	)
	holdMinutes = cfg.MinLoadDuration()
	maxHeapLimitMb = int(float64(cfg.NodeMemoryLimit)*heapFractionOfLimit) - minimalMemoryUsage
	reportedMiB = float64(maxHeapLimitMb) * decimalMBToMiB
	expectedUtilPct := reportedMiB / float64(cfg.NodeMemoryLimit) * 100
	memoryUtilScaleOut = int64(expectedUtilPct * memoryUtilSafetyFactor)
})

func AppAfterEach() {
	if os.Getenv("SKIP_TEARDOWN") == "true" {
		fmt.Println("Skipping Teardown...")
	} else {
		DebugInfo(cfg, setup, appToScaleName)
		if appToScaleName != "" {
			DeleteService(cfg, instanceName, appToScaleName)
			DeleteTestApp(appToScaleName, cfg.DefaultTimeoutDuration())
		}
		if metricProducerAppName != "" {
			DebugInfo(cfg, setup, metricProducerAppName)
			DeleteService(cfg, instanceName, metricProducerAppName)
			DeleteTestApp(metricProducerAppName, cfg.DefaultTimeoutDuration())
		}
	}
}

var _ = AfterSuite(func() {
	if os.Getenv("SKIP_TEARDOWN") == "true" {
		fmt.Println("Skipping Teardown...")
	} else {
		DisableServiceAccess(cfg, setup)
		if cfg.UseExistingOrganization {
			CleanupInExistingOrg(cfg, setup)
		} else {
			setup.Teardown()
		}
	}
})

func getStartAndEndTime(location *time.Location, offset, duration time.Duration) (time.Time, time.Time) {
	// Since the validation of time could fail if spread over two days and will result in acceptance test failure
	// Need to fix dates in that case.
	startTime := time.Now().In(location).Add(offset)
	if startTime.Day() != startTime.Add(duration).Day() {
		startTime = startTime.Add(duration).Truncate(24 * time.Hour)
	}
	endTime := startTime.Add(duration)
	return startTime, endTime
}

func setupCustomMetricTestApp() {
	policy := GenerateDynamicScaleOutAndInPolicy(1, 2, "test_metric", 500, 500)
	appToScaleName = CreateTestAppFromDroplet(cfg, dropletPath, "labeled-go_app", 1)
	var err error
	appToScaleGUID, err = GetAppGuid(cfg, appToScaleName)
	Expect(err).NotTo(HaveOccurred())
	instanceName = CreatePolicy(cfg, appToScaleName, appToScaleGUID, policy)
	StartApp(appToScaleName, cfg.CfPushTimeoutDuration())
}

func waitForCustomMetricScaling(fn func() (int, error), instances int) {
	GinkgoHelper()
	Eventually(fn).
		WithTimeout(5 * time.Minute).
		WithPolling(5 * time.Second).
		Should(Equal(instances))
}

func DeletePolicyWithAPI(appGUID string) {
	By(fmt.Sprintf("Deleting policy using api for appguid :'%s'", appGUID))
	oauthToken := OauthToken(cfg)
	policyURL := fmt.Sprintf("%s%s", cfg.ASApiEndpoint, strings.ReplaceAll(PolicyPath, "{appId}", appGUID))
	req, err := http.NewRequest(http.MethodDelete, policyURL, nil)
	Expect(err).ShouldNot(HaveOccurred())
	req.Header.Add("Authorization", oauthToken)

	resp, err := client.Do(req)
	Expect(err).ShouldNot(HaveOccurred())
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	Expect(resp.StatusCode).To(Equal(http.StatusOK), "Failed to delete policy '%s'", string(body))
}
