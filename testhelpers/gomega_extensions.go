package testhelpers

import (
	"net/http"
)

var noOpHandler = func(_ http.ResponseWriter, _ *http.Request) {
	// empty function for Nop
}

func RespondWithMultiple(handlers ...http.HandlerFunc) http.HandlerFunc {
	responseNumber := 0
	if len(handlers) > 0 {
		return func(w http.ResponseWriter, req *http.Request) {
			handlerNumber := Min(responseNumber, len(handlers)-1)
			handlers[handlerNumber](w, req)
			responseNumber += 1
		}
	}
	return noOpHandler
}

func RoundRobinWithMultiple(handlers ...http.HandlerFunc) http.HandlerFunc {
	responseNumber := 0
	if len(handlers) > 0 {
		return func(w http.ResponseWriter, req *http.Request) {
			handlerNumber := responseNumber % len(handlers)
			handlers[handlerNumber](w, req)
			responseNumber += 1
		}
	}
	return noOpHandler
}

func Min(one, two int) int {
	if one < two {
		return one
	}
	return two
}
