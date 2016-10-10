package server_test

import (
	"autoscaler/metricscollector/config"
	"autoscaler/metricscollector/fakes"
	"autoscaler/metricscollector/server"
	"net/url"
	"strconv"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"testing"
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
	port := 1111
	cfc := &fakes.FakeCfClient{}
	consumer := &fakes.FakeNoaaConsumer{}
	conf := config.ServerConfig{Port: port}
	database := &fakes.FakeMetricsDB{}
	httpServer := server.NewServer(lager.NewLogger("test"), conf, cfc, consumer, database)

	var err error
	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
	Expect(err).ToNot(HaveOccurred())

	serverProcess = ginkgomon.Invoke(httpServer)
})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)
})
