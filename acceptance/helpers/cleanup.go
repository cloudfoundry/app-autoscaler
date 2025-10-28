package helpers

import (
	"acceptance/config"
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo/v2"

	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
)

func CleanupOrgs(cfg *config.Config, wfh *workflowhelpers.ReproducibleTestSuiteSetup) {
	By("Clearing down existing test orgs/spaces...")
	workflowhelpers.AsUser(wfh.AdminUserContext(), cfg.DefaultTimeoutDuration(), func() {
		DeleteOrgs(GetTestOrgs(cfg), cfg.DefaultTimeoutDuration())
	})
	By("Clearing down existing test orgs/spaces... Complete")
}

func CleanupInExistingOrg(cfg *config.Config, setup *workflowhelpers.ReproducibleTestSuiteSetup) {
	workflowhelpers.AsUser(setup.AdminUserContext(), cfg.DefaultTimeoutDuration(), func() {
		if cfg.UseExistingOrganization {
			targetOrgWithSpace(setup.GetOrganizationName(), "", cfg.DefaultTimeoutDuration())
			orgGuid := GetOrgGuid(cfg, cfg.ExistingOrganization)
			spaceNames := GetTestSpaces(orgGuid, cfg)
			if len(spaceNames) == 0 {
				return
			}
			spaceName := spaceNames[0]
			deleteAllServices(cfg, orgGuid, setup.GetOrganizationName(), GetSpaceGuid(cfg, orgGuid), spaceName)
			deleteAllApps(cfg, orgGuid, setup.GetOrganizationName(), GetSpaceGuid(cfg, orgGuid), spaceName)
			DeleteSpaces(cfg.ExistingOrganization, GetTestSpaces(orgGuid, cfg), cfg.DefaultTimeoutDuration())
		}
	})
}

func deleteAllServices(cfg *config.Config, orgGuid string, orgName string, spaceGuid string, spaceName string) {
	waitGroup := sync.WaitGroup{}
	servicesChan := make(chan string)

	services := GetServices(cfg, orgGuid, spaceGuid)
	if len(services) == 0 {
		return
	}
	fmt.Printf(" - deleting existing service instances: %d\n", len(services))
	targetOrgWithSpace(orgName, spaceName, cfg.DefaultTimeoutDuration())
	for i := 1; i <= cfg.Performance.SetupWorkers; i++ {
		waitGroup.Add(1)
		go deleteExistingServiceInstances(i, cfg, servicesChan, &waitGroup)
	}
	putResourceOnChannel(services, servicesChan)
	close(servicesChan)
	fmt.Printf("waiting for workers to finish...\n")
	waitGroup.Wait()
}

func putResourceOnChannel(services []string, servicesChan chan<- string) {
	for _, serviceInstanceName := range services {
		servicesChan <- serviceInstanceName
	}
}

func deleteAllApps(cfg *config.Config, orgGuid string, orgName string, spaceGuid string, spaceName string) {
	waitGroup := sync.WaitGroup{}
	appsChan := make(chan string)

	apps := GetApps(cfg, orgGuid, spaceGuid, "node-custom-metric-benchmark-")
	fmt.Printf("\n - deleting existing apps: %d\n", len(apps))
	if len(apps) == 0 {
		return
	}
	targetOrgWithSpace(orgName, spaceName, cfg.DefaultTimeoutDuration())
	for i := 1; i <= cfg.Performance.SetupWorkers; i++ {
		waitGroup.Add(1)
		go deleteExistingApps(i, cfg, appsChan, &waitGroup)
	}
	putResourceOnChannel(apps, appsChan)
	close(appsChan)
	fmt.Printf("waiting for workers to finish...\n")
	waitGroup.Wait()
}

func deleteExistingServiceInstances(workerId int, cfg *config.Config, servicesChan <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer GinkgoRecover()

	for instanceName := range servicesChan {
		fmt.Printf(" - worker %d    - deleting service instance - %s\n", workerId, instanceName)
		DeleteServiceInstance(cfg, instanceName)
	}
}

func deleteExistingApps(workerId int, cfg *config.Config, appsChan <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer GinkgoRecover()

	for appName := range appsChan {
		fmt.Printf(" - worker %d    - deleting app instance - %s\n", workerId, appName)
		DeleteTestApp(appName, cfg.DefaultTimeoutDuration())
	}
}
