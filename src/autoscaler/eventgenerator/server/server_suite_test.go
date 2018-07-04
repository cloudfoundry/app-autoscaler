package server_test

import (
	"autoscaler/eventgenerator/aggregator/fakes"
	"autoscaler/eventgenerator/config"
	"autoscaler/eventgenerator/server"

	"net/url"
	"strconv"
	"testing"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var (
	serverProcess ifrit.Process
	serverUrl     *url.URL
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = BeforeSuite(func() {
	port := 1111 + GinkgoParallelNode()
	conf := &config.Config{
		Server: config.ServerConfig{
			Port: port,
		},
	}
	database := &fakes.FakeAppMetricDB{}

	httpServer, err := server.NewServer(lager.NewLogger("test"), conf, database)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
	Expect(err).ToNot(HaveOccurred())

	serverProcess = ginkgomon.Invoke(httpServer)
})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)
})
