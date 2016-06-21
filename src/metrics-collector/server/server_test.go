package server_test

import (
	"metrics-collector/config"
	. "metrics-collector/server"
	"metrics-collector/server/fakes"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"net/http"
	"net/url"
)

const TEST_PATH_MEMORY_METRICS = "/v1/apps/an-app-id/metrics/memory"

var _ = Describe("Server", func() {
	var (
		server    ifrit.Process
		serverUrl *url.URL
		rsp       *http.Response
		err       error
	)

	BeforeEach(func() {
		port := 1111
		cfc := &fakes.FakeCfClient{}
		consumer := &fakes.FakeNoaaConsumer{}
		conf := config.ServerConfig{Port: port}
		httpServer := NewServer(lager.NewLogger("test"), conf, cfc, consumer)
		serverUrl, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(port) + TEST_PATH_MEMORY_METRICS)
		Expect(err).ToNot(HaveOccurred())

		server = ginkgomon.Invoke(httpServer)
	})

	AfterEach(func() {
		ginkgomon.Interrupt(server)
	})

	JustBeforeEach(func() {
		rsp, err = http.Get(serverUrl.String())
	})

	It("retrieves metrics", func() {
		Expect(err).ToNot(HaveOccurred())
		Expect(rsp.StatusCode).To(Equal(http.StatusOK))
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
			rsp, err = http.Post(serverUrl.String(), "garbage", nil)
		})

		It("should return 404", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})
