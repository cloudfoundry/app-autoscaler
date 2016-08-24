package server_test

import (
	"engine/fakes"
	. "engine/server"
	"models"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
)

var _ = Describe("Server", func() {

	var (
		server    ifrit.Process
		serverUrl *url.URL
		rsp       *http.Response
		cfc       *fakes.FakeCfClient
		database  *fakes.FakePolicyDB
		body      []byte
		err       error
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		database = &fakes.FakePolicyDB{}
		httpServer := NewServer(lager.NewLogger("test"), cfc, database)
		serverUrl, _ = url.Parse("http://127.0.0.1:8080")
		server = ginkgomon.Invoke(httpServer)
		body, err = json.Marshal(models.Trigger{Adjustment: "+1"})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		ginkgomon.Interrupt(server)
	})

	JustBeforeEach(func() {
		rsp, err = http.Post(serverUrl.String(), "application/json", bytes.NewReader(body))
	})

	Context("when trigger scaling action", func() {
		BeforeEach(func() {
			serverUrl.Path = PathScalingActions
			database.GetAppPolicyReturns(&models.ScalingPolicy{}, nil)
		})

		It("should return 201", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusCreated))
		})
	})

	Context("when requesting the wrong path", func() {
		BeforeEach(func() {
			serverUrl.Path = "/not-exist-path"
		})

		It("should return 404", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Context("when requesting the wrong method", func() {
		JustBeforeEach(func() {
			rsp, err = http.Post(serverUrl.String(), "unknown", nil)
		})

		It("should return 404", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

})
