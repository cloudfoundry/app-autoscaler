package testhelpers

import (
	"net/http"
	"regexp"

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

type usage struct {
	State         State         `json:"state"`
	InstanceCount InstanceCount `json:"instance_count"`
}

func (a AddMock) AppUsageEvents(instanceCount int, appState string) {
	a.server.RouteToHandler("GET", regexp.MustCompile(`^/v3/app_usage_events/.*$`),
		ghttp.RespondWithJSONEncoded(http.StatusOK, usage{
			State:         State{Current: appState},
			InstanceCount: InstanceCount{Current: instanceCount},
		}))
}
