package cf_test

import (
	"context"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf/mocks"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"
)

const maxIdleConnsPerHost = 200

var _ = Describe("Cf client App", func() {
	BeforeEach(login)

	appTestJson := LoadFile("testdata/app.json")

	autoscalingDisabled := "true"
	Describe("GetApp", func() {
		When("get app succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/apps/test-app-id"),
						VerifyHeaderKV("Authorization", "Bearer test-access-token"),
						RespondWith(http.StatusOK, appTestJson, http.Header{"Content-Type": []string{"application/json"}}),
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
					Metadata: cf.Metadata{
						Labels: cf.Labels{
							DisableAutoscaling: &autoscalingDisabled,
						},
					},
				}))
			})
		})
	})

	setupStress := func() (*ConnectionWatcher, *ConnectionWatcher) {
		var ccWatcher, loginWatcher *ConnectionWatcher
		fakeCC.Close()
		server := NewUnstartedServer()
		fakeCC = mocks.NewWithServer(server)
		ccWatcher = NewConnectionWatcher(fakeCC.HTTPTestServer.Config.ConnState)
		fakeCC.HTTPTestServer.Config.ConnState = ccWatcher.OnStateChange
		fakeCC.Start()

		fakeLoginServer.Close()
		server = NewUnstartedServer()
		fakeLoginServer = mocks.NewWithServer(server)
		loginWatcher = NewConnectionWatcher(fakeLoginServer.HTTPTestServer.Config.ConnState)
		fakeLoginServer.HTTPTestServer.Config.ConnState = loginWatcher.OnStateChange
		fakeLoginServer.Start()
		fakeLoginUrl = fakeLoginServer.URL()

		// quiet logger
		logger = lager.NewLogger("cf")
		setCfcClient(2)
		login()
		return ccWatcher, loginWatcher
	}

	Describe("GetAppAndProcesses", func() {

		appProcessesJson := LoadFile("testdata/app_processes.json")

		When("get app & process return ok", func() {
			BeforeEach(func() {
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id/processes", CombineHandlers(
					RespondWith(http.StatusOK, appProcessesJson, http.Header{"Content-Type": []string{"application/json"}}),
				))
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id", CombineHandlers(
					RespondWith(http.StatusOK, appTestJson, http.Header{"Content-Type": []string{"application/json"}}),
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
						Metadata: cf.Metadata{
							Labels: cf.Labels{
								DisableAutoscaling: &autoscalingDisabled,
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
					RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
				))
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id", CombineHandlers(
					RespondWith(http.StatusOK, appTestJson, http.Header{"Content-Type": []string{"application/json"}}),
				))
			})

			It("should error", func() {
				appAndProcesses, err := cfc.GetAppAndProcesses("test-app-id")
				Expect(appAndProcesses).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp(`get state&instances failed: failed GetAppProcesses 'test-app-id': failed getting page 1: failed GET-ing cf.Response\[.*cf.Process\]:.*'UnknownError'`)))
			})
		})

		When("get processes return OK get app returns 500", func() {
			BeforeEach(func() {
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id/processes", CombineHandlers(
					RespondWith(http.StatusOK, appProcessesJson, http.Header{"Content-Type": []string{"application/json"}}),
				))
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id", CombineHandlers(
					RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
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
					RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
				))
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id", CombineHandlers(
					RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
				))
			})

			It("should error", func() {
				appAndProcesses, err := cfc.GetAppAndProcesses("test-app-id")
				Expect(appAndProcesses).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp(`get state&instances failed: .*'UnknownError'`)))
			})
		})

		Context("Given a significant load", func() {
			numberConcurrentUsers := 100
			numberSequentialPerUser := 10
			var ccWatcher *ConnectionWatcher
			var loginWatcher *ConnectionWatcher
			stats := &reqStats{}
			/*
			 * Note there is a login that goes to a separate host then 2 concurrent requests to the api server.
			 */

			BeforeEach(func() {

				ccWatcher, loginWatcher = setupStress()

				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id/processes",
					RoundRobinWithMultiple(
						RespondWith(http.StatusOK, appProcessesJson, http.Header{"Content-Type": []string{"application/json"}}),
						RespondWith(http.StatusOK, appProcessesJson, http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusNotFound, cf.CfResourceNotFound),
						RespondWith(http.StatusOK, appProcessesJson, http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
					))

				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id",
					RoundRobinWithMultiple(
						RespondWith(http.StatusOK, appTestJson, http.Header{"Content-Type": []string{"application/json"}}),
						RespondWith(http.StatusOK, appTestJson, http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusNotFound, cf.CfResourceNotFound),
						RespondWith(http.StatusOK, appTestJson, http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
					))
			})
			It("Should not leak file handles or cause errors", func() {

				apiCall := func(ctx context.Context, client cf.ContextClient) error {
					_, err := client.GetAppAndProcesses(ctx, "test-app-id")
					return err
				}

				numErrors := doStressTest(numberConcurrentUsers, numberSequentialPerUser, stats, apiCall)

				ccWatcher.printStats("CC states left")
				loginWatcher.printStats("login states left")

				Eventually(ccWatcher.Count()).WithTimeout(100 * time.Millisecond).Should(BeNumerically("<=", maxIdleConnsPerHost))
				Eventually(loginWatcher.Count()).WithTimeout(100 * time.Millisecond).Should(BeNumerically("<=", maxIdleConnsPerHost))

				stats.Report()
				//Each sequence is
				// - a login to the login server
				// - 2 parallel requests to the api server.
				numberOfConcurrentRequests := numberConcurrentUsers * 2
				numberOfRetriedRequests := (numberConcurrentUsers * numberSequentialPerUser) / 5
				Expect(ccWatcher.MaxOpenConnections()).To(BeNumerically("<=", numberOfConcurrentRequests+numberOfRetriedRequests), "maximum number of api connections in play at one time")
				Expect(loginWatcher.MaxOpenConnections()).To(BeNumerically("<=", 2*numberConcurrentUsers), "number of login connections open at one time")
				Expect(numErrors).To(Equal(int64(0)), "Number of errors received while under stress")
				//There are 2 different servers so there has to be 2 new and unused connections
				Expect(stats.GetReused()).To(BeNumerically(">=", stats.reUsed-int32(numberConcurrentUsers/2)), "Number of re-used connections")
			})
		})
	})

	Describe("ScaleAppWebProcess", func() {
		JustBeforeEach(func() {
			err = cfc.ScaleAppWebProcess("test-app-id", 6)
		})

		scaleResponse := LoadFile("scale_response.yml")
		When("scaling web app succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("POST", "/v3/apps/test-app-id/processes/web/actions/scale"),
						VerifyHeaderKV("Authorization", "Bearer test-access-token"),
						VerifyJSON(`{"instances":6}`),
						RespondWith(http.StatusAccepted, scaleResponse),
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
					RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError))
			})

			It("should error correctly", func() {
				Expect(err).To(MatchError(MatchRegexp("failed scaling app 'test-app-id' to 6: failed POST-ing cf.Process: POST request failed:.*'UnknownError'.*")))
			})
		})

		Context("Given a significant load", func() {

			numberConcurrentUsers := 100
			numberSequentialPerUser := 10
			var ccWatcher *ConnectionWatcher
			var loginWatcher *ConnectionWatcher
			stats := &reqStats{}

			BeforeEach(func() {
				ccWatcher, loginWatcher = setupStress()
				fakeCC.RouteToHandler("POST", "/v3/apps/test-app-id/processes/web/actions/scale",
					RoundRobinWithMultiple(
						RespondWith(http.StatusAccepted, scaleResponse, http.Header{"Content-Type": []string{"application/json"}}),
						RespondWith(http.StatusAccepted, scaleResponse, http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusNotFound, cf.CfResourceNotFound),
						RespondWith(http.StatusAccepted, scaleResponse, http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusInternalServerError, cf.CfInternalServerError),
					))
			})
			It("Should not leak file handles or cause errors", func() {

				apiCall := func(ctx context.Context, client cf.ContextClient) error {
					return client.ScaleAppWebProcess(ctx, "test-app-id", 6)
				}

				numErrors := doStressTest(numberConcurrentUsers, numberSequentialPerUser, stats, apiCall)

				ccWatcher.printStats("CC states left")
				loginWatcher.printStats("login states left")

				Eventually(ccWatcher.Count()).WithTimeout(100 * time.Millisecond).Should(BeNumerically("<=", maxIdleConnsPerHost))
				Eventually(loginWatcher.Count()).WithTimeout(100 * time.Millisecond).Should(BeNumerically("<=", maxIdleConnsPerHost))

				stats.Report()
				//Each sequence is
				// - a login to the login server
				// - 1 request to the api server.
				numberOfConcurrentRequests := numberConcurrentUsers
				numberOfRetriedRequests := (numberConcurrentUsers * numberSequentialPerUser) / 5
				Expect(ccWatcher.MaxOpenConnections()).To(BeNumerically("<=", numberOfConcurrentRequests+numberOfRetriedRequests), "maximum number of api connections in play at one time")
				Expect(loginWatcher.MaxOpenConnections()).To(BeNumerically("<=", 2*numberConcurrentUsers), "number of login connections open at one time")
				Expect(numErrors).To(Equal(int64(0)), "Number of errors received while under stress")
				//There are 2 different servers so there has to be 2 new and unused connections
				Expect(stats.GetReused()).To(BeNumerically(">=", stats.reUsed-int32(numberConcurrentUsers/2)), "Number of re-used connections")
			})
		})

	})

})
