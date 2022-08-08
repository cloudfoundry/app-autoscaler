package cf_test

import (
	"errors"
	"regexp"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"

	"encoding/json"
	"net/http"
)

var _ = Describe("Cf client Roles", func() {

	var (
		conf            *cf.Config
		cfc             *cf.Client
		fakeCC          *MockServer
		fakeLoginServer *Server
		err             error
		logger          lager.Logger
	)

	var setCfcClient = func(maxRetries int) {
		conf = &cf.Config{}
		conf.API = fakeCC.URL()
		conf.MaxRetries = maxRetries
		conf.MaxRetryWaitMs = 1
		cfc = cf.NewCFClient(conf, logger, clock.NewClock())
		err = cfc.Login()
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		fakeCC = NewMockServer()
		fakeLoginServer = NewServer()
		fakeCC.Add().Info(fakeLoginServer.URL())
		fakeLoginServer.RouteToHandler("POST", cf.PathCFAuth, RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
			AccessToken: "test-access-token",
			ExpiresIn:   12000,
		}))
		logger = lager.NewLogger("cf")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		setCfcClient(0)
	})

	AfterEach(func() {
		if fakeCC != nil {
			fakeCC.Close()
		}
		if fakeLoginServer != nil {
			fakeLoginServer.Close()
		}
	})

	Describe("Roles.HasRole", func() {
		When("The role is present", func() {
			roles := cf.Roles{{Type: cf.RoleSpaceDeveloper}}
			It("Should return true", func() {
				Expect(roles.HasRole("space_developer")).To(BeTrue())
			})
		})
		When("The role is not present", func() {
			roles := cf.Roles{
				{Type: cf.RoleSpaceManager},
				{Type: cf.RoleOrganizationManager},
				{Type: cf.RoleOrganizationBillingManager},
				{Type: cf.RoleOrganisationUser},
				{Type: cf.RoleOrganizationAuditor},
				{Type: cf.RoleSpaceSupporter},
			}
			It("should return false", func() {
				Expect(roles.HasRole(cf.RoleSpaceDeveloper)).To(BeFalse())

			})

		})
		When("the roles is nil", func() {})
	})

	Describe("GetRoles", func() {

		When("the mocks are used", func() {
			var mocks = NewMockServer()
			BeforeEach(func() {
				conf.API = mocks.URL()
				mocks.Add().Info(fakeLoginServer.URL()).Roles(cf.Role{Guid: "mock_guid", Type: cf.RoleSpaceDeveloper})

				DeferCleanup(mocks.Close)
			})
			It("will return success", func() {
				roles, err := cfc.GetSpaceDeveloperRoles("some_space", "some_user")
				Expect(err).NotTo(HaveOccurred())
				Expect(roles).To(Equal(cf.Roles{
					{
						Guid: "mock_guid",
						Type: cf.RoleSpaceDeveloper,
					},
				}))
				Expect(roles.HasRole(cf.RoleSpaceDeveloper)).To(BeTrue())
			})
		})

		When("get roles succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/roles", "types=space_developer&space_guids=some_space_id&user_guids=someUserId"),
						RespondWith(http.StatusCreated, LoadFile("roles.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct struct", func() {
				spaceId := cf.SpaceId("some_space_id")
				userId := cf.UserId("someUserId")
				roles, err := cfc.GetSpaceDeveloperRoles(spaceId, userId)
				Expect(err).NotTo(HaveOccurred())
				Expect(roles).To(Equal(cf.Roles{
					{
						Guid: "40557c70-d1bd-4976-a2ab-a85f5e882418",
						Type: "organization_auditor",
					},
					{
						Guid: "12347c70-d1bd-4976-a2ab-a85f5e882418",
						Type: "space_auditor",
					},
					{
						Guid: "12347c70-d1bd-4976-a2ab-a85f5e882418",
						Type: "space_auditor",
					}}))
			})
		})

		When("get app usage return 404 status code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/apps/404"),
						RespondWithJSONEncoded(http.StatusNotFound, models.CfResourceNotFound),
					),
				)
			})

			It("should error", func() {
				app, err := cfc.GetApp("404")
				Expect(app).To(BeNil())
				var cfError *models.CfError
				Expect(errors.As(err, &cfError) && cfError.IsNotFound()).To(BeTrue())
				Expect(models.IsNotFound(err)).To(BeTrue())
			})
		})

		When("get app returns 500 status code", func() {
			BeforeEach(func() {
				setCfcClient(3)
			})
			When("it never recovers", func() {

				BeforeEach(func() {
					fakeCC.RouteToHandler("GET", regexp.MustCompile(`^/v3/apps/[^/]+$`),
						RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
					)
				})

				It("should error", func() {
					app, err := cfc.GetApp("500")
					Expect(app).To(BeNil())
					Expect(fakeCC.Count().Requests(`^/v3/apps/[^/]+$`)).To(Equal(4))
					Expect(err).To(MatchError(MatchRegexp("failed getting app '500':.*'UnknownError'")))
				})
			})
			When("it recovers after 3 retries", func() {
				BeforeEach(func() {
					fakeCC.RouteToHandler("GET", regexp.MustCompile(`^/v3/apps/[^/]+$`),
						RespondWithMultiple(
							RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
							RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
							RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
							RespondWith(http.StatusOK, LoadFile("testdata/app.json"), http.Header{"Content-Type": []string{"application/json"}}),
						))
				})

				It("should return success", func() {
					app, err := cfc.GetApp("500")
					Expect(err).NotTo(HaveOccurred())
					Expect(app).ToNot(BeNil())
					Expect(fakeCC.Count().Requests(`^/v3/apps/[^/]+$`)).To(Equal(4))
				})
			})
		})

		When("get app returns a non-200 and non-404 status code with non-JSON response", func() {
			BeforeEach(func() {
				fakeCC.RouteToHandler("GET", "/v3/apps/invalid_json", RespondWithJSONEncoded(http.StatusInternalServerError, ""))
			})

			It("should error", func() {
				app, err := cfc.GetApp("invalid_json")
				Expect(app).To(BeNil())
				Expect(err.Error()).To(MatchRegexp("failed getting app '.*':.*failed to unmarshal"))
			})
		})

		When("cloud controller is not reachable", func() {
			BeforeEach(func() {
				fakeCC.Close()
				fakeCC = nil
			})

			It("should error", func() {
				app, err := cfc.GetApp("something")
				Expect(app).To(BeNil())
				IsUrlNetOpError(err)
			})
		})

		When("cloud controller returns incorrect message body", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/apps/incorrect_object"),
						RespondWithJSONEncoded(http.StatusOK, `{"entity":{"instances:"abc"}}`),
					),
				)
			})

			It("should error", func() {
				app, err := cfc.GetApp("incorrect_object")
				Expect(app).To(BeNil())
				Expect(err).To(MatchError(MatchRegexp(`failed unmarshalling \*cf.App:.*cannot unmarshal string`)))
				var errType *json.UnmarshalTypeError
				Expect(errors.As(err, &errType)).Should(BeTrue(), "Error was: %#v", interface{}(err))
			})

		})
	})

})
