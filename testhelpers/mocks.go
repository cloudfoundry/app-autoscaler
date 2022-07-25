package testhelpers

import (
	"net/http"
	"regexp"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"github.com/onsi/gomega/ghttp"
)

type MockServer struct {
	*ghttp.Server
}
type AddMock struct{ server *MockServer }

func NewMockServer() *MockServer {
	return &MockServer{ghttp.NewServer()}
}
func (m *MockServer) Add() *AddMock {
	return &AddMock{m}
}

type State struct {
	Current  string `json:"current"`
	Previous string `json:"previous"`
}
type InstanceCount struct {
	Current  int `json:"current"`
	Previous int `json:"previous"`
}

func (a AddMock) GetApp(appState string) {
	a.server.RouteToHandler("GET",
		regexp.MustCompile(`^/v3/apps/[^/]+$`),
		ghttp.RespondWithJSONEncoded(http.StatusOK, cf.App{State: appState}))
}

func (a AddMock) GetAppProcesses(processes int) {
	a.server.RouteToHandler("GET",
		regexp.MustCompile(`^/v3/apps/[^/]+/processes$`),
		ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Processes{{Instances: processes}}))
}
