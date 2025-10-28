package server_test

import (
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon_v2"

	"testing"
)

var (
	conf          *config.Config
	serverProcess ifrit.Process
	policyDB      *fakes.FakePolicyDB

	fakeBindingDB   *fakes.FakeBindingDB
	rateLimiter     *fakes.FakeLimiter
	fakeCredentials *fakes.FakeCredentials

	allowedMetricCache cache.Cache
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {

	_, err := os.ReadFile("../../test-certs/metron.key")
	Expect(err).NotTo(HaveOccurred())

	_, err = os.ReadFile("../../test-certs/metron.crt")
	Expect(err).NotTo(HaveOccurred())

	_, err = os.ReadFile("../../test-certs/loggregator-ca.crt")
	Expect(err).NotTo(HaveOccurred())

	return nil
}, func(_ []byte) {

	testCertDir := "../../test-certs"
	loggregatorConfig := config.LoggregatorConfig{
		TLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "metron.key"),
			CertFile:   filepath.Join(testCertDir, "metron.crt"),
			CACertFile: filepath.Join(testCertDir, "loggregator-ca.crt"),
		},
		MetronAddress: "invalid-host-name-blah:12345",
	}
	serverConfig := helpers.ServerConfig{
		Port: 2222 + GinkgoParallelProcess(),
	}

	loggerConfig := helpers.LoggingConfig{
		Level: "debug",
	}

	healthConfig := helpers.HealthConfig{
		ReadinessCheckEnabled: true,
		BasicAuth: models.BasicAuth{
			Username: "metricsforwarderhealthcheckuser",
			Password: "metricsforwarderhealthcheckpassword",
		},
	}
	conf = &config.Config{
		Server:            serverConfig,
		Logging:           loggerConfig,
		LoggregatorConfig: loggregatorConfig,
		Health:            healthConfig,
	}
	policyDB = &fakes.FakePolicyDB{}
	fakeBindingDB = &fakes.FakeBindingDB{}

	allowedMetricCache = *cache.New(10*time.Minute, -1)
	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "metricsforwarder")

	rateLimiter = &fakes.FakeLimiter{}
	fakeCredentials = &fakes.FakeCredentials{}

	logger := lager.NewLogger("server_suite_test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

	httpServer, err := NewServer(logger, conf, policyDB, fakeBindingDB,
		fakeCredentials, allowedMetricCache, httpStatusCollector, rateLimiter)
	Expect(err).NotTo(HaveOccurred())

	serverProcess = ginkgomon_v2.Invoke(httpServer)
})

var _ = SynchronizedAfterSuite(func() {
	ginkgomon_v2.Interrupt(serverProcess)
}, func() {
})
