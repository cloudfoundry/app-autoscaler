package collector_test

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/tedsuo/ifrit/ginkgomon_v2"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsserver/collector"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"testing"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var (
	serverProcess ifrit.Process
	serverUrl     *url.URL
)

const (
	TestCollectInterval = 1 * time.Second
	TestRefreshInterval = 2 * time.Second
	TestSaveInterval    = 2 * time.Second
)

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collector Suite")
}

var _ = BeforeSuite(func() {

	port := 1111 + GinkgoParallelProcess()
	serverConf := &collector.ServerConfig{
		Port:      port,
		NodeAddrs: []string{fmt.Sprintf("%s:%d", "localhost", port)},
		NodeIndex: 0,
	}

	queryFunc := func(appID string, instanceIndex int, name string, start, end int64, order db.OrderType) ([]*models.AppInstanceMetric, error) {
		return nil, nil
	}

	httpStatusCollector := &fakes.FakeHTTPStatusCollector{}

	logger := lager.NewLogger("collector_suite_test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

	httpServer, err := collector.NewServer(logger, serverConf, queryFunc, httpStatusCollector)
	Expect(err).NotTo(HaveOccurred())

	serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port))
	Expect(err).ToNot(HaveOccurred())

	serverProcess = ginkgomon_v2.Invoke(httpServer)
})

var _ = AfterSuite(func() {
	ginkgomon_v2.Interrupt(serverProcess)
})
