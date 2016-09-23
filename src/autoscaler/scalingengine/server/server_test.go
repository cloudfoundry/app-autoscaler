package server_test

import (
	"autoscaler/models"
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
	"net/http"
)

var server ifrit.Process
var serverUrl string

var _ = BeforeSuite(func() {
	conf := config.ServerConfig{Port: 8080}
	cfc := &fakes.FakeCfClient{}
	policyDB := &fakes.FakePolicyDB{}
	policyDB.GetAppPolicyReturns(&models.ScalingPolicy{}, nil)
	historyDB := &fakes.FakeHistoryDB{}
	httpServer := NewServer(lager.NewLogger("test"), conf, cfc, policyDB, historyDB)
	server = ginkgomon.Invoke(httpServer)
	serverUrl = fmt.Sprintf("http://127.0.0.1:%d", conf.Port)
})

var _ = AfterSuite(func() {
	ginkgomon.Interrupt(server)
})

var _ = Describe("Server", func() {
	var (
		urlPath string
		rsp     *http.Response
		body    []byte
		err     error
	)

	BeforeEach(func() {
		body, err = json.Marshal(models.Trigger{Adjustment: "+1"})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when triggering scaling action", func() {
		BeforeEach(func() {
			route := mux.Route{}
			uPath, err := route.Path(PathScale).URLPath("appid", "test-app-id")
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
			route := mux.Route{}
			uPath, err := route.Path(PathScalingHistories).URLPath("appid", "test-app-id")
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

})
