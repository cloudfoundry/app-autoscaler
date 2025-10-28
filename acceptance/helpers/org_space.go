package helpers

import (
	"acceptance/config"

	"github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

func FindExistingOrgAndSpace(cfg *config.Config) (orgName string, spaceName string) {
	organizations := GetTestOrgs(cfg)
	Expect(len(organizations)).To(Equal(1))
	orgName = organizations[0]
	orgName, _, spaceName, _ = GetOrgSpaceNamesAndGuids(cfg, orgName)

	return orgName, spaceName
}

func GetOrgSpaceNamesAndGuids(cfg *config.Config, org string) (orgName string, orgGuid string, spaceName string, spaceGuid string) {
	orgGuid = GetOrgGuid(cfg, org)
	spaces := GetRawSpaces(orgGuid, cfg.DefaultTimeoutDuration())
	if len(spaces) == 0 {
		return org, orgGuid, "", ""
	}
	spaceName = spaces[0].Name
	spaceGuid = spaces[0].Guid

	ginkgo.GinkgoWriter.Printf("\nUsing Org: %s - %s\n", org, orgGuid)
	ginkgo.GinkgoWriter.Printf("\nUsing Space: %s - %s\n", spaceName, spaceGuid)
	return org, orgGuid, spaceName, spaceGuid
}
