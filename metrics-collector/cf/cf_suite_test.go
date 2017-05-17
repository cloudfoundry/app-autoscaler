package cf_test

import (
	"github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/cf/fakes"
	"github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/config"
	. "github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http/httptest"
	"testing"
)

func TestSecurity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cf Suite")
}

var (
	testApiServer  *httptest.Server
	testAuthServer *httptest.Server

	testApiUrl  string
	testAuthUrl string
)

var _ = BeforeSuite(func() {

	testLoggingConfig := config.LoggingConfig{
		Level:       "info",
		File:        "",
		LogToStdout: false,
	}

	InitailizeLogger(&testLoggingConfig)

	testAuthServer = httptest.NewServer(fakes.NewFakeAuthServerHandler())
	testAuthUrl = "http://" + testAuthServer.Listener.Addr().String()

	testApiServer = httptest.NewServer(fakes.NewFakeApiServerHandler(testAuthUrl))
	testApiUrl = "http://" + testApiServer.Listener.Addr().String()
})

var _ = AfterSuite(func() {
	testAuthServer.Close()
	testApiServer.Close()

})
