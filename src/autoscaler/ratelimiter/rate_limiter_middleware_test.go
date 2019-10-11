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
	"autoscaler/routes"
)

var _ = Describe("RateLimiterMiddleware", func() {
	var (
		req         *http.Request
		resp        *httptest.ResponseRecorder
		router      *mux.Router
		rateLimiter *fakes.FakeLimiter
		rlmw        *ratelimiter.RateLimiterMiddleware
	)
	const (
		TEST_APP_ID = "test-app-id"
	)
	
	Describe("CheckRateLimit on metricsforwarder", func() {
		BeforeEach(func() {
			rateLimiter = &fakes.FakeLimiter{}
			rlmw = ratelimiter.NewRateLimiterMiddlewareWithLimiter("appid", rateLimiter, lagertest.NewTestLogger("ratelimiter-middleware"))
			router = mux.NewRouter()
			router.HandleFunc("/", GetTestHandler())
			router.HandleFunc(routes.CustomMetricsPath, GetTestHandler())
			router.Use(rlmw.CheckRateLimit)

			resp = httptest.NewRecorder()
		})

		JustBeforeEach(func() {
			router.ServeHTTP(resp, req)
		})

		Context("metrics api", func() {
			Context("exceed rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(true)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metrics", nil)
				})
				It("should succeed with 429", func() {
					Expect(resp.Code).To(Equal(http.StatusTooManyRequests))
					Expect(resp.Body.String()).To(Equal(`{"code":"Request-Limit-Exceeded","message":"Too many requests"}`))
				})
			})
			Context("below rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(false)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metrics", nil)
				})
				It("should succeed with 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
				})
			})
		})
	})
	
	Describe("CheckRateLimit on golangapiserver", func() {
		BeforeEach(func() {
			rateLimiter = &fakes.FakeLimiter{}
			rlmw = ratelimiter.NewRateLimiterMiddlewareWithLimiter("appId", rateLimiter, lagertest.NewTestLogger("ratelimiter-middleware"))
			router = mux.NewRouter()
			router.HandleFunc("/", GetTestHandler())
			router.HandleFunc(routes.PublicApiPolicyPath, GetTestHandler())
			router.HandleFunc(routes.PublicApiCredentialPath, GetTestHandler())
			router.HandleFunc("/v1/apps" + routes.PublicApiMetricsHistoryPath, GetTestHandler())
			router.HandleFunc("/v1/apps" + routes.PublicApiAggregatedMetricsHistoryPath, GetTestHandler())
			router.HandleFunc("/v1/apps" + routes.PublicApiScalingHistoryPath, GetTestHandler())
			router.Use(rlmw.CheckRateLimit)

			resp = httptest.NewRecorder()
		})

		JustBeforeEach(func() {
			router.ServeHTTP(resp, req)
		})

		Context("policy api", func() {
			Context("exceed rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(true)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/policy", nil)
				})
				It("should succeed with 429", func() {
					Expect(resp.Code).To(Equal(http.StatusTooManyRequests))
					Expect(resp.Body.String()).To(Equal(`{"code":"Request-Limit-Exceeded","message":"Too many requests"}`))
				})
			})
			Context("below rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(false)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/policy", nil)
				})
				It("should succeed with 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
				})
			})
		})

		Context("instance metrics api", func() {
			Context("exceed rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(true)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/cpu", nil)
				})
				It("should succeed with 429", func() {
					Expect(resp.Code).To(Equal(http.StatusTooManyRequests))
					Expect(resp.Body.String()).To(Equal(`{"code":"Request-Limit-Exceeded","message":"Too many requests"}`))
				})
			})
			Context("below rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(false)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/metric_histories/cpu", nil)
				})
				It("should succeed with 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
				})
			})
		})

		Context("aggregated metrics api", func() {
			Context("exceed rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(true)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/cpu", nil)
				})
				It("should succeed with 429", func() {
					Expect(resp.Code).To(Equal(http.StatusTooManyRequests))
					Expect(resp.Body.String()).To(Equal(`{"code":"Request-Limit-Exceeded","message":"Too many requests"}`))
				})
			})
			Context("below rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(false)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/aggregated_metric_histories/cpu", nil)
				})
				It("should succeed with 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
				})
			})
		})

		Context("scaling histories api", func() {
			Context("exceed rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(true)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories", nil)
				})
				It("should succeed with 429", func() {
					Expect(resp.Code).To(Equal(http.StatusTooManyRequests))
					Expect(resp.Body.String()).To(Equal(`{"code":"Request-Limit-Exceeded","message":"Too many requests"}`))
				})
			})
			Context("below rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(false)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/scaling_histories", nil)
				})
				It("should succeed with 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
				})
			})
		})

		Context("credential api", func() {
			Context("exceed rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(true)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/credential", nil)
				})
				It("should succeed with 429", func() {
					Expect(resp.Code).To(Equal(http.StatusTooManyRequests))
					Expect(resp.Body.String()).To(Equal(`{"code":"Request-Limit-Exceeded","message":"Too many requests"}`))
				})
			})
			Context("below rate limiting", func() {
				BeforeEach(func() {
					rateLimiter.ExceedsLimitReturns(false)
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/credential", nil)
				})
				It("should succeed with 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
				})
			})
		})

	})
})

func GetTestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Success"))
	}
}
