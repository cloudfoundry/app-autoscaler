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

func NewMockServer() *MockServer {
	return &MockServer{ghttp.NewServer()}
}
func (m *MockServer) Add() *AddMock {
	return &AddMock{m}
}

func (m *MockServer) Count() *CountMock {
	return &CountMock{m}
}

type CountMock struct{ server *MockServer }

func (m CountMock) Requests(urlRegExp string) int {
	count := 0
	for _, req := range m.server.ReceivedRequests() {
		found, err := regexp.Match(urlRegExp, []byte(req.RequestURI))
		if err != nil {
			panic(err)
		}
		if found {
			count++
		}
	}
	return count
}

type State struct {
	Current  string `json:"current"`
	Previous string `json:"previous"`
}
type InstanceCount struct {
	Current  int `json:"current"`
	Previous int `json:"previous"`
}

type AddMock struct{ server *MockServer }

func (a AddMock) GetApp(appState string) AddMock {
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
	return a
}

func (a AddMock) GetAppProcesses(processes int) AddMock {
	type processesResponse struct {
		Pagination cf.Pagination `json:"pagination"`
		Resources  cf.Processes  `json:"resources"`
	}
	a.server.RouteToHandler("GET",
		regexp.MustCompile(`^/v3/apps/[^/]+/processes$`),
		ghttp.RespondWithJSONEncoded(http.StatusOK, processesResponse{Resources: cf.Processes{{Instances: processes}}}))
	return a
}

func (a AddMock) Info(url string) AddMock {
	a.server.RouteToHandler("GET", cf.PathCFInfo, ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Endpoints{
		AuthEndpoint:    url,
		TokenEndpoint:   url,
		DopplerEndpoint: "test-doppler-endpoint",
	}))
	return a
}
