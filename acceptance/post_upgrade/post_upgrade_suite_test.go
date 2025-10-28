package post_upgrade_test

import (
	"acceptance/config"
	. "acceptance/helpers"
	"fmt"
	"os"
	"testing"

	cth "github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	cfg       *config.Config
	setup     *workflowhelpers.ReproducibleTestSuiteSetup
	orgName   string
	orgGUID   string
	spaceName string
	spaceGUID string
)

const componentName = "Post Upgrade Test Suite"

func TestSetup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, componentName)
}

var _ = BeforeSuite(func() {
	cfg = config.LoadConfig(config.DefaultTerminateSuite)
	if cfg.GetArtifactsDirectory() != "" {
		cth.EnableCFTrace(cfg, componentName)
	}

	// use smoke test to avoid creating a new user
	setup = workflowhelpers.NewSmokeTestSuiteSetup(cfg)

	workflowhelpers.AsUser(setup.AdminUserContext(), cfg.DefaultTimeoutDuration(), func() {
		orgs := GetTestOrgs(cfg)
		Expect(len(orgs)).To(Equal(1))
		orgName = orgs[0]
		_, orgGUID, spaceName, spaceGUID = GetOrgSpaceNamesAndGuids(cfg, orgName)
	})

	Expect(orgName).ToNot(Equal(""), "orgName has not been determined")
	Expect(spaceName).ToNot(Equal(""), "spaceName has not been determined")

	// discover the org / space from the environment
	cfg.UseExistingOrganization = true
	cfg.UseExistingSpace = true

	cfg.ExistingOrganization = orgName
	cfg.ExistingSpace = spaceName

	setup = workflowhelpers.NewTestSuiteSetup(cfg)

	setup.Setup()

	CheckServiceExists(cfg, setup.TestSpace.SpaceName(), cfg.ServiceName)
})

var _ = AfterSuite(func() {
	if os.Getenv("SKIP_TEARDOWN") == "true" {
		fmt.Println("Skipping Teardown...")
	} else {
		CleanupOrgs(cfg, setup)
		setup.Teardown()
	}
})
