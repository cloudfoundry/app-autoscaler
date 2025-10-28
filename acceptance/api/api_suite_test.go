package api_test

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"acceptance/config"
	. "acceptance/helpers"

	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	HealthPath           = "/health"
	AggregatedMetricPath = "/v1/apps/{appId}/aggregated_metric_histories/{metric_type}"
	HistoryPath          = "/v1/apps/{appId}/scaling_histories"
)

var (
	cfg                 *config.Config
	setup               *workflowhelpers.ReproducibleTestSuiteSetup
	otherSetup          *workflowhelpers.ReproducibleTestSuiteSetup
	appName             string
	appGUID             string
	instanceName        string
	healthURL           string
	policyURL           string
	metricURL           string
	aggregatedMetricURL string
	historyURL          string
	client              *http.Client
	err                 error
)

const componentName = "Public API Suite"

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, componentName)
}

var _ = BeforeSuite(func() {
	cfg = config.LoadConfig(config.DefaultTerminateSuite)
	componentName := "Public API Suite"

	if cfg.GetArtifactsDirectory() != "" {
		helpers.EnableCFTrace(cfg, componentName)
	}

	otherConfig := *cfg
	otherConfig.NamePrefix = otherConfig.NamePrefix + "_other"

	By("Setup test environment")
	setup = workflowhelpers.NewTestSuiteSetup(cfg)
	otherSetup = workflowhelpers.NewTestSuiteSetup(&otherConfig)

	otherSetup.Setup()
	setup.Setup()

	EnableServiceAccess(setup, cfg, setup.GetOrganizationName())

	appName = CreateTestApp(cfg, "apitest", 1)
	appGUID, err = GetAppGuid(cfg, appName)
	Expect(err).NotTo(HaveOccurred())

	By("Creating test service")
	instanceName = CreateService(cfg)
	BindServiceToApp(cfg, appName, instanceName)
	StartApp(appName, cfg.CfPushTimeoutDuration())

	//nolint:gosec // #nosec G402 -- due to https://github.com/securego/gosec/issues/1105
	client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableCompression:  true,
			DisableKeepAlives:   true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipSSLValidation,
			},
		},
		Timeout: 30 * time.Second,
	}

	healthURL = fmt.Sprintf("%s%s", cfg.ASApiEndpoint, HealthPath)
	policyURL = fmt.Sprintf("%s%s", cfg.ASApiEndpoint, strings.ReplaceAll(PolicyPath, "{appId}", appGUID))
	metricURL = fmt.Sprintf("%s%s", cfg.ASApiEndpoint, strings.ReplaceAll(metricURL, "{appId}", appGUID))
	aggregatedMetricURL = strings.ReplaceAll(AggregatedMetricPath, "{metric_type}", "memoryused")
	aggregatedMetricURL = fmt.Sprintf("%s%s", cfg.ASApiEndpoint, strings.ReplaceAll(aggregatedMetricURL, "{appId}", appGUID))
	historyURL = fmt.Sprintf("%s%s", cfg.ASApiEndpoint, strings.ReplaceAll(HistoryPath, "{appId}", appGUID))
})

var _ = AfterSuite(func() {
	if os.Getenv("SKIP_TEARDOWN") == "true" {
		fmt.Println("Skipping Teardown...")
	} else {
		DeleteService(cfg, instanceName, appName)
		DeleteTestApp(appName, cfg.DefaultTimeoutDuration())
		DisableServiceAccess(cfg, setup)
		otherSetup.Teardown()
		setup.Teardown()
	}
})
