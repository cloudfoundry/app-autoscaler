package server_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const TestPathAggregatedMetricHistories = "/v1/apps/an-app-id/aggregated_metric_histories/a-metric-type"

var _ = Describe("Server", func() {
	var (
		rsp *http.Response
		err error
	)

	Context("when retrieving aggregared metrics history", func() {
		BeforeEach(func() {
			serverUrl.Path = TestPathAggregatedMetricHistories
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

	Context("when using wrong method to retrieve aggregared metrics history", func() {
		BeforeEach(func() {
			serverUrl.Path = TestPathAggregatedMetricHistories
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
