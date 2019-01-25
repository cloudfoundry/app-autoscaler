package brokerserver_test

import (
	"autoscaler/api/brokerserver"
	"autoscaler/api/config"
	"autoscaler/fakes"
	"io/ioutil"
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
)

var (
	serverProcess ifrit.Process
	serverUrl     *url.URL
	httpClient    *http.Client
	conf          *config.Config
	catalogBytes  []byte
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BrokerServer Suite")
}

var _ = BeforeSuite(func() {
	port := 10000 + GinkgoParallelNode()
	conf = &config.Config{
		BrokerServer: config.ServerConfig{
			Port: port,
		},
		BrokerUsername:    username,
		BrokerPassword:    password,
		CatalogPath:       "../exampleconfig/catalog-example.json",
		CatalogSchemaPath: "../schemas/catalog.schema.json",
		InfoFilePath:      "../exampleconfig/info-file.json",
	}
	fakeBindingDB := &fakes.FakeBindingDB{}

	httpServer, err := brokerserver.NewBrokerServer(lager.NewLogger("test"), conf, fakeBindingDB)
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
