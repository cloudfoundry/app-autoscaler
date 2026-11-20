package scaletozero_test

import (
	"acceptance/config"
	"acceptance/helpers"
	"net/http"
	"os"
	"testing"

	cfh "github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	cfg                  *config.Config
	setup                *workflowhelpers.ReproducibleTestSuiteSetup
	client               *http.Client
	dropletPath          string
	appToScaleName       string
	appToScaleGUID       string
	instanceName         string
	initialInstanceCount int
)

const componentName = "Scale To Zero Suite"

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, componentName)
}

var _ = BeforeSuite(func() {
	cfg = config.LoadConfig(config.DefaultTerminateSuite)
	if cfg.GetArtifactsDirectory() != "" {
		cfh.EnableCFTrace(cfg, componentName)
	}
	setup = workflowhelpers.NewTestSuiteSetup(cfg)
	setup.Setup()
	helpers.EnableServiceAccess(setup, cfg, setup.GetOrganizationName())
	helpers.CheckServiceExists(cfg, setup.TestSpace.SpaceName(), cfg.ServiceName)
	client = helpers.GetHTTPClient(cfg)
	dropletPath = helpers.CreateDroplet(cfg)
})

var _ = BeforeEach(func() {
	appToScaleName = ""
	appToScaleGUID = ""
	instanceName = ""
	initialInstanceCount = 1
})

var _ = AfterEach(func() {
	if os.Getenv("SKIP_TEARDOWN") == "true" {
		GinkgoWriter.Println("Skipping Teardown...")
		return
	}
	if appToScaleName != "" {
		helpers.DebugInfo(cfg, setup, appToScaleName)
		if instanceName != "" {
			helpers.DeleteService(cfg, instanceName, appToScaleName)
		}
		helpers.DeleteTestApp(appToScaleName, cfg.DefaultTimeoutDuration())
	}
})

var _ = AfterSuite(func() {
	if os.Getenv("SKIP_TEARDOWN") == "true" {
		GinkgoWriter.Println("Skipping Teardown...")
		return
	}
	helpers.DisableServiceAccess(cfg, setup)
	setup.Teardown()
})
