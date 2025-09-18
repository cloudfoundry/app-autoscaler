package server_test

import (
	"strconv"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/config"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/server"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon_v2"

	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

var _ = Describe("Server", func() {
	var (
		serverUrl     *url.URL
		server        *Server
		serverProcess ifrit.Process

		conf *config.Config

		rsp        *http.Response
		req        *http.Request
		body       []byte
		err        error
		method     string
		bodyReader io.Reader
		route      = routes.NewRouter().CreateScalingEngineRoutes()

		scalingEngineDB    *fakes.FakeScalingEngineDB
		sychronizer        *fakes.FakeActiveScheduleSychronizer
		scalingEngine      *fakes.FakeScalingEngine
		policyDb           *fakes.FakePolicyDB
		schedulerDB        *fakes.FakeSchedulerDB
		xfccAuthMiddleware *fakes.FakeXFCCAuthMiddleware
	)

	BeforeEach(func() {
		conf = &config.Config{
			BaseConfig: configutil.BaseConfig{
				Server: helpers.ServerConfig{
					Port: 2222 + GinkgoParallelProcess(),
				},
				CFServer: helpers.ServerConfig{
					Port: 3333 + GinkgoParallelProcess(),
				},
			},
		}
		scalingEngineDB = &fakes.FakeScalingEngineDB{}
		scalingEngine = &fakes.FakeScalingEngine{}
		policyDb = &fakes.FakePolicyDB{}
		schedulerDB = &fakes.FakeSchedulerDB{}
		sychronizer = &fakes.FakeActiveScheduleSychronizer{}
		xfccAuthMiddleware = &fakes.FakeXFCCAuthMiddleware{}
		server = NewServer(lager.NewLogger("test"), conf, policyDb, scalingEngineDB, schedulerDB, scalingEngine, sychronizer)
	})

	AfterEach(func() {
		ginkgomon_v2.Interrupt(serverProcess)
	})

	JustBeforeEach(func() {
		req, err = http.NewRequest(method, serverUrl.String(), bodyReader)
		Expect(err).NotTo(HaveOccurred())
		rsp, err = http.DefaultClient.Do(req)
	})

	Describe("#CreateMTLSServer", func() {
		BeforeEach(func() {
			httpServer, err := server.CreateMtlsServer()
			Expect(err).NotTo(HaveOccurred())
			serverProcess = ginkgomon_v2.Invoke(httpServer)
			serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(conf.Server.Port))
			Expect(err).ToNot(HaveOccurred())
		})

		When("triggering scaling action", func() {
			BeforeEach(func() {
				body, err = json.Marshal(models.Trigger{Adjustment: "+1"})
				Expect(err).NotTo(HaveOccurred())

				bodyReader = bytes.NewReader(body)
				uPath, err := route.Get(routes.ScaleRouteName).URLPath("appid", "test-app-id")
				Expect(err).NotTo(HaveOccurred())
				serverUrl.Path = uPath.Path
			})

			When("requesting correctly", func() {
				BeforeEach(func() {
					method = http.MethodPost
				})

				It("should return 200", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					rsp.Body.Close()
				})
			})
		})

		When("GET /v1/liveness", func() {
			BeforeEach(func() {
				uPath, err := route.Get(routes.LivenessRouteName).URLPath()
				Expect(err).NotTo(HaveOccurred())
				method = http.MethodGet
				serverUrl.Path = uPath.Path
			})

			It("should return 200", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})
		})

		When("GET /v1/apps/{guid}/scaling_histories", func() {
			BeforeEach(func() {
				uPath, err := route.Get(routes.GetScalingHistoriesRouteName).URLPath("guid", "8ea70e4e-e0bc-4e15-9d32-cd69daaf012a")
				Expect(err).NotTo(HaveOccurred())
				method = http.MethodGet
				serverUrl.Path = uPath.Path
			})

			JustBeforeEach(func() {
				req, err = http.NewRequest(method, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())

			})

			It("should return 200", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})
		})

		Describe("PUT /v1/apps/{appid}/active_schedules/{scheduleid}", func() {
			BeforeEach(func() {
				uPath, err := route.Get(routes.SetActiveScheduleRouteName).URLPath("appid", "test-app-id", "scheduleid", "test-schedule-id")
				Expect(err).NotTo(HaveOccurred())
				serverUrl.Path = uPath.Path
				method = http.MethodPut
			})

			When("setting active schedule", func() {
				BeforeEach(func() {
					bodyReader = bytes.NewReader([]byte(`{"instance_min_count":1, "instance_max_count":5, "initial_min_instance_count":3}`))
				})

				When("credentials are correct", func() {

					It("should return 200", func() {
						Expect(err).ToNot(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusOK))
						rsp.Body.Close()
					})
				})

			})

			When("deleting active schedule", func() {
				BeforeEach(func() {
					uPath, err := route.Get(routes.DeleteActiveScheduleRouteName).URLPath("appid", "test-app-id", "scheduleid", "test-schedule-id")
					Expect(err).NotTo(HaveOccurred())
					serverUrl.Path = uPath.Path
					bodyReader = nil
					method = http.MethodDelete
				})

				When("requesting correctly", func() {
					It("should return 200", func() {
						Expect(err).ToNot(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusOK))
						rsp.Body.Close()
					})
				})
			})

			When("getting active schedule", func() {
				BeforeEach(func() {
					uPath, err := route.Get(routes.GetActiveSchedulesRouteName).URLPath("appid", "test-app-id")
					Expect(err).NotTo(HaveOccurred())
					serverUrl.Path = uPath.Path
					bodyReader = nil
					method = http.MethodGet
				})

				When("requesting correctly", func() {
					BeforeEach(func() {
						activeSchedule := &models.ActiveSchedule{
							ScheduleId:         "a-schedule-id",
							InstanceMin:        1,
							InstanceMax:        5,
							InstanceMinInitial: 3,
						}

						scalingEngineDB.GetActiveScheduleReturns(activeSchedule, nil)
					})

					It("should return 200", func() {
						Expect(err).ToNot(HaveOccurred())
						Expect(rsp.StatusCode).To(Equal(http.StatusOK))
						rsp.Body.Close()
					})
				})
			})
		})

		When("requesting sync shedule", func() {
			BeforeEach(func() {
				uPath, err := route.Get(routes.SyncActiveSchedulesRouteName).URLPath()
				Expect(err).NotTo(HaveOccurred())
				serverUrl.Path = uPath.Path
				bodyReader = nil
			})

			When("requesting correctly", func() {
				BeforeEach(func() {
					method = http.MethodPut
				})

				It("should return 200", func() {
					Eventually(sychronizer.SyncCallCount).Should(Equal(1))
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					rsp.Body.Close()
				})
			})

			When("requesting with incorrect http method", func() {
				BeforeEach(func() {
					method = http.MethodGet
				})

				It("should return 405", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
					rsp.Body.Close()
				})
			})
		})

		DescribeTable("when requesting non existing path", func(method string) {
			serverUrl.Path = "/not-exist"
			req, err = http.NewRequest(method, serverUrl.String(), bodyReader)
			Expect(err).NotTo(HaveOccurred())
			req.Method = method
			rsp, err = http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
			rsp.Body.Close()
		},
			Entry("PUT /not-exist", http.MethodPut),
			Entry("GET /not-exist", http.MethodGet),
		)
	})

	Describe("#CreateCFServer", func() {
		BeforeEach(func() {
			xfccAuthMiddleware.XFCCAuthenticationMiddlewareReturns(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.RequestURI, "invalid-guid") {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}))
			httpServer, err := server.CreateCFServer(xfccAuthMiddleware)
			Expect(err).NotTo(HaveOccurred())
			serverProcess = ginkgomon_v2.Invoke(httpServer)

			serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(conf.CFServer.Port))
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("GET /v1/apps/{appid}/scaling_histories", func() {
			BeforeEach(func() {
			})

			Describe("when XFCC authentication is ok", func() {
				BeforeEach(func() {
					uPath, err := route.Get(routes.GetScalingHistoriesRouteName).URLPath("guid", "valid-guid")
					Expect(err).NotTo(HaveOccurred())
					serverUrl.Path = uPath.Path
					method = http.MethodGet
				})

				It("should return 200", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					rsp.Body.Close()
				})
			})

			Describe("when XFCC authentication fails", func() {
				BeforeEach(func() {
					uPath, err := route.Get(routes.GetScalingHistoriesRouteName).URLPath("guid", "invalid-guid")
					Expect(err).NotTo(HaveOccurred())
					serverUrl.Path = uPath.Path
					method = http.MethodGet
				})

				It("should return 401", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
					rsp.Body.Close()
				})

			})
		})
	})
})
