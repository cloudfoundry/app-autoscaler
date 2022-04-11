package collector_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"net/http"
)

const TestPathMetricHistories = "/v1/apps/an-app-id/metric_histories/a-metric-type"

var _ = Describe("Server", func() {
	var (
		rsp *http.Response
		err error
	)

	Context("when retrieving metrics history", func() {
		BeforeEach(func() {
			serverUrl.Path = TestPathMetricHistories
		})

		JustBeforeEach(func() {
			rsp, err = http.Get(serverUrl.String())
		})

		It("should return 200", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			rsp.Body.Close()
		})
	})

	Context("when requesting the wrong path", func() {
		BeforeEach(func() {
			serverUrl.Path = "/not-exist-path"
		})

		JustBeforeEach(func() {
			rsp, err = http.Get(serverUrl.String())
		})

		It("should return 404", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
			rsp.Body.Close()
		})
	})

	Context("when using wrong method to retrieve metric histories", func() {
		BeforeEach(func() {
			serverUrl.Path = TestPathMetricHistories
		})

		JustBeforeEach(func() {
			rsp, err = http.Post(serverUrl.String(), "garbage", nil)
		})

		It("should return 405", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
			rsp.Body.Close()
		})
	})

})
