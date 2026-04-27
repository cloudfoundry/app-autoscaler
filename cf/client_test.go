package cf_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/lager/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RetryClient", func() {
	var (
		logger lager.Logger
	)

	BeforeEach(func() {
		logger = lager.NewLogger("retry-client-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
	})

	It("does not retry by default (zero-value config)", func() {
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		retryHTTPClient := cf.RetryClient(cf.ClientConfig{}, server.Client(), logger)

		resp, err := retryHTTPClient.Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
		// No retries: exactly 1 request
		Expect(requestCount.Load()).To(Equal(int32(1)))
	})

	It("preserves RetryWaitMax default when not configured", func() {
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		// MaxRetries set but MaxRetryWaitMs left at zero — should preserve library default (30s)
		retryHTTPClient := cf.RetryClient(cf.ClientConfig{MaxRetries: 1}, server.Client(), logger)

		resp, err := retryHTTPClient.Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		// 1 initial + 1 retry = 2 total
		Expect(requestCount.Load()).To(Equal(int32(2)))
	})

	It("respects explicit MaxRetries config", func() {
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		retryHTTPClient := cf.RetryClient(cf.ClientConfig{MaxRetries: 2}, server.Client(), logger)

		resp, err := retryHTTPClient.Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		// 1 initial + 2 retries = 3 total
		Expect(requestCount.Load()).To(Equal(int32(3)))
	})
})
