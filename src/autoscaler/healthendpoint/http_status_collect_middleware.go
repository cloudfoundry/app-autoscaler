package healthendpoint

import (
	"net/http"
)

type HTTPStatusCollectMiddleware struct {
	httpStatusCollector HTTPStatusCollector
}

func NewHTTPStatusCollectMiddleware(httpStatusCollector HTTPStatusCollector) *HTTPStatusCollectMiddleware {
	return &HTTPStatusCollectMiddleware{
		httpStatusCollector: httpStatusCollector,
	}
}

func (h *HTTPStatusCollectMiddleware) Collect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.httpStatusCollector.IncConcurrentHTTPRequest()
		defer h.httpStatusCollector.DecConcurrentHTTPRequest()
		next.ServeHTTP(w, r)
	})
}
