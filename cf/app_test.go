package cf_test

import (
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
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

	Describe("GetApp", func() {
		When("get app succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/apps/test-app-id"),
						RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct state", func() {
				app, err := cfc.GetApp("test-app-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(app).To(Equal(&cf.App{
					Guid:      "663e9a25-30ba-4fb4-91fa-9b784f4a8542",
					Name:      "autoscaler-1--0cde0e473e3e47f4",
					State:     "STOPPED",
					CreatedAt: ParseDate("2022-07-21T13:42:30Z"),
					UpdatedAt: ParseDate("2022-07-21T14:30:17Z"),
					Relationships: cf.Relationships{
						Space: &cf.Space{
							Data: cf.SpaceData{
								Guid: "3dfc4a10-6e70-44f8-989d-b3842f339e3b",
							},
						},
					},
				}))
			})
		})
	})

	Describe("GetAppProcesses", func() {

		When("get process succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/apps/test-app-id/processes", "per_page=100"),
						RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct state", func() {
				processes, err := cfc.GetAppProcesses("test-app-id")
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
						VerifyRequest("GET", "/v3/apps/invalid_json/processes"),
						RespondWithJSONEncoded(http.StatusInternalServerError, ""),
					),
				)
			})

			It("should error", func() {
				process, err := cfc.GetAppProcesses("invalid_json")
				Expect(process).To(BeNil())
				Expect(err.Error()).To(MatchRegexp("failed GetAppProcesses 'invalid_json': failed getting page 1:.*failed to unmarshal"))
			})
		})
	})

	Describe("GetAppAndProcesses", func() {

		When("the mocks are used", func() {
			var mocks = NewMockServer()
			BeforeEach(func() {
				conf.API = mocks.URL()
				mocks.Add().GetAppProcesses(27).Info(fakeLoginServer.URL())
				mocks.Add().GetApp("STARTED")
				DeferCleanup(mocks.Close)
			})
			It("will return success", func() {
				app, err := cfc.GetAppAndProcesses("test-app-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(app).To(Equal(&cf.AppAndProcesses{
					App: &cf.App{
						Guid:      "testing-guid-get-app",
						Name:      "mock-get-app",
						State:     "STARTED",
						CreatedAt: ParseDate("2022-07-21T13:42:30Z"),
						UpdatedAt: ParseDate("2022-07-21T14:30:17Z"),
						Relationships: cf.Relationships{
							Space: &cf.Space{
								Data: cf.SpaceData{
									Guid: "test_space_guid",
								},
							},
						},
					},
					Processes: cf.Processes{
						{
							Guid:       "",
							Type:       "",
							Instances:  27,
							MemoryInMb: 0,
							DiskInMb:   0,
							CreatedAt:  ParseDate("0001-01-01T00:00:00Z"),
							UpdatedAt:  ParseDate("0001-01-01T00:00:00Z"),
						},
					}}))
			})
		})

		When("get app & process return ok", func() {
			BeforeEach(func() {
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id/processes", CombineHandlers(
					RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
				))
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id", CombineHandlers(
					RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
				))
			})

			It("returns correct state", func() {
				appAndProcess, err := cfc.GetAppAndProcesses("test-app-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(appAndProcess).To(Equal(&cf.AppAndProcesses{
					App: &cf.App{
						Guid:      "663e9a25-30ba-4fb4-91fa-9b784f4a8542",
						Name:      "autoscaler-1--0cde0e473e3e47f4",
						State:     "STOPPED",
						CreatedAt: ParseDate("2022-07-21T13:42:30Z"),
						UpdatedAt: ParseDate("2022-07-21T14:30:17Z"),
						Relationships: cf.Relationships{
							Space: &cf.Space{
								Data: cf.SpaceData{
									Guid: "3dfc4a10-6e70-44f8-989d-b3842f339e3b",
								},
							},
						},
					},
					Processes: cf.Processes{
						{
							Guid:       "6a901b7c-9417-4dc1-8189-d3234aa0ab82",
							Type:       "web",
							Instances:  5,
							MemoryInMb: 256,
							DiskInMb:   1024,
							CreatedAt:  ParseDate("2016-03-23T18:48:22Z"),
							UpdatedAt:  ParseDate("2016-03-23T18:48:42Z"),
						},
						{
							Guid:       "3fccacd9-4b02-4b96-8d02-8e865865e9eb",
							Type:       "worker",
							Instances:  1,
							MemoryInMb: 256,
							DiskInMb:   1024,
							CreatedAt:  ParseDate("2016-03-23T18:48:22Z"),
							UpdatedAt:  ParseDate("2016-03-23T18:48:42Z"),
						}},
				}))
			})
		})

		When("get app returns 500 and get process return ok", func() {
			BeforeEach(func() {
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id/processes", CombineHandlers(
					RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
				))
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id", CombineHandlers(
					RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
				))
			})

			It("should error", func() {
				appAndProcesses, err := cfc.GetAppAndProcesses("test-app-id")
				Expect(appAndProcesses).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp(`get state&instances failed: failed GetAppProcesses 'test-app-id': failed getting page 1: failed getting cf.Response\[.*cf.Process\]:.*'UnknownError'`)))
			})
		})

		When("get processes return OK get app returns 500", func() {
			BeforeEach(func() {
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id/processes", CombineHandlers(
					RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
				))
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id", CombineHandlers(
					RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
				))
			})

			It("should error", func() {
				appAndProcesses, err := cfc.GetAppAndProcesses("test-app-id")
				Expect(appAndProcesses).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp("get state&instances failed: failed getting app 'test-app-id':.*'UnknownError'")))
			})
		})

		When("get processes return 500 and get app returns 500", func() {
			BeforeEach(func() {
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id/processes", CombineHandlers(
					RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
				))
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id", CombineHandlers(
					RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
				))
			})

			It("should error", func() {
				appAndProcesses, err := cfc.GetAppAndProcesses("test-app-id")
				Expect(appAndProcesses).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp(`get state&instances failed: .*'UnknownError'`)))
			})
		})
	})

	Describe("ScaleAppWebProcess", func() {
		JustBeforeEach(func() {
			err = cfc.ScaleAppWebProcess("test-app-id", 6)
		})

		When("scaling web app succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("POST", "/v3/apps/test-app-id/processes/web/actions/scale"),
						VerifyJSON(`{"instances":6}`),
						RespondWith(http.StatusAccepted, LoadFile("scale_response.yml")),
					),
				)
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("scaling endpoint return 500", func() {
			BeforeEach(func() {
				setCfcClient(3)
				fakeCC.RouteToHandler("POST",
					"/v3/apps/test-app-id/processes/web/actions/scale",
					RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError))
			})

			It("should error correctly", func() {
				Expect(fakeCC.Count().Requests(`^/v3/apps/[^/]+/processes/web/actions/scale$`)).To(Equal(4))
				Expect(err).To(MatchError(MatchRegexp("failed scaling app 'test-app-id' to 6: POST request failed:.*'UnknownError'.*")))
			})
		})

	})

})
