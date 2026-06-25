package helpers

import (
	"acceptance/config"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type SpaceResources struct {
	Resources []Space `json:"resources"`
}
type Space struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
}

func GetSpaceGuid(cfg *config.Config, orgGuid string) string {
	testSpace := GetTestSpaces(orgGuid, cfg)[0]
	return getSpaceGuidByName(testSpace, cfg.DefaultTimeoutDuration())
}

func getSpaceGuidByName(spaceName string, timeout time.Duration) string {
	spaceGuidCmd := cf.Cf("space", spaceName, "--guid").Wait(timeout)
	Expect(spaceGuidCmd).To(Exit(0), fmt.Sprintf("failed to get GUID for space %s: %s", spaceName, string(spaceGuidCmd.Err.Contents())))
	return strings.TrimSuffix(string(spaceGuidCmd.Out.Contents()), "\n")
}

func GetTestSpaces(orgGuid string, cfg *config.Config) []string {
	rawSpaces := GetRawSpaces(orgGuid, cfg.DefaultTimeoutDuration())

	var spaceNames []string
	for _, space := range rawSpaces {
		if strings.HasPrefix(space.Name, cfg.NamePrefix) {
			spaceNames = append(spaceNames, space.Name)
		}
	}
	ginkgo.GinkgoWriter.Printf("\nGot spaces: %s\n", spaceNames)
	return spaceNames
}

func filterTestSpaces(spaces []Space, namePrefix string) []Space {
	var result []Space
	for _, space := range spaces {
		if strings.HasPrefix(space.Name, namePrefix) {
			result = append(result, space)
		}
	}
	return result
}

func GetRawSpaces(orgGuid string, timeout time.Duration) []Space {
	params := url.Values{"organization_guids": []string{orgGuid}}
	rawSpaces := cf.CfSilent("curl", fmt.Sprintf("/v3/spaces?%s", params.Encode())).Wait(timeout)
	Expect(rawSpaces).To(Exit(0), "unable to get spaces", timeout)
	spaces := SpaceResources{}
	err := json.Unmarshal(rawSpaces.Out.Contents(), &spaces)
	Expect(err).ShouldNot(HaveOccurred())
	return spaces.Resources
}

func DeleteSpaces(orgName string, spaces []string, timeout time.Duration) {
	if len(spaces) == 0 {
		return
	}
	fmt.Printf("\nDeleting spaces: %s \n", strings.Join(spaces, ", "))
	for _, spaceName := range spaces {
		if timeout.Seconds() != 0 {
			deleteSpace := cf.Cf("delete-space", "-f", "-o", orgName, spaceName).Wait(timeout)
			Expect(deleteSpace).To(Exit(0), fmt.Sprintf("failed deleting space: %s in org: %s: %s", spaceName, orgName, string(deleteSpace.Err.Contents())))
		} else {
			cf.Cf("delete-space", "-f", "-o", orgName, spaceName)
		}
	}
}
