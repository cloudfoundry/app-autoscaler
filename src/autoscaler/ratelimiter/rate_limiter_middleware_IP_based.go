package ratelimiter

import (
	"net/http"
	"strings"

	"autoscaler/models"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
)

type RateLimiterMiddlewareIPBased struct {
	logger      lager.Logger
	RateLimiter Limiter
}

func NewRateLimiterMiddlewareIPBased(rateLimiter Limiter, logger lager.Logger) *RateLimiterMiddlewareIPBased {
	return &RateLimiterMiddlewareIPBased{
		logger:      logger,
		RateLimiter: rateLimiter,
	}
}

func (mw *RateLimiterMiddlewareIPBased) CheckRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteIP := strings.Split(r.RemoteAddr, ":")[0]
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			remoteIP = strings.Split(forwardedFor, ", ")[0]
		}
		mw.logger.Info("error-exceed-rate-limit", lager.Data{"RemoteIP": remoteIP, "X-Vcap-Request-Id": r.Header.Get("X-Vcap-Request-Id"), "URL": r.URL.String()})
		if mw.RateLimiter.ExceedsLimit(remoteIP) {
			mw.logger.Info("error-exceed-rate-limit", lager.Data{"RemoteIP": remoteIP, "X-Vcap-Request-Id": r.Header.Get("X-Vcap-Request-Id"), "URL": r.URL.String()})
			handlers.WriteJSONResponse(w, http.StatusTooManyRequests, models.ErrorResponse{
				Code:    "Request-Limit-Exceeded",
				Message: "Too many requests"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
