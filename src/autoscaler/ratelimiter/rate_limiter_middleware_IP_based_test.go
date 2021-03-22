package ratelimiter_test

import (
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"autoscaler/fakes"
	"autoscaler/ratelimiter"
)

var _ = Describe("RateLimiterMiddlewareHealth", func() {
	var (
		req         *http.Request
		resp        *httptest.ResponseRecorder
		router      *mux.Router
		rateLimiter *fakes.FakeLimiter
		rlmw        *ratelimiter.RateLimiterMiddlewareIPBased
	)

	Describe("CheckRateLimit", func() {
		BeforeEach(func() {
			rateLimiter = &fakes.FakeLimiter{}
			rlmw = ratelimiter.NewRateLimiterMiddlewareIPBased(rateLimiter, lagertest.NewTestLogger("ratelimiter-middleware"))
			router = mux.NewRouter()
			router.HandleFunc("/", GetTestHandler())
			router.HandleFunc("/ratelimit/path", GetTestHandler())
			router.HandleFunc("/ratelimit/anotherpath", GetTestHandler())
			router.Use(rlmw.CheckRateLimit)

			resp = httptest.NewRecorder()
		})

		JustBeforeEach(func() {
			router.ServeHTTP(resp, req)
		})

		Context("exceed rate limiting", func() {
			BeforeEach(func() {
				rateLimiter.ExceedsLimitReturns(true)
				req = httptest.NewRequest(http.MethodGet, "/ratelimit/path", nil)
			})
			It("should succeed with 429", func() {
				Expect(resp.Code).To(Equal(http.StatusTooManyRequests))
				Expect(resp.Body.String()).To(Equal(`{"code":"Request-Limit-Exceeded","message":"Too many requests"}`))
			})
		})
		Context("below rate limiting", func() {
			BeforeEach(func() {
				rateLimiter.ExceedsLimitReturns(false)
				req = httptest.NewRequest(http.MethodGet, "/ratelimit/path", nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("exceed rate limiting with X-Forwarded-For set in the header of the request", func() {
			BeforeEach(func() {
				rateLimiter.ExceedsLimitReturns(true)
				req = httptest.NewRequest(http.MethodGet, "/ratelimit/path", nil)
				req.Header.Set("X-Forwarded-For", "0.0.0.0, 0.0.0.1, 0.0.0.2")
			})
			It("should succeed with 429", func() {
				Expect(resp.Code).To(Equal(http.StatusTooManyRequests))
				Expect(resp.Body.String()).To(Equal(`{"code":"Request-Limit-Exceeded","message":"Too many requests"}`))
				Expect(rateLimiter.ExceedsLimitArgsForCall(0)).To(Equal("0.0.0.0"))
			})
		})
	})

})
