package cf_test

import (
	"net/http"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

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

const (
	TestUserToken = "bearer test-user-token"
	TestAppId     = "test-app-id"
	TestSpaceId   = "test-space-id"
	TestUserId    = "test-user-id"
)

var _ = Describe("Oauth", func() {

	var (
		conf      *Config
		cfc       CFClient
		err       error
		userToken string
		logger    *lagertest.TestLogger

		isUserSpaceDeveloperFlag bool
		isUserAdminFlag          bool

		fakeCCServer    *testhelpers.MockServer
		fakeTokenServer *testhelpers.MockServer

		userInfoStatus   int
		userInfoResponse userInfo

		userScopeStatus   int
		userScopeResponse userScope

		appStatus int

		rolesStatus int
		roles       Roles
	)

	BeforeEach(func() {
		userToken = TestUserToken
		appStatus = http.StatusOK
		userScopeStatus = http.StatusOK
		userInfoStatus = http.StatusOK
		rolesStatus = http.StatusOK
		roles = Roles{{Type: RoleSpaceDeveloper}}
		userInfoResponse = userInfo{
			UserId: TestUserId,
		}

		fakeCCServer = testhelpers.NewMockServer()
		fakeTokenServer = testhelpers.NewMockServer()
		fakeTokenServer.RouteToHandler(http.MethodGet, "/userinfo", ghttp.RespondWithJSONEncodedPtr(&userInfoStatus, &userInfoResponse))
		fakeTokenServer.RouteToHandler(http.MethodPost, "/check_token", ghttp.RespondWithJSONEncodedPtr(&userScopeStatus, &userScopeResponse))
		fakeTokenServer.RouteToHandler("POST", PathCFAuth, ghttp.RespondWithJSONEncoded(http.StatusOK, Tokens{
			AccessToken: "test-access-token",
			ExpiresIn:   12000,
		}))
		fakeCCServer.Add().Info(fakeTokenServer.URL())
		conf = &Config{}
		conf.API = fakeCCServer.URL()
		logger = lagertest.NewTestLogger("oauth-test")
		cfc = NewCFClient(conf, logger, clock.NewClock())
		err = cfc.Login()

	})
	JustBeforeEach(func() {
		fakeCCServer.Add().GetApp(TestAppId, appStatus, TestSpaceId).Roles(rolesStatus, roles...)
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
			isUserSpaceDeveloperFlag, err = cfc.IsUserSpaceDeveloper(TestUserToken, TestAppId)
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
				Expect(err).To(MatchError(MatchRegexp("Unauthorized")))
			})
		})

		Context("user info endpoint returns non-200 and non-401 statusCode", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusBadRequest
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("Failed to get user info, statuscode :400")))
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
				fakeCCServer.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`failed IsUserSpaceDeveloper for appId\(test-app-id\): getSpaceId failed:.*connection refused`)))
			})
		})

		Context("apps endpoint returns 400 status code", func() {
			BeforeEach(func() {
				appStatus = http.StatusBadRequest
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("400")))
			})
		})

		Context("apps endpoint returns 401 status code", func() {
			BeforeEach(func() {
				appStatus = http.StatusUnauthorized
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("401")))
			})
		})

		Context("roles endpoint returns 200 and 400 status code", func() {
			BeforeEach(func() {
				rolesStatus = http.StatusBadRequest
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`failed IsUserSpaceDeveloper userId\(test-user-id\), spaceId\(test-space-id\):.*page 1:.*cf.Response\[.*cf.Role\]:.*400`)))
			})
		})

		Context("roles endpoint returns 401 status code", func() {
			BeforeEach(func() {
				rolesStatus = http.StatusUnauthorized
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`.*userId\(test-user-id\), spaceId\(test-space-id\):.*cf.Response\[.*cf.Role\]:.*invalid error json`)))
			})
		})

		Context("user is not space developer", func() {
			BeforeEach(func() {
				roles = Roles{{Type: RoleOrganizationManager}}
			})
			It("should return false", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(isUserSpaceDeveloperFlag).To(BeFalse())
			})
		})

		Context("user is space developer", func() {
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
