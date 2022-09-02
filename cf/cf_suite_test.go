package cf_test

import (
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	conf            *cf.Config
	cfc             *cf.Client
	fakeCC          *MockServer
	fakeLoginServer *MockServer
	err             error
	logger          lager.Logger
	fclock          *fakeclock.FakeClock
	fakeLoginUrl    string
)

func setCfcClient(maxRetries int) {
	conf = &cf.Config{}
	conf.ClientID = "test-client-id"
	conf.Secret = "test-client-secret"
	conf.API = fakeCC.URL()
	conf.MaxRetries = maxRetries
	conf.MaxRetryWaitMs = 1
	conf.IdleTimeoutMs = 50
	conf.MaxIdleConnsPerHost = maxIdleConnsPerHost
	conf.SkipSSLValidation = true
	fclock = fakeclock.NewFakeClock(time.Now())
	cfc = cf.NewCFClient(conf, logger, fclock)
}

func login() {
	fakeCC.Add().Info(fakeLoginUrl)
	fakeLoginServer.Add().OauthToken("test-access-token")
	err = cfc.Login()
}

var _ = BeforeEach(func() {
	err = nil
	fakeCC = NewMockServer()
	fakeLoginServer = NewMockServer()
	logger = lager.NewLogger("cf")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
	fakeLoginUrl = fakeLoginServer.URL()
	setCfcClient(0)
})

var _ = AfterEach(func() {
	if fakeCC != nil {
		fakeCC.Close()
	}
	if fakeLoginServer != nil {
		fakeLoginServer.Close()
	}
})

func TestCfClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cf Suite")
}
