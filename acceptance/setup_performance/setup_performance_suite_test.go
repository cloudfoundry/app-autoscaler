package peformance_setup_test

import (
	"acceptance/config"
	. "acceptance/helpers"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	cfg                *config.Config
	setup              *workflowhelpers.ReproducibleTestSuiteSetup
	originalOrgQuota   OrgQuota
	nodeAppDropletPath string
)

func TestSetup(t *testing.T) {
	RegisterFailHandler(Fail)
	cfg = config.LoadConfig(config.DefaultTerminateSuite)
	cfg.Prefix = "autoscaler-performance-TESTS"
	setup = workflowhelpers.NewTestSuiteSetup(cfg)
	RunSpecs(t, "Setup Performance Test Suite")
}

var _ = BeforeSuite(func() {

	if os.Getenv("SKIP_TEARDOWN") == "true" {
		fmt.Println("Skipping Teardown...")
	} else {
		cleanup(cfg, setup)
	}
	setup = workflowhelpers.NewRunawayAppTestSuiteSetup(cfg)
	setup.Setup()

	EnableServiceAccess(setup, cfg, setup.GetOrganizationName())
	workflowhelpers.AsUser(setup.AdminUserContext(), cfg.DefaultTimeoutDuration(), func() {
		_, orgGuid, _, spaceGuid := GetOrgSpaceNamesAndGuids(cfg, setup.GetOrganizationName())
		Expect(spaceGuid).NotTo(BeEmpty())
		updateOrgQuotaForPerformanceTest(orgGuid)
	})

	CheckServiceExists(cfg, setup.TestSpace.SpaceName(), cfg.ServiceName)

	fmt.Print("\ncreating droplet...")
	nodeAppDropletPath = CreateDroplet(cfg)
	fmt.Printf("done and downloaded to %s\n", nodeAppDropletPath)
})

func updateOrgQuotaForPerformanceTest(orgGuid string) {
	if cfg.Performance.UpdateExistingOrgQuota {
		originalOrgQuota = GetOrgQuota(orgGuid, cfg.DefaultTimeoutDuration())
		fmt.Printf("\n=> originalOrgQuota %+v\n", originalOrgQuota)
		performanceOrgQuota := OrgQuota{
			Name:             originalOrgQuota.Name,
			AppInstances:     strconv.Itoa(cfg.Performance.AppCount * 2),
			TotalMemory:      strconv.Itoa(cfg.Performance.AppCount*256) + "MB",
			Routes:           strconv.Itoa(cfg.Performance.AppCount * 2),
			ServiceInstances: strconv.Itoa(cfg.Performance.AppCount * 2),
			RoutePorts:       "-1",
		}
		fmt.Printf("=> setting new org quota %s\n", originalOrgQuota.Name)
		UpdateOrgQuota(performanceOrgQuota, cfg.DefaultTimeoutDuration())
	}
}

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
