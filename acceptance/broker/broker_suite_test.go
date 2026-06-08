package broker_test

import (
	"fmt"
	"os"
	"testing"

	"acceptance/config"
	. "acceptance/helpers"

	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	cfg   *config.Config
	setup *workflowhelpers.ReproducibleTestSuiteSetup
)

const componentName = "Broker Suite"

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
	DeferCleanup(func() {
		if os.Getenv("SKIP_TEARDOWN") == "true" {
			fmt.Println("Skipping Teardown...")
		} else {
			DisableServiceAccess(cfg, setup)
			setup.Teardown()
		}
	})
})
