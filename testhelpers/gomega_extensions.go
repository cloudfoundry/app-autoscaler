package testhelpers

import (
	"net/http"
)

func RespondWithMultiple(handlers ...http.HandlerFunc) http.HandlerFunc {
	responseNumber := 0
	if len(handlers) > 0 {
		return func(w http.ResponseWriter, req *http.Request) {
			handlerNumber := Min(responseNumber, len(handlers)-1)
			handlers[handlerNumber](w, req)
			responseNumber += 1
		}
	}
	return func(w http.ResponseWriter, req *http.Request) {}
}

func Min(one, two int) int {
	if one < two {
		return one
	}
	return two
}
