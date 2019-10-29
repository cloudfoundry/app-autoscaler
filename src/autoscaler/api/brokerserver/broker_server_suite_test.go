package brokerserver_test

import (
	"autoscaler/api/brokerserver"
	"autoscaler/api/config"
	"autoscaler/fakes"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/onsi/gomega/ghttp"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	username       = "brokeruser"
	usernameHash   = "$2a$12$S44P8nP0b.wq7kW21anaR.uU1dBMCGHUZxw7pdcy42z6oJK0TFTM." // ruby -r bcrypt -e 'puts BCrypt::Password.create("brokeruser")'
	password       = "supersecretpassword"
	passwordHash   = "$2a$12$8/xRXDhCyl0I..z76PG5Q.pWNLoVs0aYncx6UU1hToRAuevVjKm6O" // ruby -r bcrypt -e 'puts BCrypt::Password.create("supersecretpassword")'
	testAppId      = "an-app-id"
	testInstanceId = "an-instance-id"
	testOrgId      = "an-org-id"
	testSpaceId    = "a-space-id"
)

var (
	serverProcess     ifrit.Process
	serverUrl         *url.URL
	httpClient        *http.Client
	conf              *config.Config
	catalogBytes      []byte
	schedulerServer   *ghttp.Server
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
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BrokerServer Suite")
}

var _ = BeforeSuite(func() {
	schedulerServer = ghttp.NewServer()
	port := 10000 + GinkgoParallelNode()
	conf = &config.Config{
		BrokerServer: config.ServerConfig{
			Port: port,
		},
		BrokerUsernameHash: usernameHash,
		BrokerPasswordHash: passwordHash,
		CatalogPath:        "../exampleconfig/catalog-example.json",
		CatalogSchemaPath:  "../schemas/catalog.schema.json",
		PolicySchemaPath:   "../policyvalidator/policy_json.schema.json",
		Scheduler: config.SchedulerConfig{
			SchedulerURL: schedulerServer.URL(),
		},
		InfoFilePath: "../exampleconfig/info-file.json",
	}
	fakeBindingDB := &fakes.FakeBindingDB{}
	fakePolicyDB := &fakes.FakePolicyDB{}
	httpStatusCollector := &fakes.FakeHTTPStatusCollector{}
	httpServer, err := brokerserver.NewBrokerServer(lager.NewLogger("test"), conf, fakeBindingDB, fakePolicyDB, httpStatusCollector)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
	Expect(err).NotTo(HaveOccurred())

	serverProcess = ginkgomon.Invoke(httpServer)

	httpClient = &http.Client{}

	catalogBytes, err = ioutil.ReadFile("../exampleconfig/catalog-example.json")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)
})
