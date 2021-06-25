package collector_test

import (
	"autoscaler/db"
	"autoscaler/fakes"
	"autoscaler/metricsserver/collector"
	"autoscaler/metricsserver/config"
	"autoscaler/models"
	"fmt"
	"net/url"
	"strconv"
	"time"

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

const (
	TestCollectInterval time.Duration = 1 * time.Second
	TestRefreshInterval time.Duration = 2 * time.Second
	TestSaveInterval    time.Duration = 2 * time.Second
)

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collector Suite")
}

var _ = BeforeSuite(func() {

	port := 1111 + GinkgoParallelNode()
	serverConf := &config.ServerConfig{
		Port: port,
	}

	conf := &config.Config{
		NodeAddrs: []string{fmt.Sprintf("%s:%d", "localhost", port)},
		NodeIndex: 0,
	}

	queryFunc := func(appID string, instanceIndex int, name string, start, end int64, order db.OrderType) ([]*models.AppInstanceMetric, error) {
		return nil, nil
	}

	httpStatusCollector := &fakes.FakeHTTPStatusCollector{}
	httpServer, err := collector.NewServer(lager.NewLogger("test"), serverConf, conf, queryFunc, httpStatusCollector)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
	Expect(err).ToNot(HaveOccurred())

	serverProcess = ginkgomon.Invoke(httpServer)
})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)
})
