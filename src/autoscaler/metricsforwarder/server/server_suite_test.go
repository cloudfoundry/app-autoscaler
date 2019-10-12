package server_test

import (
	"autoscaler/fakes"
	"autoscaler/helpers"
	"autoscaler/metricsforwarder/config"
	. "autoscaler/metricsforwarder/server"
	"autoscaler/models"

	"fmt"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cache "github.com/patrickmn/go-cache"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"testing"
)

var (
	serverProcess ifrit.Process
	serverUrl     string
	policyDB      *fakes.FakePolicyDB
	rateLimiter   *fakes.FakeLimiter

	credentialCache    cache.Cache
	allowedMetricCache cache.Cache
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	return nil
}, func(_ []byte) {

	testCertDir := "../../../../test-certs"
	loggregatorConfig := config.LoggregatorConfig{
		TLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "metron.key"),
			CertFile:   filepath.Join(testCertDir, "metron.crt"),
			CACertFile: filepath.Join(testCertDir, "loggregator-ca.crt"),
		},
	}
	serverConfig := config.ServerConfig{
		Port: 2222 + GinkgoParallelNode(),
	}

	loggerConfig := helpers.LoggingConfig{
		Level: "debug",
	}

	conf := &config.Config{
		Server:            serverConfig,
		Logging:           loggerConfig,
		LoggregatorConfig: loggregatorConfig,
	}
	policyDB = &fakes.FakePolicyDB{}
	credentialCache = *cache.New(10*time.Minute, -1)
	allowedMetricCache = *cache.New(10*time.Minute, -1)
	httpStatusCollector := &fakes.FakeHTTPStatusCollector{}
	rateLimiter = &fakes.FakeLimiter{}
	httpServer, err := NewServer(lager.NewLogger("test"), conf, policyDB, credentialCache, allowedMetricCache, httpStatusCollector, rateLimiter)
	Expect(err).NotTo(HaveOccurred())
	serverUrl = fmt.Sprintf("http://127.0.0.1:%d", conf.Server.Port)
	serverProcess = ginkgomon.Invoke(httpServer)
})

var _ = SynchronizedAfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)
}, func() {
})
