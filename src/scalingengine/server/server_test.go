package server_test

import (
	"models"
	"scalingengine/fakes"
	. "scalingengine/server"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"bytes"
	"encoding/json"
	"net/http"
)

const serverURL = "http://127.0.0.1:8080"

var server ifrit.Process

var _ = BeforeSuite(func() {
	cfc := &fakes.FakeCfClient{}
	policyDB := &fakes.FakePolicyDB{}
	policyDB.GetAppPolicyReturns(&models.ScalingPolicy{}, nil)
	historyDB := &fakes.FakeHistoryDB{}
	httpServer := NewServer(lager.NewLogger("test"), cfc, policyDB, historyDB)
	server = ginkgomon.Invoke(httpServer)
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
		JustBeforeEach(func() {
			rsp, err = http.Post(serverURL+urlPath, "application/json", bytes.NewReader(body))
		})

		BeforeEach(func() {
			route := mux.Route{}
			uPath, err := route.Path(PathScale).URLPath("appid", "test-app-id")
			Expect(err).NotTo(HaveOccurred())
			urlPath = uPath.Path
		})

		It("should return 200", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
		})

		Context("when requesting the wrong path", func() {
			BeforeEach(func() {
				urlPath = "/not-exist-path"
			})

			It("should return 404", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
			})
		})

		Context("when using the wrong method", func() {
			JustBeforeEach(func() {
				rsp, err = http.Get(serverURL + urlPath)
			})

			It("should return 404", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
			})
		})

	})

	Context("when getting scaling histories", func() {
		JustBeforeEach(func() {
			rsp, err = http.Get(serverURL + urlPath)
		})

		BeforeEach(func() {
			route := mux.Route{}
			uPath, err := route.Path(PathScalingHistories).URLPath("appid", "test-app-id")
			Expect(err).NotTo(HaveOccurred())
			urlPath = uPath.Path
		})

		It("should return 200", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
		})

		Context("when requesting the wrong path", func() {
			BeforeEach(func() {
				urlPath = "/not-exist-path"
			})

			It("should return 404", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
			})
		})

		Context("when using the wrong method", func() {
			JustBeforeEach(func() {
				rsp, err = http.Post(serverURL+urlPath, "gabage", nil)
			})

			It("should return 404", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
			})
		})

	})

})
