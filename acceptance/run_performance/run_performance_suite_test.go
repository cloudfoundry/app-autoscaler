package run_performance_test

import (
	"acceptance/config"
	. "acceptance/helpers"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	cfg       *config.Config
	setup     *workflowhelpers.ReproducibleTestSuiteSetup
	orgName   string
	spaceName string
)

func TestSetup(t *testing.T) {
	RegisterFailHandler(Fail)
	cfg = config.LoadConfig(config.DefaultTerminateSuite)
	cfg.Prefix = "autoscaler-performance"
	setup = workflowhelpers.NewTestSuiteSetup(cfg)
	RunSpecs(t, "Performance Test Suite")
}

var _ = BeforeSuite(func() {
	// use smoke test to avoid creating a new user
	setup = workflowhelpers.NewSmokeTestSuiteSetup(cfg)

	if cfg.UseExistingOrganization && !cfg.UseExistingSpace {
		orgGuid := GetOrgGuid(cfg, cfg.ExistingOrganization)
		spaces := GetTestSpaces(orgGuid, cfg)
		Expect(len(spaces)).To(Equal(1), "Found more than one space in existing org %s", cfg.ExistingOrganization)
		cfg.ExistingSpace = spaces[0]
	} else {
		workflowhelpers.AsUser(setup.AdminUserContext(), cfg.DefaultTimeoutDuration(), func() {
			orgName, spaceName = FindExistingOrgAndSpace(cfg)
		})

		Expect(orgName).ToNot(Equal(""), "orgName has not been determined")
		Expect(spaceName).ToNot(Equal(""), "spaceName has not been determined")

		cfg.ExistingOrganization = orgName
		cfg.ExistingSpace = spaceName
	}

	cfg.UseExistingOrganization = true
	cfg.UseExistingSpace = true

	setup = workflowhelpers.NewTestSuiteSetup(cfg)
	setup.Setup()

	CheckServiceExists(cfg, setup.TestSpace.SpaceName(), cfg.ServiceName)
})

var _ = AfterSuite(func() {

	if os.Getenv("SKIP_TEARDOWN") == "true" {
		fmt.Println("Skipping Teardown...")
	} else {
		cleanup(cfg, setup)
		setup.Teardown()
	}
})

func cleanup(cfg *config.Config, setup *workflowhelpers.ReproducibleTestSuiteSetup) {
	fmt.Printf("\nCleaning up test leftovers...")
	workflowhelpers.AsUser(setup.AdminUserContext(), cfg.DefaultTimeoutDuration(), func() {
		if cfg.UseExistingOrganization {
			CleanupInExistingOrg(cfg, setup)
		} else {
			DeleteOrgs(GetTestOrgs(cfg), time.Duration(120)*time.Second)
		}
	})
	fmt.Printf("\nCleaning up test leftovers...completed")
}
