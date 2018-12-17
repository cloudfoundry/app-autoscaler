package server_test

import (
	"autoscaler/api/config"
	"autoscaler/api/fakes"
	"autoscaler/api/server"
	"net/http"
	"net/url"
	"strconv"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	username = "brokeruser"
	password = "supersecretpassword"
	catalog  = `{
		"services": [{
		  "id": "autoscaler-guid",
		  "name": "autoscaler",
		  "description": "Automatically increase or decrease the number of application instances based on a policy you define.",
		  "bindable": true,
		  "plans": [{
			  "id": "autoscaler-free-plan-id",
			  "name": "autoscaler-free-plan",
			  "description": "This is the free service plan for the Auto-Scaling service."
		  }]
		}]
		}`
)

var (
	serverProcess ifrit.Process
	serverUrl     *url.URL
	httpClient    *http.Client
	conf          *config.Config
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = BeforeSuite(func() {
	port := 10000 + GinkgoParallelNode()
	conf = &config.Config{
		Server: config.ServerConfig{
			Port: port,
		},
		BrokerUsername: username,
		BrokerPassword: password,
		Catalog:        catalog,
	}
	fakeBindingDB := &fakes.FakeBindingDB{}

	httpServer, err := server.NewServer(lager.NewLogger("test"), conf, fakeBindingDB)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
	Expect(err).NotTo(HaveOccurred())

	serverProcess = ginkgomon.Invoke(httpServer)

	httpClient = &http.Client{}
})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)
})
