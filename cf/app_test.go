package cf_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"
)

const maxIdleConnsPerHost = 200

var _ = Describe("Cf client App", func() {
	BeforeEach(login)

	Describe("GetApp", func() {
		When("get app succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/apps/test-app-id"),
						VerifyHeaderKV("Authorization", "Bearer test-access-token"),
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

	Describe("GetAppAndProcesses", func() {

		When("get app & process return ok", func() {
			BeforeEach(func() {
				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id/processes",
					RoundRobinWithMultiple(
						RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusNotFound, models.CfResourceNotFound),
						RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
					))

				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id",
					RoundRobinWithMultiple(
						RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusNotFound, models.CfResourceNotFound),
						RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
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

		Context("Given a significant load", func() {
			idleUsed := int32(0)
			reUsed := int32(0)
			numRequests := int32(0)
			numResponses := int32(0)
			numberConcurrentUsers := 100
			numberSequentialPerUser := 10
			/*
			 * Note there is a login that goes to a separate host then 2 concurrent requests to the api server.
			 */
			doAppProcessRequest := func() int64 {
				clientTrace := &httptrace.ClientTrace{
					GotConn: func(info httptrace.GotConnInfo) {
						if info.WasIdle {
							atomic.AddInt32(&idleUsed, 1)
						}
						if info.Reused {
							atomic.AddInt32(&reUsed, 1)
						}
					},
					WroteRequest: func(info httptrace.WroteRequestInfo) {
						atomic.AddInt32(&numRequests, 1)
					},
					GotFirstResponseByte: func() {
						atomic.AddInt32(&numResponses, 1)
					},
				}
				traceCtx := httptrace.WithClientTrace(context.Background(), clientTrace)
				numErr := 0
				var cfError = &models.CfError{}
				anErr := cfc.Login()
				if anErr != nil && !errors.As(anErr, &cfError) {
					GinkgoWriter.Printf(" Error: %+v\n", anErr)
					numErr += 1
				}
				_, anErr = cfc.GetCtxClient().GetAppAndProcesses(traceCtx, "test-app-id")

				if anErr != nil && !errors.As(anErr, &cfError) {
					GinkgoWriter.Printf(" Error: %+v\n", anErr)
					numErr += 1
				}
				return int64(numErr)
			}
			BeforeEach(func() {
				// quiet logger
				logger = lager.NewLogger("cf")
				setCfcClient(2)

				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id/processes",
					RoundRobinWithMultiple(
						RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusNotFound, models.CfResourceNotFound),
						RespondWith(http.StatusOK, LoadFile("testdata/app_processes.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
					))

				fakeCC.RouteToHandler("GET", "/v3/apps/test-app-id",
					RoundRobinWithMultiple(
						RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusNotFound, models.CfResourceNotFound),
						RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
						RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
					))
			})
			It("Should not leak file handles or cause errors", func() {
				ccWatcher := NewConnectionWatcher(fakeCC.HTTPTestServer.Config.ConnState)
				loginWatcher := NewConnectionWatcher(fakeLoginServer.HTTPTestServer.Config.ConnState)
				fakeCC.HTTPTestServer.Config.ConnState = ccWatcher.OnStateChange
				fakeLoginServer.HTTPTestServer.Config.ConnState = loginWatcher.OnStateChange

				var numErrors int64 = 0

				wg := sync.WaitGroup{}
				for i := 0; i < numberConcurrentUsers; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for i := 0; i < numberSequentialPerUser; i++ {
							atomic.AddInt64(&numErrors, doAppProcessRequest())
						}
					}()
				}
				wg.Wait()

				GinkgoWriter.Printf("\n# CC states left\n")
				for key, value := range ccWatcher.GetStates() {
					GinkgoWriter.Printf("\t%s:\t%d\n", key, value)
				}

				GinkgoWriter.Printf("\n# login states left\n")
				for key, value := range loginWatcher.GetStates() {
					GinkgoWriter.Printf("\t%s:\t%d\n", key, value)
				}
				Eventually(ccWatcher.Count()).WithTimeout(100 * time.Millisecond).Should(BeNumerically("<=", maxIdleConnsPerHost))
				Eventually(loginWatcher.Count()).WithTimeout(100 * time.Millisecond).Should(BeNumerically("<=", maxIdleConnsPerHost))

				GinkgoWriter.Printf("\n# Client trace stats\n")
				GinkgoWriter.Printf("\tidle pool connections used:\t%d\n", atomic.LoadInt32(&idleUsed))
				GinkgoWriter.Printf("\tconnections re-used:\t\t%d\n", atomic.LoadInt32(&reUsed))
				GinkgoWriter.Printf("\tnumber of requests:\t\t\t%d\n", atomic.LoadInt32(&numRequests))
				GinkgoWriter.Printf("\tnumber of responses:\t\t\t%d\n", atomic.LoadInt32(&numResponses))

				//Each sequence is
				// - a login to the login server
				// - 2 parallel requests to the api server.
				numberOfConcurrentRequests := numberConcurrentUsers * 2
				numberOfRetriedRequests := (numberConcurrentUsers * numberSequentialPerUser) / 5
				Expect(ccWatcher.MaxOpenConnections()).To(BeNumerically("<=", numberOfConcurrentRequests+numberOfRetriedRequests), "maximum number of api connections in play at one time")
				Expect(loginWatcher.MaxOpenConnections()).To(BeNumerically("<=", 2*numberConcurrentUsers), "number of login connections open at one time")
				Expect(numErrors).To(Equal(int64(0)), "Number of errors received while under stress")
				//There are 2 different servers so there has to be 2 new and unused connections
				Expect(reUsed).To(BeNumerically(">=", numRequests-2), "Number of re-used connections")
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
						VerifyHeaderKV("Authorization", "Bearer test-access-token"),
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
				Expect(err).To(MatchError(MatchRegexp("failed scaling app 'test-app-id' to 6: POST request failed:.*'UnknownError'.*")))
			})
		})

	})

})
