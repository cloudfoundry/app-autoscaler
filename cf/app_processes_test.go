package cf_test

import (
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"

	"net/http"
)

var _ = Describe("Cf client App", func() {

	var (
		conf            *cf.Config
		cfc             cf.CFClient
		fakeCC          *MockServer
		fakeLoginServer *Server
		err             error
		logger          lager.Logger
	)

	var setCfcClient = func(maxRetries int) {
		conf = &cf.Config{}
		conf.API = fakeCC.URL()
		conf.MaxRetries = maxRetries
		conf.MaxRetryWaitMs = 1
		cfc = cf.NewCFClient(conf, logger, clock.NewClock())
		err = cfc.Login()
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		fakeCC = NewMockServer()
		fakeLoginServer = NewServer()
		fakeCC.Add().Info(fakeLoginServer.URL())
		fakeLoginServer.RouteToHandler("POST", cf.PathCFAuth, RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
			AccessToken: "test-access-token",
			ExpiresIn:   12000,
		}))
		logger = lager.NewLogger("cf")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		setCfcClient(0)
	})

	AfterEach(func() {
		if fakeCC != nil {
			fakeCC.Close()
		}
		if fakeLoginServer != nil {
			fakeLoginServer.Close()
		}
	})

	Describe("GetAppProcesses", func() {

		When("get process succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/apps/test-app-id/processes", "per_page=100&types=web,worker"),
						VerifyHeaderKV("Authorization", "Bearer test-access-token"),
						RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct state", func() {
				processes, err := cfc.GetAppProcesses("test-app-id", cf.ProcessTypeWeb, cf.ProcessTypeWorker)
				Expect(err).NotTo(HaveOccurred())
				created, err := time.Parse(time.RFC3339, "2016-03-23T18:48:22Z")
				Expect(err).NotTo(HaveOccurred())
				updated, err := time.Parse(time.RFC3339, "2016-03-23T18:48:42Z")
				Expect(err).NotTo(HaveOccurred())
				Expect(processes).To(Equal(cf.Processes{
					{
						Guid:       "6a901b7c-9417-4dc1-8189-d3234aa0ab82",
						Type:       "web",
						Instances:  5,
						MemoryInMb: 256,
						DiskInMb:   1024,
						CreatedAt:  created,
						UpdatedAt:  updated,
					},
					{
						Guid:       "3fccacd9-4b02-4b96-8d02-8e865865e9eb",
						Type:       "worker",
						Instances:  1,
						MemoryInMb: 256,
						DiskInMb:   1024,
						CreatedAt:  created,
						UpdatedAt:  updated,
					},
				}))
				Expect(processes.GetInstances()).To(Equal(6))
			})
		})

		When("get processes returns a 500 status code with non-JSON response", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						RespondWithJSONEncoded(http.StatusInternalServerError, ""),
					),
				)
			})

			It("should error", func() {
				process, err := cfc.GetAppProcesses("invalid_json", cf.ProcessTypeWeb)
				Expect(process).To(BeNil())
				Expect(err.Error()).To(MatchRegexp("failed GetAppProcesses 'invalid_json': failed getting page 1:.*failed to unmarshal"))
			})
		})
	})
})
