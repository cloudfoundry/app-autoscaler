package ratelimiter

import (
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
)

type RateLimiterMiddleware struct {
	Key         string
	logger      lager.Logger
	RateLimiter Limiter
}

func NewRateLimiterMiddleware(key string, rateLimiter Limiter, logger lager.Logger) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		Key:         key,
		logger:      logger,
		RateLimiter: rateLimiter,
	}
}

func (mw *RateLimiterMiddleware) CheckRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		key := vars[mw.Key]
		if key == "" {
			mw.logger.Error("Key "+mw.Key+" is not present in the request", nil, lager.Data{"url": r.URL.String()})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad Request",
				Message: "Missing rate limit key",
			})
			return
		}
		if mw.RateLimiter.ExceedsLimit(key) {
			mw.logger.Info("error-exceed-rate-limit", lager.Data{mw.Key: key})
			handlers.WriteJSONResponse(w, http.StatusTooManyRequests, models.ErrorResponse{
				Code:    "Request-Limit-Exceeded",
				Message: "Too many requests"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
