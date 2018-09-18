package server_test

import (
	"autoscaler/db"
	"autoscaler/metricscollector/config"
	"autoscaler/metricscollector/fakes"
	"autoscaler/metricscollector/server"
	"autoscaler/models"
	"fmt"
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

	port := 1111 + GinkgoParallelNode()
	cfc := &fakes.FakeCFClient{}
	consumer := &fakes.FakeNoaaConsumer{}
	conf := &config.Config{
		Server: config.ServerConfig{
			Port:      port,
			NodeAddrs: []string{fmt.Sprintf("%s:%s", "localhost", port)},
			NodeIndex: 0,
		},
	}
	database := &fakes.FakeInstanceMetricsDB{}
	queryFunc := func(appID string, start int64, end int64, order db.OrderType, labels map[string]string) ([]*models.AppInstanceMetric, bool) {
		return nil, false
	}

	httpServer, err := server.NewServer(lager.NewLogger("test"), conf, cfc, consumer, queryFunc, database)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
	Expect(err).ToNot(HaveOccurred())

	serverProcess = ginkgomon.Invoke(httpServer)
})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)
})
