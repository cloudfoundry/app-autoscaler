package cf_test

import (
	"net/http/httptest"
	"testing"

	"metrics-collector/cf/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
	testAuthServer = httptest.NewServer(fakes.NewFakeAuthServerHandler())
	testAuthUrl = "http://" + testAuthServer.Listener.Addr().String()

	testApiServer = httptest.NewServer(fakes.NewFakeApiServerHandler(testAuthUrl))
	testApiUrl = "http://" + testApiServer.Listener.Addr().String()
})

var _ = AfterSuite(func() {
	testAuthServer.Close()
	testApiServer.Close()

})
