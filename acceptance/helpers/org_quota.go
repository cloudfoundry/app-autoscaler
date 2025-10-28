package helpers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type OrgQuota struct {
	Name             string
	TotalMemory      string
	InstanceMemory   string
	Routes           string
	ServiceInstances string
	AppInstances     string
	RoutePorts       string
}

func appendIfPresent(args []string, option string, property string) []string {
	if property != "" {
		args = append(args, option, property)
	}
	return args
}

func UpdateOrgQuota(orgQuota OrgQuota, timeout time.Duration) {
	args := []string{"update-org-quota", orgQuota.Name}
	args = appendIfPresent(args, "-a", orgQuota.AppInstances)
	args = appendIfPresent(args, "-r", orgQuota.Routes)
	args = appendIfPresent(args, "-s", orgQuota.ServiceInstances)
	args = appendIfPresent(args, "-m", orgQuota.TotalMemory)
	args = appendIfPresent(args, "--reserved-route-ports", orgQuota.RoutePorts)
	updateOrgQuota := cf.Cf(args...).Wait(timeout)
	Expect(updateOrgQuota).To(Exit(0), "unable to update org quota "+orgQuota.Name+" : "+string(updateOrgQuota.Out.Contents()[:]))
	args = []string{"org-quota", orgQuota.Name}
	currentQuota := cf.Cf(args...).Wait(timeout)
	Expect(currentQuota).To(Exit(0), "unable to get org quota "+orgQuota.Name+" : "+string(updateOrgQuota.Out.Contents()[:]))
	fmt.Printf("%s", currentQuota.Out.Contents())
}

func GetOrgQuota(orgGuid string, timeout time.Duration) (orgQuota OrgQuota) {
	rawQuota := getRawOrgQuota(orgGuid, timeout).Resources[0]
	orgQuota = OrgQuota{Name: rawQuota.Name}

	if rawQuota.Apps.TotalMemoryInMb != 0 {
		orgQuota.TotalMemory = strconv.Itoa(rawQuota.Apps.TotalMemoryInMb) + "MB"
	}
	if rawQuota.Apps.PerProcessMemoryInMb != 0 {
		orgQuota.InstanceMemory = strconv.Itoa(rawQuota.Apps.PerProcessMemoryInMb) + "MB"
	}
	if rawQuota.Routes.TotalRoutes != 0 {
		orgQuota.Routes = strconv.Itoa(rawQuota.Routes.TotalRoutes)
	}
	if rawQuota.Services.TotalServiceInstances != 0 {
		orgQuota.ServiceInstances = strconv.Itoa(rawQuota.Services.TotalServiceInstances)
	}
	if rawQuota.Apps.TotalInstances != 0 {
		orgQuota.AppInstances = strconv.Itoa(rawQuota.Apps.TotalInstances)
	}
	if rawQuota.Routes.TotalRoutes != 0 {
		orgQuota.Routes = strconv.Itoa(rawQuota.Routes.TotalRoutes)
	}
	return orgQuota
}

func getRawOrgQuota(orgGuid string, timeout time.Duration) cfResourceObject {
	var quota cfResourceObject
	rawQuota := cf.Cf("curl", "/v3/organization_quotas?organization_guids="+orgGuid).Wait(timeout)
	Expect(rawQuota).To(Exit(0), "unable to get services")
	err := json.Unmarshal(rawQuota.Out.Contents(), &quota)
	Expect(err).NotTo(HaveOccurred())
	return quota
}
