package testhelpers

import (
	"net/http"
	"sync/atomic"
)

var noOpHandler = func(_ http.ResponseWriter, _ *http.Request) {
	// empty function for Nop
}

func RespondWithMultiple(handlers ...http.HandlerFunc) http.HandlerFunc {
	var responseNumber int64 = 0
	if len(handlers) > 0 {
		return func(w http.ResponseWriter, req *http.Request) {
			responseNum := atomic.LoadInt64(&responseNumber)
			handlerNumber := min(responseNum, int64(len(handlers)-1))
			handlers[handlerNumber](w, req)
			atomic.AddInt64(&responseNumber, 1)
		}
	}
	return noOpHandler
}

func RoundRobinWithMultiple(handlers ...http.HandlerFunc) http.HandlerFunc {
	var responseNumber int64 = 0

	if len(handlers) > 0 {
		return func(w http.ResponseWriter, req *http.Request) {
			handlerNumber := atomic.LoadInt64(&responseNumber) % int64(len(handlers))
			handlers[handlerNumber](w, req)
			atomic.AddInt64(&responseNumber, 1)
		}
	}
	return noOpHandler
}
