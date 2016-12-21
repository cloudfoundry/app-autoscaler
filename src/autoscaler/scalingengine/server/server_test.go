package server_test

import (
	"autoscaler/models"
	"autoscaler/routes"
	"autoscaler/scalingengine/config"
	"autoscaler/scalingengine/fakes"
	. "autoscaler/scalingengine/server"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var server ifrit.Process
var serverUrl string

var _ = SynchronizedBeforeSuite(func() []byte {
	return nil
}, func(_ []byte) {
	port := 2222 + GinkgoParallelNode()
	conf := &config.Config{
		Server: config.ServerConfig{
			Port: port,
		},
	}
	scalingEngineDB := &fakes.FakeScalingEngineDB{}
	scalingEngine := &fakes.FakeScalingEngine{}
	httpServer, err := NewServer(lager.NewLogger("test"), conf, scalingEngineDB, scalingEngine)
	Expect(err).NotTo(HaveOccurred())
	server = ginkgomon.Invoke(httpServer)
	serverUrl = fmt.Sprintf("http://127.0.0.1:%d", conf.Server.Port)
})

var _ = SynchronizedAfterSuite(func() {
	ginkgomon.Interrupt(server)
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
		route      *mux.Router = routes.ScalingEngineRoutes()
	)

	Context("when triggering scaling action", func() {
		BeforeEach(func() {
			body, err = json.Marshal(models.Trigger{Adjustment: "+1"})
			Expect(err).NotTo(HaveOccurred())

			uPath, err := route.Get(routes.ScaleRoute).URLPath("appid", "test-app-id")
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

		Context("when using the wrong method", func() {
			JustBeforeEach(func() {
				rsp, err = http.Get(serverUrl + urlPath)
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
			uPath, err := route.Get(routes.HistoreisRoute).URLPath("appid", "test-app-id")
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

		Context("when using the wrong method", func() {
			JustBeforeEach(func() {
				rsp, err = http.Post(serverUrl+urlPath, "gabage", nil)
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
				uPath, err := route.Get(routes.UpdateActiveSchedulesRoute).URLPath("appid", "test-app-id", "scheduleid", "test-schedule-id")
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

			Context("when using the wrong method", func() {
				BeforeEach(func() {
					method = http.MethodPost
				})

				It("should return 404", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
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
				uPath, err := route.Get(routes.DeleteActiveSchedulesRoute).URLPath("appid", "test-app-id", "scheduleid", "test-schedule-id")
				Expect(err).NotTo(HaveOccurred())
				urlPath = uPath.Path
				bodyReader = nil
				method = http.MethodDelete
			})
			Context("when requesting correctly", func() {
				It("should return 204", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusNoContent))
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
	})
})
