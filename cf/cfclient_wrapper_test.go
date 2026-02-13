package cf_test

import (
	"context"
	"encoding/json"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf/mocks"
	"code.cloudfoundry.org/lager/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CFClientWrapper", func() {
	var (
		mockServer   *mocks.Server
		client       cf.CFClient
		clientErr    error
		conf         *cf.Config
		logger       lager.Logger
		createClient bool
		ctx          context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockServer = mocks.NewMockTlsServer()
		logger = lager.NewLogger("cf-wrapper-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		conf = &cf.Config{
			ClientConfig: cf.ClientConfig{
				// SkipSSLValidation is false - we inject the test server's pre-configured HTTP client
				// which trusts the test server's certificate for both go-cfclient and UAA requests.
				SkipSSLValidation: false,
			},
			API:      mockServer.URL(),
			ClientID: "test-client",
			Secret:   "test-secret",
		}
		createClient = true
	})

	JustBeforeEach(func() {
		if createClient {
			mockServer.Add().OauthToken("test-access-token")
			mockServer.Add().Info(mockServer.URL())
			// Use the test server's HTTP client which trusts the test server's certificate
			client, clientErr = cf.NewCFClient(conf, logger, cf.WithHTTPClient(mockServer.HTTPTestServer.Client()))
			Expect(clientErr).NotTo(HaveOccurred())
		}
	})

	AfterEach(func() {
		mockServer.Close()
	})

	Describe("NewCFClientWrapper", func() {
		BeforeEach(func() {
			createClient = false
		})

		It("creates a client successfully with client credentials", func() {
			mockServer.Add().OauthToken("test-access-token")
			mockServer.Add().Info(mockServer.URL())

			var err error
			client, err = cf.NewCFClient(conf, logger, cf.WithHTTPClient(mockServer.HTTPTestServer.Client()))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("creates a client successfully with password grant", func() {
			conf.GrantType = cf.GrantTypePassword
			conf.Username = "test-user"
			conf.Password = "test-password"
			conf.ClientID = "cf"
			conf.Secret = ""

			mockServer.Add().OauthToken("test-password-grant-token")
			mockServer.Add().Info(mockServer.URL())

			var err error
			client, err = cf.NewCFClient(conf, logger, cf.WithHTTPClient(mockServer.HTTPTestServer.Client()))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify the client can login (which validates the token was obtained)
			err = client.Login(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error for invalid API URL", func() {
			conf.API = "://invalid-url"
			_, err := cf.NewCFClient(conf, logger, cf.WithHTTPClient(mockServer.HTTPTestServer.Client()))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Login", func() {
		It("logs in successfully", func() {

			err := client.Login(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetApp", func() {
		It("returns app details", func() {
			mockServer.Add().GetApp("STARTED", http.StatusOK, "test-space-guid")

			app, err := client.GetApp(ctx, "test-app-guid")
			Expect(err).NotTo(HaveOccurred())
			Expect(app).NotTo(BeNil())
			Expect(app.State).To(Equal("STARTED"))
			Expect(app.Relationships.Space.Data.Guid).To(Equal(cf.SpaceId("test-space-guid")))
		})

		It("returns error for non-existent app", func() {
			mockServer.Add().GetApp("", http.StatusNotFound, "")

			_, err := client.GetApp(ctx, "non-existent-app")
			Expect(err).To(HaveOccurred())
			Expect(cf.IsNotFound(err)).To(BeTrue())
		})
	})

	Describe("GetAppProcesses", func() {
		It("returns process details", func() {
			mockServer.Add().GetAppProcesses(3)

			processes, err := client.GetAppProcesses(ctx, "test-app-guid", cf.ProcessTypeWeb)
			Expect(err).NotTo(HaveOccurred())
			Expect(processes).To(HaveLen(1))
			Expect(processes[0].Instances).To(Equal(3))
			Expect(processes[0].Type).To(Equal("web"))
		})
	})

	Describe("GetAppAndProcesses", func() {
		It("returns both app and processes", func() {
			mockServer.Add().GetApp("STARTED", http.StatusOK, "test-space-guid")
			mockServer.Add().GetAppProcesses(2)

			result, err := client.GetAppAndProcesses(ctx, "test-app-guid")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.App).NotTo(BeNil())
			Expect(result.App.State).To(Equal("STARTED"))
			Expect(result.Processes).To(HaveLen(1))
			Expect(result.Processes[0].Instances).To(Equal(2))
		})
	})

	Describe("ScaleAppWebProcess", func() {
		It("scales the app successfully", func() {
			mockServer.Add().GetAppProcesses(2)
			mockServer.Add().ScaleAppWebProcess()

			err := client.ScaleAppWebProcess(ctx, "test-app-guid", 5)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetServiceInstance", func() {
		It("returns service instance details", func() {
			mockServer.Add().ServiceInstance("test-plan-guid")

			si, err := client.GetServiceInstance(ctx, "test-si-guid")
			Expect(err).NotTo(HaveOccurred())
			Expect(si).NotTo(BeNil())
			Expect(si.Type).To(Equal("managed"))
			Expect(si.Relationships.ServicePlan.Data.Guid).To(Equal("test-plan-guid"))
		})
	})

	Describe("GetServicePlan", func() {
		It("returns service plan details", func() {
			mockServer.Add().ServicePlan("broker-plan-id")

			sp, err := client.GetServicePlan(ctx, "test-plan-guid")
			Expect(err).NotTo(HaveOccurred())
			Expect(sp).NotTo(BeNil())
			Expect(sp.BrokerCatalog.Id).To(Equal("broker-plan-id"))
		})
	})

	Describe("IsUserAdmin", func() {
		It("returns true when user has admin scope", func() {
			mockServer.Add().Introspect([]string{"cloud_controller.admin", "openid"})

			isAdmin, err := client.IsUserAdmin(ctx, "user-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(isAdmin).To(BeTrue())
		})

		It("returns false when user does not have admin scope", func() {
			mockServer.Add().Introspect([]string{"openid", "cloud_controller.read"})

			isAdmin, err := client.IsUserAdmin(ctx, "user-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(isAdmin).To(BeFalse())
		})
	})

	Describe("IsUserSpaceDeveloper", func() {
		It("returns true when user is space developer", func() {
			mockServer.Add().UserInfo(http.StatusOK, "test-user-id")
			mockServer.Add().GetApp("STARTED", http.StatusOK, "test-space-guid")
			mockServer.Add().Roles(http.StatusOK, cf.Role{Guid: "role-guid", Type: cf.RoleSpaceDeveloper})

			isSpaceDev, err := client.IsUserSpaceDeveloper(ctx, "user-token", "test-app-guid")
			Expect(err).NotTo(HaveOccurred())
			Expect(isSpaceDev).To(BeTrue())
		})

		It("returns false when user is not space developer", func() {
			mockServer.Add().UserInfo(http.StatusOK, "test-user-id")
			mockServer.Add().GetApp("STARTED", http.StatusOK, "test-space-guid")
			mockServer.Add().Roles(http.StatusOK) // No roles

			isSpaceDev, err := client.IsUserSpaceDeveloper(ctx, "user-token", "test-app-guid")
			Expect(err).NotTo(HaveOccurred())
			Expect(isSpaceDev).To(BeFalse())
		})

		It("returns false without error when token is unauthorized", func() {
			mockServer.Add().UserInfo(http.StatusUnauthorized, "")

			isSpaceDev, err := client.IsUserSpaceDeveloper(ctx, "invalid-token", "test-app-guid")
			// ErrUnauthorized is handled gracefully - returns false without error
			Expect(err).NotTo(HaveOccurred())
			Expect(isSpaceDev).To(BeFalse())
		})
	})

	Describe("IsTokenAuthorized", func() {
		It("returns true when token is authorized for client", func() {
			mockServer.RouteToHandler(http.MethodPost, "/introspect",
				RespondWithJSON(http.StatusOK, map[string]any{
					"active":    true,
					"client_id": "expected-client",
				}))

			isAuthorized, err := client.IsTokenAuthorized(ctx, "some-token", "expected-client")
			Expect(err).NotTo(HaveOccurred())
			Expect(isAuthorized).To(BeTrue())
		})

		It("returns false when client_id does not match", func() {
			mockServer.RouteToHandler(http.MethodPost, "/introspect",
				RespondWithJSON(http.StatusOK, map[string]any{
					"active":    true,
					"client_id": "different-client",
				}))

			isAuthorized, err := client.IsTokenAuthorized(ctx, "some-token", "expected-client")
			Expect(err).NotTo(HaveOccurred())
			Expect(isAuthorized).To(BeFalse())
		})

		It("returns false when token is inactive", func() {
			mockServer.RouteToHandler(http.MethodPost, "/introspect",
				RespondWithJSON(http.StatusOK, map[string]any{
					"active":    false,
					"client_id": "expected-client",
				}))

			isAuthorized, err := client.IsTokenAuthorized(ctx, "some-token", "expected-client")
			Expect(err).NotTo(HaveOccurred())
			Expect(isAuthorized).To(BeFalse())
		})
	})

	Describe("GetEndpoints", func() {
		It("returns endpoints and caches them", func() {
			// Count requests before calling GetEndpoints
			requestsBefore := mockServer.Count().Requests("^/$")

			endpoints, err := client.GetEndpoints(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(endpoints.Uaa.Url).To(Equal(mockServer.URL()))
			Expect(endpoints.Login.Url).To(Equal(mockServer.URL()))

			// Second call should use cached value (no additional request)
			endpoints2, err := client.GetEndpoints(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(endpoints2).To(Equal(endpoints))

			// Verify only one additional request was made to / (for GetEndpoints, not during client init)
			requestsAfter := mockServer.Count().Requests("^/$")
			Expect(requestsAfter - requestsBefore).To(Equal(1))
		})
	})
})

func RespondWithJSON(statusCode int, body any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		data, _ := json.Marshal(body)
		_, _ = w.Write(data)
	}
}
