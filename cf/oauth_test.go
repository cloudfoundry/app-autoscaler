package cf_test

import (
	"net/http"
	"regexp"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

type userInfo struct {
	UserId string `json:"user_id"`
}

type userScope struct {
	Scope []string `json:"scope"`
}

type roles struct {
	Pagination struct {
		Total int `json:"total_results"`
	} `json:"pagination"`
}

type app struct {
	Relationships struct {
		Space struct {
			Data struct {
				GUID string `json:"guid"`
			} `json:"data"`
		} `json:"space"`
	} `json:"relationships"`
}

const (
	TEST_USER_TOKEN = "bearer test-user-token"
	TEST_APP_ID     = "test-app-id"
	TEST_SPACE_ID   = "test-space-id"
	TEST_USER_ID    = "test-user-id"
)

var (
	conf      *CFConfig
	cfc       CFClient
	err       error
	userToken string
	logger    *lagertest.TestLogger

	isUserSpaceDeveloperFlag bool
	isUserAdminFlag          bool

	fakeCCServer    *ghttp.Server
	fakeTokenServer *ghttp.Server

	userInfoStatus   int
	userInfoResponse userInfo

	userScopeStatus   int
	userScopeResponse userScope

	appStatus       int
	appResponse     app
	appsPathMatcher *regexp.Regexp

	rolesStatus   int
	rolesResponse roles
)

var _ = Describe("Oauth", func() {

	BeforeEach(func() {
		userToken = TEST_USER_TOKEN

		fakeCCServer = ghttp.NewServer()
		fakeTokenServer = ghttp.NewServer()

		fakeCCServer.RouteToHandler(http.MethodGet, "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK, Endpoints{
			AuthEndpoint:    "test-auth-endpoint",
			TokenEndpoint:   fakeTokenServer.URL(),
			DopplerEndpoint: "test-doppler-endpoint",
		}))

		fakeTokenServer.RouteToHandler(http.MethodGet, "/userinfo", ghttp.RespondWithJSONEncodedPtr(&userInfoStatus, &userInfoResponse))
		fakeTokenServer.RouteToHandler(http.MethodPost, "/check_token", ghttp.RespondWithJSONEncodedPtr(&userScopeStatus, &userScopeResponse))
		fakeTokenServer.RouteToHandler("POST", PathCFAuth, ghttp.RespondWithJSONEncoded(http.StatusOK, Tokens{
			AccessToken: "test-access-token",
			ExpiresIn:   12000,
		}))

		appsPathMatcher, _ = regexp.Compile(`/v3/apps/[A-Za-z0-9\-]+`)
		fakeCCServer.RouteToHandler(http.MethodGet, appsPathMatcher, ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/v3/apps/"+TEST_APP_ID),
			ghttp.RespondWithJSONEncodedPtr(&appStatus, &appResponse)))

		fakeCCServer.RouteToHandler(http.MethodGet, "/v3/roles", ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/v3/roles", "types=space_developer&space_guids="+TEST_SPACE_ID+"&user_guids="+TEST_USER_ID),
			ghttp.RespondWithJSONEncodedPtr(&rolesStatus, &rolesResponse)))

		conf = &CFConfig{}
		conf.API = fakeCCServer.URL()
		logger = lagertest.NewTestLogger("oauth-test")
		cfc = NewCFClient(conf, logger, clock.NewClock())
		err = cfc.Login()
	})

	AfterEach(func() {
		if fakeCCServer != nil {
			fakeCCServer.Close()
		}
		if fakeTokenServer != nil {
			fakeTokenServer.Close()
		}
	})

	Describe("IsUserSpaceDeveloper", func() {
		JustBeforeEach(func() {
			isUserSpaceDeveloperFlag, err = cfc.IsUserSpaceDeveloper(TEST_USER_TOKEN, TEST_APP_ID)
		})

		Context("token server is not reachable", func() {
			BeforeEach(func() {
				fakeTokenServer.Close()
				fakeTokenServer = nil
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("user info endpoint returns 401 statusCode", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusUnauthorized
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Unauthorized"))
			})
		})

		Context("user info endpoint returns non-200 and non-401 statusCode", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusBadRequest
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Failed to get user info, statuscode :400"))
			})
		})

		Context("user info response is not in json format", func() {
			BeforeEach(func() {
				fakeTokenServer.RouteToHandler(http.MethodGet, "/userinfo", ghttp.RespondWith(http.StatusOK, "non-json-response"))
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("cc server is not reachable", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}

				fakeCCServer.Close()
				fakeCCServer = nil

			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("apps endpoint returns non-200 and non-401 status code", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}
				appStatus = http.StatusBadRequest
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Failed to get app, statusCode : 400"))
			})
		})

		Context("apps endpoint returns non-200 and non-401 status code", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}
				appStatus = http.StatusUnauthorized
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Unauthorized"))
			})
		})

		Context("apps endpoint returns non json response", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}
				fakeCCServer.RouteToHandler(http.MethodGet, appsPathMatcher, ghttp.RespondWith(http.StatusOK, "non-json-response"))

			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("roles endpoint returns non-200 and non-401 status code", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}
				appStatus = http.StatusOK
				appResponse = app{}
				appResponse.Relationships.Space.Data.GUID = TEST_SPACE_ID

				rolesStatus = http.StatusBadRequest
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Failed to get roles, statusCode : 400"))
			})
		})

		Context("roles endpoint returns 401 status code", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}
				appStatus = http.StatusOK
				appResponse = app{}
				appResponse.Relationships.Space.Data.GUID = TEST_SPACE_ID

				rolesStatus = http.StatusUnauthorized
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Unauthorized"))
			})
		})

		Context("roles endpoint returns non json response", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}
				appStatus = http.StatusOK
				appResponse = app{}
				appResponse.Relationships.Space.Data.GUID = TEST_SPACE_ID

				fakeCCServer.RouteToHandler(http.MethodGet, "/v3/roles", ghttp.RespondWith(http.StatusOK, "non-json-response"))

			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("user is not space developer", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}
				appStatus = http.StatusOK
				appResponse = app{}
				appResponse.Relationships.Space.Data.GUID = TEST_SPACE_ID

				rolesStatus = http.StatusOK
				rolesResponse = roles{}

			})
			It("should return false", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(isUserSpaceDeveloperFlag).To(BeFalse())
			})
		})

		Context("user is space developer", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}
				appStatus = http.StatusOK
				appResponse = app{}
				appResponse.Relationships.Space.Data.GUID = TEST_SPACE_ID

				rolesStatus = http.StatusOK
				rolesResponse = roles{}
				rolesResponse.Pagination.Total = 2

			})
			It("should return true", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(isUserSpaceDeveloperFlag).To(BeTrue())
			})
		})

	})

	Describe("IsUserAdmin", func() {
		JustBeforeEach(func() {
			isUserAdminFlag, err = cfc.IsUserAdmin(userToken)
		})

		Context("token server is not reachable", func() {
			BeforeEach(func() {
				fakeTokenServer.Close()
				fakeTokenServer = nil
			})
			It("should error", func() {
				Expect(isUserAdminFlag).To(BeFalse())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("user scope endpoint returns non-200 status code", func() {
			BeforeEach(func() {
				userScopeStatus = http.StatusBadRequest
			})
			It("should return false", func() {
				Expect(isUserAdminFlag).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Failed to get user scope, statusCode : 400"))
			})
		})

		Context("userscope response is not in json format", func() {
			BeforeEach(func() {
				fakeTokenServer.RouteToHandler(http.MethodPost, "/check_token", ghttp.RespondWith(http.StatusOK, "non-json-response"))
			})
			It("should error", func() {
				Expect(isUserAdminFlag).To(BeFalse())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("user is not admin", func() {
			BeforeEach(func() {
				userScopeStatus = http.StatusOK
				userScopeResponse = userScope{
					Scope: []string{"some.scope"},
				}
			})
			It("should return false", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(isUserAdminFlag).To(BeFalse())
			})
		})

		Context("user is admin", func() {
			BeforeEach(func() {
				userScopeStatus = http.StatusOK
				userScopeResponse = userScope{
					Scope: []string{CCAdminScope},
				}
			})
			It("should return true", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(isUserAdminFlag).To(BeTrue())
			})
		})
	})
})
