package ratelimiter_test

import (
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager/v3/lagertest"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"
)

var _ = Describe("RateLimiterMiddleware", func() {
	var (
		req         *http.Request
		resp        *httptest.ResponseRecorder
		router      *mux.Router
		rateLimiter *fakes.FakeLimiter
		rlmw        *ratelimiter.RateLimiterMiddleware
	)

	Describe("CheckRateLimit", func() {
		BeforeEach(func() {
			rateLimiter = &fakes.FakeLimiter{}
			rlmw = ratelimiter.NewRateLimiterMiddleware("key", rateLimiter, lagertest.NewTestLogger("ratelimiter-middleware"))
			router = mux.NewRouter()
			router.HandleFunc("/", GetTestHandler())
			router.HandleFunc("/ratelimit/{key}/path", GetTestHandler())
			router.HandleFunc("/ratelimit/anotherpath", GetTestHandler())
			router.Use(rlmw.CheckRateLimit)

			resp = httptest.NewRecorder()
		})

		JustBeforeEach(func() {
			router.ServeHTTP(resp, req)
		})

		Context("without key in the url", func() {
			BeforeEach(func() {
				rateLimiter.ExceedsLimitReturns(true)
				req = httptest.NewRequest(http.MethodGet, "/ratelimit/anotherpath", nil)
			})
			It("should succeed with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Missing rate limit key"}`))
			})
		})
		Context("exceed rate limiting", func() {
			BeforeEach(func() {
				rateLimiter.ExceedsLimitReturns(true)
				req = httptest.NewRequest(http.MethodGet, "/ratelimit/MY-KEY/path", nil)
			})
			It("should succeed with 429", func() {
				Expect(resp.Code).To(Equal(http.StatusTooManyRequests))
				Expect(resp.Body.String()).To(Equal(`{"code":"Request-Limit-Exceeded","message":"Too many requests"}`))
			})
		})
		Context("below rate limiting", func() {
			BeforeEach(func() {
				rateLimiter.ExceedsLimitReturns(false)
				req = httptest.NewRequest(http.MethodGet, "/ratelimit/MY-KEY/path", nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})
	})

})

func GetTestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Success"))
		Expect(err).NotTo(HaveOccurred())
	}
}
