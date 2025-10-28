package helpers

import (
	"acceptance/config"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type cfResourceObject struct {
	Pagination struct {
		TotalPages int `json:"total_pages"`
		Next       struct {
			Href string `json:"href"`
		} `json:"next"`
	} `json:"pagination"`
	Resources []cfResource `json:"resources"`
}

// "apps": {
// "total_memory_in_mb": 5120,
// "per_process_memory_in_mb": null,
// "total_instances": null,
// "per_app_tasks": null,
// "log_rate_limit_in_bytes_per_second": null
// },
// "services": {
// "paid_services_allowed": true,
// "total_service_instances": 40,
// "total_service_keys": null
// },
// "routes": {
// "total_routes": 40,
// "total_reserved_ports": null
// }
type cfResource struct {
	GUID      string `json:"guid"`
	CreatedAt string `json:"created_at"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	State     string `json:"state"`

	// ----------------- OrgQuota Resource fields ----------------
	Apps struct {
		TotalMemoryInMb              int `json:"total_memory_in_mb"`
		PerProcessMemoryInMb         int `json:"per_process_memory_in_mb"`
		TotalInstances               int `json:"total_instances"`
		PerAppTasks                  int `json:"per_app_tasks"`
		LogRateLimitInBytesPerSecond int `json:"log_rate_limit_in_bytes_per_second"`
	} `json:"apps"`
	Services struct {
		PaidServicesAllowed   bool `json:"paid_services_allowed"`
		TotalServiceInstances int  `json:"total_service_instances"`
		TotalServiceKeys      int  `json:"total_service_keys"`
	} `json:"services"`
	Routes struct {
		TotalRoutes        int `json:"total_routes"`
		TotalReservedPorts int `json:"total_reserved_ports"`
	} `json:"routes"`
	// ----------------- OrgQuota Resource fields ----------------
}

const (
	CustomMetricPath = "/v1/apps/{appId}/credential"
)

func GetServices(cfg *config.Config, orgGuid, spaceGuid string) []string {
	rawServices := getRawServices(spaceGuid, orgGuid, cfg.DefaultTimeoutDuration())
	return filterByPrefix(cfg.Prefix, getNames(rawServices))
}

func getRawServices(spaceGuid string, orgGuid string, timeout time.Duration) []cfResource {
	var rawServices []cfResource
	totalPages := 1

	for page := 1; page <= totalPages; page++ {
		var appsResponse = getRawServicesByPage(spaceGuid, orgGuid, page, timeout)
		GinkgoWriter.Println(appsResponse.Pagination.TotalPages)
		totalPages = appsResponse.Pagination.TotalPages
		rawServices = append(rawServices, appsResponse.Resources...)
	}

	return rawServices
}

func getRawServicesByPage(spaceGuid string, orgGuid string, page int, timeout time.Duration) cfResourceObject {
	var appsResponse cfResourceObject
	rawServices := cf.Cf("curl", "/v3/service_instances?space_guids="+spaceGuid+"&organization_guids="+orgGuid+"&page="+strconv.Itoa(page)).Wait(timeout)
	Expect(rawServices).To(Exit(0), "unable to get service instances")
	err := json.Unmarshal(rawServices.Out.Contents(), &appsResponse)
	Expect(err).ShouldNot(HaveOccurred())
	return appsResponse
}

func DeleteService(cfg *config.Config, instanceName, appName string) {
	if appName != "" && instanceName != "" {
		UnbindService(cfg, instanceName, appName)
	}
	DeleteServiceInstance(cfg, instanceName)
}

func DeleteServiceInstance(cfg *config.Config, instanceName string) {
	if instanceName != "" {
		deleteService := cf.Cf("delete-service", instanceName, "-f").Wait(cfg.DefaultTimeoutDuration())
		if deleteService.ExitCode() != 0 {
			PurgeService(cfg, instanceName)
		}
	}
}
func UnbindService(cfg *config.Config, instanceName string, appName string) {
	unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(cfg.DefaultTimeoutDuration())
	if unbindService.ExitCode() != 0 {
		PurgeService(cfg, instanceName)
	}
}

func PurgeService(cfg *config.Config, instanceName string) {
	purgeService := cf.Cf("purge-service-instance", instanceName, "-f").Wait(cfg.DefaultTimeoutDuration())
	Expect(purgeService).To(Exit(0), fmt.Sprintf("failed to purge service instance %s: %s: %s", instanceName, purgeService.Out.Contents(), purgeService.Err.Contents()))
}
