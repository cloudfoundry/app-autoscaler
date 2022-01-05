package server_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/config"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/server"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon_v2"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var (
	server              ifrit.Process
	serverUrl           string
	scalingEngineDB     *fakes.FakeScalingEngineDB
	sychronizer         *fakes.FakeActiveScheduleSychronizer
	httpStatusCollector *fakes.FakeHTTPStatusCollector
)

var _ = SynchronizedBeforeSuite(func() []byte {
	return nil
}, func(_ []byte) {
	port := 2222 + GinkgoParallelProcess()
	conf := &config.Config{
		Server: config.ServerConfig{
			Port: port,
		},
	}
	scalingEngineDB = &fakes.FakeScalingEngineDB{}
	scalingEngine := &fakes.FakeScalingEngine{}
	sychronizer = &fakes.FakeActiveScheduleSychronizer{}
	httpStatusCollector = &fakes.FakeHTTPStatusCollector{}

	httpServer, err := NewServer(lager.NewLogger("test"), conf, scalingEngineDB, scalingEngine, sychronizer, httpStatusCollector)
	Expect(err).NotTo(HaveOccurred())
	server = ginkgomon_v2.Invoke(httpServer)
	serverUrl = fmt.Sprintf("http://127.0.0.1:%d", conf.Server.Port)
})

var _ = SynchronizedAfterSuite(func() {
	ginkgomon_v2.Interrupt(server)
}, func() {
})

var _ = Describe("Server", func() {
	var (
		urlPath    string
		rsp        *http.Response
		req        *http.Request
		body       []byte
		err        error
		method     string
		bodyReader io.Reader
		route      = routes.ScalingEngineRoutes()
	)

	BeforeEach(func() {

	})

	Context("when triggering scaling action", func() {
		BeforeEach(func() {
			body, err = json.Marshal(models.Trigger{Adjustment: "+1"})
			Expect(err).NotTo(HaveOccurred())

			uPath, err := route.Get(routes.ScaleRouteName).URLPath("appid", "test-app-id")
			Expect(err).NotTo(HaveOccurred())
			urlPath = uPath.Path
		})

		Context("when requesting correctly", func() {
			JustBeforeEach(func() {
				rsp, err = http.Post(serverUrl+urlPath, "application/json", bytes.NewReader(body))
			})

			It("should return 200", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})
		})

		Context("when requesting the wrong path", func() {
			JustBeforeEach(func() {
				rsp, err = http.Post(serverUrl+"/not-exist-path", "application/json", bytes.NewReader(body))
			})

			It("should return 404", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
				rsp.Body.Close()
			})
		})

	})

	Context("when getting scaling histories", func() {
		BeforeEach(func() {
			uPath, err := route.Get(routes.GetScalingHistoriesRouteName).URLPath("appid", "test-app-id")
			Expect(err).NotTo(HaveOccurred())
			urlPath = uPath.Path
		})

		Context("when requesting correctly", func() {
			JustBeforeEach(func() {
				rsp, err = http.Get(serverUrl + urlPath)
			})

			It("should return 200", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})
		})

		Context("when requesting the wrong path", func() {
			JustBeforeEach(func() {
				rsp, err = http.Get(serverUrl + "/not-exist-path")
			})

			It("should return 404", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
				rsp.Body.Close()
			})
		})
	})

	Context("when requesting active shedule", func() {

		JustBeforeEach(func() {
			req, err = http.NewRequest(method, serverUrl+urlPath, bodyReader)
			Expect(err).NotTo(HaveOccurred())
			rsp, err = http.DefaultClient.Do(req)
		})

		Context("when setting active schedule", func() {
			BeforeEach(func() {
				uPath, err := route.Get(routes.SetActiveScheduleRouteName).URLPath("appid", "test-app-id", "scheduleid", "test-schedule-id")
				Expect(err).NotTo(HaveOccurred())
				urlPath = uPath.Path
				bodyReader = bytes.NewReader([]byte(`{"instance_min_count":1, "instance_max_count":5, "initial_min_instance_count":3}`))
			})

			Context("when requesting correctly", func() {
				BeforeEach(func() {
					method = http.MethodPut
				})

				It("should return 200", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					rsp.Body.Close()
				})
			})

			Context("when requesting the wrong path", func() {
				BeforeEach(func() {
					method = http.MethodPut
					urlPath = "/not-exist"
				})

				It("should return 404", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
					rsp.Body.Close()
				})
			})
		})

		Context("when deleting active schedule", func() {
			BeforeEach(func() {
				uPath, err := route.Get(routes.DeleteActiveScheduleRouteName).URLPath("appid", "test-app-id", "scheduleid", "test-schedule-id")
				Expect(err).NotTo(HaveOccurred())
				urlPath = uPath.Path
				bodyReader = nil
				method = http.MethodDelete
			})
			Context("when requesting correctly", func() {
				It("should return 200", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					rsp.Body.Close()
				})
			})

			Context("when requesting the wrong path", func() {
				BeforeEach(func() {
					urlPath = "/not-exist"
				})

				It("should return 404", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
					rsp.Body.Close()
				})
			})
		})

		Context("when getting active schedule", func() {
			BeforeEach(func() {
				uPath, err := route.Get(routes.GetActiveSchedulesRouteName).URLPath("appid", "test-app-id")
				Expect(err).NotTo(HaveOccurred())
				urlPath = uPath.Path
				bodyReader = nil
				method = http.MethodGet
			})

			Context("when requesting correctly", func() {
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

	Context("when requesting sync shedule", func() {
		JustBeforeEach(func() {
			uPath, err := route.Get(routes.SyncActiveSchedulesRouteName).URLPath()
			Expect(err).NotTo(HaveOccurred())
			urlPath = uPath.Path
			bodyReader = nil

			req, err = http.NewRequest(method, serverUrl+urlPath, bodyReader)
			Expect(err).NotTo(HaveOccurred())
			rsp, err = http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when requesting correctly", func() {
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

		Context("when requesting with incorrect http method", func() {
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
})
