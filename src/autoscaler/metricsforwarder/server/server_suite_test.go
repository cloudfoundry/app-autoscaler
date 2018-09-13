package server_test

import (
	"autoscaler/metricsforwarder/config"
	"autoscaler/metricsforwarder/fakes"
	. "autoscaler/metricsforwarder/server"

	"fmt"
	"time"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/patrickmn/go-cache"

	"testing"
)

var (
	serverProcess ifrit.Process
	serverUrl     string
	policyDB      *fakes.FakePolicyDB

	credentialCache cache.Cache
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
		CACertFile:     filepath.Join(testCertDir, "loggregator-ca.crt"),
		ClientCertFile: filepath.Join(testCertDir, "metron.crt"),
		ClientKeyFile:  filepath.Join(testCertDir, "metron.key"),
	}
	serverConfig := config.ServerConfig{
		Port: 2222 + GinkgoParallelNode(),
	}
	loggerConfig := config.LoggingConfig{
		Level: "debug",
	}

	conf := &config.Config{
		Server:            serverConfig,
		Logging:           loggerConfig,
		LoggregatorConfig: loggregatorConfig,
	}
	policyDB = &fakes.FakePolicyDB{}
	credentialCache = *cache.New(10 * time.Minute, -1)
	httpServer, err := NewServer(lager.NewLogger("test"), conf, policyDB, credentialCache)
	Expect(err).NotTo(HaveOccurred())
	serverUrl = fmt.Sprintf("http://127.0.0.1:%d", conf.Server.Port)
	serverProcess = ginkgomon.Invoke(httpServer)
})

var _ = SynchronizedAfterSuite(func() {
	ginkgomon.Interrupt(serverProcess)
}, func() {
})
