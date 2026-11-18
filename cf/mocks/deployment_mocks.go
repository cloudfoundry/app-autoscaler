package mocks

import (
	"net/http"

	"github.com/onsi/gomega/ghttp"
)

// NoActiveDeployments models a foundation with no deployment blocking scaling.
func (a AddMock) NoActiveDeployments() AddMock {
	a.server.RouteToHandler(http.MethodGet, "/v3/deployments",
		ghttp.RespondWith(http.StatusOK, `{"pagination":{"total_results":0,"total_pages":1,"next":null},"resources":[]}`))
	return a
}
