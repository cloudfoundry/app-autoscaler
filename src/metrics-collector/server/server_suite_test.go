package server_test

import (
	"metrics-collector/config"
	"metrics-collector/server/fakes"
	. "metrics-collector/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http/httptest"
	"testing"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var testDopplerServer *httptest.Server
var testDopplerUrl string

var _ = BeforeSuite(func() {

	testLoggingConfig := config.LoggingConfig{
		Level:       "info",
		File:        "",
		LogToStdout: false,
	}
	InitailizeLogger(&testLoggingConfig)

	testDopplerServer = httptest.NewServer(fakes.NewFakeDopplerHandler())
	testDopplerUrl = "ws://" + testDopplerServer.Listener.Addr().String()

})

var _ = AfterSuite(func() {
	testDopplerServer.Close()
})
