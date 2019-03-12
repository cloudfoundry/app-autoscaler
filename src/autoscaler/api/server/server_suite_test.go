package server_test

import (
	"autoscaler/api/config"
	"autoscaler/api/server"
	"autoscaler/fakes"
	"autoscaler/routes"
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
	username  = "brokeruser"
	password  = "supersecretpassword"
	testAppId = "an-app-id"
)

var (
	serverProcess   ifrit.Process
	serverUrl       *url.URL
	httpClient      *http.Client
	conf            *config.Config
	catalogBytes    []byte
	schedulerServer *ghttp.Server
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = BeforeSuite(func() {
	schedulerServer = ghttp.NewServer()
	port := 10000 + GinkgoParallelNode()
	conf = &config.Config{
		Server: config.ServerConfig{
			Port: port,
		},
		BrokerUsername:    username,
		BrokerPassword:    password,
		CatalogPath:       "../exampleconfig/catalog-example.json",
		CatalogSchemaPath: "../schemas/catalog.schema.json",
		PolicySchemaPath:  "../policyvalidator/policy_json.schema.json",
		Scheduler: config.SchedulerConfig{
			SchedulerURL: schedulerServer.URL(),
		},
	}
	fakeBindingDB := &fakes.FakeBindingDB{}
	fakePolicyDB := &fakes.FakePolicyDB{}

	httpServer, err := server.NewServer(lager.NewLogger("test"), conf, fakeBindingDB, fakePolicyDB)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
	Expect(err).NotTo(HaveOccurred())

	serverProcess = ginkgomon.Invoke(httpServer)

	httpClient = &http.Client{}

	catalogBytes, err = ioutil.ReadFile("../exampleconfig/catalog-example.json")
	Expect(err).NotTo(HaveOccurred())

	urlPath, _ := routes.SchedulerRoutes().Get(routes.UpdateScheduleRouteName).URLPath("appId", testAppId)
	schedulerServer.RouteToHandler("PUT", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
	schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))

})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)
})
