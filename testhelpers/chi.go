package testhelpers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func SetURLParams(req *http.Request, params ...string) *http.Request {
	rc := chi.NewRouteContext()
	for i := 0; i+1 < len(params); i += 2 {
		rc.URLParams.Add(params[i], params[i+1])
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	req = req.WithContext(ctx)
	return req
}
