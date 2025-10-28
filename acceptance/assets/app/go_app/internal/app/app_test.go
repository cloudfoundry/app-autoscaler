package app_test

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/app"
	"github.com/fgrosse/zaptest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/steinfletcher/apitest"
)

var _ = Describe("Ginkgo/Server", func() {

	var (
		t GinkgoTInterface
	)

	BeforeEach(func() {
		t = GinkgoT()
	})

	Context("basic endpoint tests", func() {
		It("Root should respond correctly", func() {
			apiTest(nil, nil, nil, nil).
				Get("/").
				Expect(t).
				Status(http.StatusOK).
				Body(`{"name":"test-app"}`).
				End()
		})
		It("health", func() {
			apiTest(nil, nil, nil, nil).
				Get("/health").
				Expect(t).
				Status(http.StatusOK).
				Body(`{"status":"ok"}`).
				End()
		})
	})

	Context("Basic startup", func() {
		var testApp *http.Server
		var client *http.Client
		var port int
		BeforeEach(func() {
			logger := zaptest.LoggerWriter(GinkgoWriter)
			/* #nosec G102 -- CF apps run in a container */
			l, err := net.Listen("tcp", ":0")
			Expect(err).ToNot(HaveOccurred())
			port = l.Addr().(*net.TCPAddr).Port
			testApp = app.New(logger, "")
			DeferCleanup(testApp.Close)
			go func() {
				defer GinkgoRecover()
				if err := testApp.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
					panic(err)
				}
			}()
			client = &http.Client{Timeout: time.Second * 1}
		})

		It("should start up", func() {
			apitest.New().EnableNetworking(client).Get(fmt.Sprintf("http://localhost:%d/", port)).
				Expect(t).
				Status(http.StatusOK).
				Body(`{"name":"test-app"}`).
				End()
		})

	})
})

func apiTest(timeWaster app.TimeWaster, memoryGobbler app.MemoryGobbler, cpuWaster app.CPUWaster, customMetricClient app.CustomMetricClient) *apitest.APITest {
	GinkgoHelper()
	logger := zaptest.LoggerWriter(GinkgoWriter)

	return apitest.New().
		Handler(app.Router(logger, timeWaster, memoryGobbler, cpuWaster, nil, customMetricClient))
}
