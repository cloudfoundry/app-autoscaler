package server_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"net/url"
	"strconv"
	"testing"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon_v2"
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
	port := 1111 + GinkgoParallelProcess()
	conf := &config.Config{
		Server: config.ServerConfig{
			Port: port,
		},
	}
	queryAppMetrics := func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error) {
		return nil, nil
	}

	httpStatusCollector := &fakes.FakeHTTPStatusCollector{}
	httpServer, err := server.NewServer(lager.NewLogger("test"), conf, queryAppMetrics, httpStatusCollector)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
	Expect(err).ToNot(HaveOccurred())

	serverProcess = ginkgomon_v2.Invoke(httpServer)
})

var _ = AfterSuite(func() {
	ginkgomon_v2.Interrupt(serverProcess)
})
