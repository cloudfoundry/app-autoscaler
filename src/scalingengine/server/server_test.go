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
	database := &fakes.FakePolicyDB{}
	database.GetAppPolicyReturns(&models.ScalingPolicy{}, nil)
	httpServer := NewServer(lager.NewLogger("test"), cfc, database)
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

	JustBeforeEach(func() {
		rsp, err = http.Post(serverURL+urlPath, "application/json", bytes.NewReader(body))
	})

	Context("when trigger scaling action", func() {
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

	Context("when requesting the wrong path", func() {
		JustBeforeEach(func() {
			rsp, err = http.Post(serverURL, "unknown", nil)
		})

		It("should return 404", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

})
