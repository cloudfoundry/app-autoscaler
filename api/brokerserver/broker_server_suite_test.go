package brokerserver_test

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/brokerserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"github.com/onsi/gomega/ghttp"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon_v2"

	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	username       = "broker_username"
	password       = "broker_password"
	username2      = "broker_username2"
	password2      = "broker_password2"
	testAppId      = "an-app-id"
	testInstanceId = "an-instance-id"
	testOrgId      = "an-org-id"
	testSpaceId    = "a-space-id"
	testBindingId  = "a-binding-id"
)

var (
	serverProcess     ifrit.Process
	serverUrl         *url.URL
	httpClient        *http.Client
	conf              *config.Config
	catalogBytes      []byte
	schedulerServer   *ghttp.Server
	tokenServer       *ghttp.Server
	quotaServer       *ghttp.Server
	testDefaultPolicy string
	testDefaultGuid   string
	servers           []*ghttp.Server
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BrokerServer Suite")
}

var _ = BeforeSuite(func() {
	testDefaultPolicy = `
						{
							"instance_min_count":1,
							"instance_max_count":5,
							"scaling_rules":[
							{
								"metric_type":"memoryused",
								"threshold":30,
								"operator":"<",
								"adjustment":"-1"
							}]
						}`
	testDefaultGuid = "a-not-so-guid"

	schedulerServer = ghttp.NewServer()
	servers = append(servers, schedulerServer)
	tokenServer = ghttp.NewServer()
	servers = append(servers, tokenServer)
	quotaServer = ghttp.NewServer()
	servers = append(servers, quotaServer)

	port := 10000 + GinkgoParallelProcess()
	brokerCred1 := config.BrokerCredentialsConfig{
		BrokerUsername:     "broker_username",
		BrokerUsernameHash: []byte("$2a$10$WNO1cPko4iDAT6MkhaDojeJMU8ZdNH6gt.SapsFOsC0OF4cQ9qQwu"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_username")'
		BrokerPassword:     "broker_password",
		BrokerPasswordHash: []byte("$2a$10$evLviRLcIPKnWQqlBl3DJOvBZir9vJ4gdEeyoGgvnK/CGBnxIAFRu"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_password")'
	}
	brokerCred2 := config.BrokerCredentialsConfig{
		BrokerUsername:     "broker_username2",
		BrokerUsernameHash: []byte("$2a$10$NK76ms9n/oeD1.IumovhIu2fiiQ/4FIVc81o4rdNS8beJMxYvhTqG"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_username2")'
		BrokerPassword:     "broker_password2",
		BrokerPasswordHash: []byte("$2a$10$HZOfLweDfjNfe2h3KItdg.26BxNU6TVKMDwhJMNPPIWpj7T2HCVbW"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_password2")'
	}
	var brokerCreds []config.BrokerCredentialsConfig
	brokerCreds = append(brokerCreds, brokerCred1, brokerCred2)

	conf = &config.Config{
		BrokerServer: config.ServerConfig{
			Port: port,
		},
		BrokerCredentials: brokerCreds,
		QuotaManagement: &config.QuotaManagementConfig{
			API:      quotaServer.URL(),
			ClientID: "client-id",
			Secret:   "client-secret",
			TokenURL: tokenServer.URL(),
		},
		PlanCheck: &config.PlanCheckConfig{
			PlanDefinitions: map[string]config.PlanDefinition{
				"a-plan-id": {
					PlanCheckEnabled:  true,
					PlanUpdateable:    true,
					SchedulesCount:    1,
					ScalingRulesCount: 1,
				},
				"a-plan-id-not-updatable": {
					PlanCheckEnabled:  true,
					SchedulesCount:    1,
					ScalingRulesCount: 1,
				},
			},
		},
		CatalogPath:       "../exampleconfig/catalog-example.json",
		CatalogSchemaPath: "../schemas/catalog.schema.json",
		PolicySchemaPath:  "../policyvalidator/policy_json.schema.json",
		Scheduler: config.SchedulerConfig{
			SchedulerURL: schedulerServer.URL(),
		},
		InfoFilePath: "../exampleconfig/info-file.json",
		MetricsForwarder: config.MetricsForwarderConfig{
			MetricsForwarderUrl:     "someURL",
			MetricsForwarderMtlsUrl: "Mtls-someURL",
		},
	}
	fakeBindingDB := &fakes.FakeBindingDB{}
	fakePolicyDB := &fakes.FakePolicyDB{}
	fakeCredentials := &fakes.FakeCredentials{}
	httpStatusCollector := &fakes.FakeHTTPStatusCollector{}
	httpServer, err := brokerserver.NewBrokerServer(lager.NewLogger("test"), conf, fakeBindingDB, fakePolicyDB,
		httpStatusCollector, nil, fakeCredentials)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://localhost:" + strconv.Itoa(port))
	Expect(err).NotTo(HaveOccurred())

	serverProcess = ginkgomon_v2.Invoke(httpServer)

	httpClient = &http.Client{}

	catalogBytes, err = ioutil.ReadFile("../exampleconfig/catalog-example.json")
	Expect(err).NotTo(HaveOccurred())

	urlPath, _ := routes.SchedulerRoutes().Get(routes.UpdateScheduleRouteName).URLPath("appId", testAppId)
	schedulerServer.RouteToHandler("PUT", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
	schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
})

var _ = AfterSuite(func() {
	for _, server := range servers {
		server.Close()
	}
	ginkgomon_v2.Interrupt(serverProcess)
})
