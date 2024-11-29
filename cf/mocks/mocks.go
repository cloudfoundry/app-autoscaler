package mocks

import (
	"net/http"
	"regexp"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

type Server struct {
	*ghttp.Server
}

func NewServer() *Server {
	return NewWithServer(ghttp.NewServer())
}
func NewWithServer(server *ghttp.Server) *Server {
	return &Server{server}
}

func NewMockTlsServer() *Server {
	return &Server{ghttp.NewTLSServer()}
}

func (m *Server) Add() *AddMock {
	return &AddMock{m}
}

func (m *Server) Count() *CountMock {
	return &CountMock{m}
}

type CountMock struct{ server *Server }

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

type AddMock struct{ server *Server }

func (a AddMock) GetApp(appState string, statusCode int, spaceGuid cf.SpaceId) AddMock {
	created, err := time.Parse(time.RFC3339, "2022-07-21T13:42:30Z")
	Expect(err).NotTo(HaveOccurred())
	updated, err := time.Parse(time.RFC3339, "2022-07-21T14:30:17Z")
	Expect(err).NotTo(HaveOccurred())

	response := definedResponsesOR(statusCode, cf.App{
		Guid:      "testing-guid-get-app",
		Name:      "mock-get-app",
		State:     appState,
		CreatedAt: created,
		UpdatedAt: updated,
		Relationships: cf.Relationships{
			Space: &cf.Space{
				Data: cf.SpaceData{
					Guid: spaceGuid,
				},
			},
		},
	})
	a.server.RouteToHandler("GET",
		regexp.MustCompile(`^/v3/apps/[^/]+$`),
		ghttp.RespondWithJSONEncoded(statusCode, response))
	return a
}

func definedResponsesOR(statusCode int, response interface{}) interface{} {
	switch statusCode {
	case 401:
		response = cf.CfNotAuthenticated
	case 403:
		response = cf.CfNotAuthorized
	case 404:
		response = cf.CfResourceNotFound
	default:
	}
	return response
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
	a.server.RouteToHandler("GET", "/", ghttp.RespondWithJSONEncoded(http.StatusOK, cf.EndpointsResponse{
		Links: cf.Endpoints{
			Login: cf.Href{Url: url},
			Uaa:   cf.Href{Url: url},
		},
	}))
	return a
}

func (a AddMock) ScaleAppWebProcess() AddMock {
	a.server.RouteToHandler("POST", regexp.MustCompile(`^/v3/apps/[^/]+/processes/web/actions/scale$`), ghttp.RespondWith(http.StatusAccepted, "{}"))
	return a
}

func (a AddMock) Roles(statusCode int, roles ...cf.Role) AddMock {
	a.server.RouteToHandler("GET", "/v3/roles",
		ghttp.RespondWithJSONEncoded(statusCode, definedResponsesOR(statusCode, cf.Response[cf.Role]{Resources: roles})))
	return a
}

func (a AddMock) ServiceInstance(planGuid string) AddMock {
	a.server.RouteToHandler("GET", regexp.MustCompile(`^/v3/service_instances/[^/]+$`),
		ghttp.RespondWithJSONEncoded(http.StatusOK, &cf.ServiceInstance{
			Guid:          "service-instance-mock-guid",
			Type:          "managed",
			Relationships: cf.ServiceInstanceRelationships{ServicePlan: cf.ServicePlanRelation{Data: cf.ServicePlanData{Guid: planGuid}}},
		}),
	)
	return a
}

func (a AddMock) ServicePlan(brokerPlanId string) AddMock {
	a.server.RouteToHandler("GET", regexp.MustCompile(`^/v3/service_plans/[^/]+$`),
		ghttp.RespondWithJSONEncoded(http.StatusOK, cf.ServicePlan{BrokerCatalog: cf.BrokerCatalog{Id: brokerPlanId}}),
	)
	return a
}

func (a AddMock) UserInfo(statusCode int, testUserId string) AddMock {
	a.server.RouteToHandler(http.MethodGet, "/userinfo",
		ghttp.RespondWithJSONEncoded(statusCode,
			struct {
				UserId string `json:"user_id"`
			}{testUserId}))
	return a
}

func (a AddMock) Introspect(testUserScope []string) AddMock {
	a.server.RouteToHandler(http.MethodPost, "/introspect",
		ghttp.RespondWithJSONEncoded(http.StatusOK,
			struct {
				Scope []string `json:"scope"`
			}{
				testUserScope,
			}))
	return a
}

func (a AddMock) OauthToken(accessToken string) AddMock {
	a.server.RouteToHandler(http.MethodPost, "/oauth/token",
		ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{AccessToken: accessToken, ExpiresIn: 12000}))
	return a
}
