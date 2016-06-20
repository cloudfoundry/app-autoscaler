package server_test

import (
	"metrics-collector/config"
	. "metrics-collector/server"
	"metrics-collector/server/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"

	"net/http"
	"net/url"
)

const TEST_PATH_MEMORY_METRICS = "/v1/apps/an-app-id/metrics/memory"

var _ = Describe("Server", func() {
	var (
		cfc       *fakes.FakeCfClient
		consumer  *fakes.FakeNoaaConsumer
		server    *Server
		serverUrl *url.URL
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		consumer = &fakes.FakeNoaaConsumer{}

		logger := lager.NewLogger("server-test")
		conf := config.ServerConfig{Port: 1111}

		var err error
		serverUrl, err = url.Parse("http://127.0.0.1:1111" + TEST_PATH_MEMORY_METRICS)
		Expect(err).ToNot(HaveOccurred())

		server = NewServer(logger, conf, cfc, consumer)
	})

	AfterEach(func() {
		server.Stop()
	})

	It("does not accept requests", func() {
		_, err := http.Get(serverUrl.String())
		Expect(err).To(HaveOccurred())
	})

	Describe("Start", func() {
		var (
			rsp *http.Response
			err error
		)

		BeforeEach(func() {
			go server.Start()
			Eventually(func() error {
				_, err := http.Get(serverUrl.String())
				return err
			}).ShouldNot(HaveOccurred())
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

	Describe("Stop", func() {
		BeforeEach(func() {
			server.Stop()
		})

		It("no longer listens to requests", func() {
			_, err := http.Get(serverUrl.String())
			Expect(err).To(HaveOccurred())
		})

		It("can be stopped twice", func() {
			Expect(func() { server.Stop() }).NotTo(Panic())
		})
	})
})
