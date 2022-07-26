package testhelpers

import (
	"net/http"
	"regexp"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "github.com/onsi/gomega"
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
	created, err := time.Parse(time.RFC3339, "2022-07-21T13:42:30Z")
	Expect(err).NotTo(HaveOccurred())
	updated, err := time.Parse(time.RFC3339, "2022-07-21T14:30:17Z")
	Expect(err).NotTo(HaveOccurred())
	a.server.RouteToHandler("GET",
		regexp.MustCompile(`^/v3/apps/[^/]+$`),
		ghttp.RespondWithJSONEncoded(http.StatusOK, cf.App{
			Guid:      "testing-guid-get-app",
			Name:      "mock-get-app",
			State:     appState,
			CreatedAt: created,
			UpdatedAt: updated,
			Relationships: cf.Relationships{
				Space: &cf.Space{
					Data: cf.SpaceData{
						Guid: "test_space_guid",
					},
				},
			},
		}))
}

func (a AddMock) GetAppProcesses(processes int) {
	type processesResponse struct {
		Pagination cf.Pagination `json:"pagination"`
		Resources  cf.Processes  `json:"resources"`
	}
	a.server.RouteToHandler("GET",
		regexp.MustCompile(`^/v3/apps/[^/]+/processes$`),
		ghttp.RespondWithJSONEncoded(http.StatusOK, processesResponse{Resources: cf.Processes{{Instances: processes}}}))
}
