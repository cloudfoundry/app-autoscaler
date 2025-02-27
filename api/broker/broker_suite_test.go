package broker_test

import (
	"encoding/json"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/brokerapi/v13/domain"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	testAppId      = "an-app-id"
	testInstanceId = "an-instance-id"
	testOrgId      = "an-org-id"
	testSpaceId    = "a-space-id"
	testBindingId  = "a-binding-id"
	dashBoardURL   = "https://autoscaler-dashboard.example.com"
)

var (
	conf              *config.Config
	services          []domain.Service
	testServiceID     string
	testPlanID        string
	testDefaultPolicy string
	testDefaultGuid   string
	scalingPolicy     = &models.ScalingPolicy{}
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BrokerServer Suite")
}

var _ = BeforeSuite(func() {
	policyBytes, err := os.ReadFile("testdata/policy.json")
	Expect(err).ShouldNot(HaveOccurred())
	testDefaultPolicy = string(policyBytes)
	err = json.Unmarshal(policyBytes, scalingPolicy)
	Expect(err).ShouldNot(HaveOccurred())

	testDefaultGuid = "a-not-so-guid"

	conf = &config.Config{
		CatalogPath:          "../exampleconfig/catalog-example.json",
		DashboardRedirectURI: dashBoardURL,
	}

	catalogBytes, err := os.ReadFile("../exampleconfig/catalog-example.json")
	Expect(err).NotTo(HaveOccurred())
	catalog := &struct {
		Services []domain.Service `json:"services"`
	}{}
	err = json.Unmarshal(catalogBytes, catalog)
	Expect(err).NotTo(HaveOccurred())
	services = catalog.Services

	testServiceID = services[0].ID
	testPlanID = services[0].Plans[0].ID
})
