package pre_upgrade_test

import (
	"acceptance/config"
	. "acceptance/helpers"
	"testing"

	cth "github.com/cloudfoundry/cf-test-helpers/v2/helpers"

	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	cfg   *config.Config
	setup *workflowhelpers.ReproducibleTestSuiteSetup
)

const componentName = "Pre Upgrade Test Suite"

func TestSetup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, componentName)
}

var _ = BeforeSuite(func() {
	cfg = config.LoadConfig(config.DefaultTerminateSuite)
	if cfg.GetArtifactsDirectory() != "" {
		cth.EnableCFTrace(cfg, componentName)
	}

	setup = workflowhelpers.NewTestSuiteSetup(cfg)

	GinkgoWriter.Println("Clearing down existing test orgs/spaces...")
	setup = workflowhelpers.NewTestSuiteSetup(cfg)
	CleanupOrgs(cfg, setup)
	setup.Setup()
	EnableServiceAccess(setup, cfg, setup.GetOrganizationName())

	CheckServiceExists(cfg, setup.TestSpace.SpaceName(), cfg.ServiceName)
})
