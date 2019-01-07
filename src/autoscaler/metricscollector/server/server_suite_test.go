package server_test

import (
	"autoscaler/db"
	"autoscaler/fakes"
	"autoscaler/metricscollector/config"
	"autoscaler/metricscollector/server"
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"fmt"
	"net/url"
	"strconv"
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
	conf := &config.ServerConfig{
		Port:      port,
		NodeAddrs: []string{fmt.Sprintf("%s:%d", "localhost", port)},
		NodeIndex: 0,
	}
	database := &fakes.FakeInstanceMetricsDB{}
	queryFunc := func(appID string, start int64, end int64, order db.OrderType, labels map[string]string) ([]*models.AppInstanceMetric, bool) {
		return nil, false
	}
	httpStatusCollector := &fakes.FakeHTTPStatusCollector{}
	httpServer, err := server.NewServer(lager.NewLogger("test"), conf, queryFunc, database, httpStatusCollector)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
	Expect(err).ToNot(HaveOccurred())

	serverProcess = ginkgomon.Invoke(httpServer)
})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)
})
