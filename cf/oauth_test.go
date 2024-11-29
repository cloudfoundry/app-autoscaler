package cf_test

import (
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf/mocks"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3/lagertest"
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
		conf      *cf.Config
		cfc       cf.CFClient
		err       error
		userToken string
		logger    *lagertest.TestLogger

		isUserSpaceDeveloperFlag bool
		isUserAdminFlag          bool

		fakeCCServer    *mocks.Server
		fakeTokenServer *mocks.Server

		userInfoStatus   int
		userInfoResponse interface{}

		userScopeStatus   int
		userScopeResponse userScope

		appStatus int

		rolesStatus int
		roles       cf.Roles
	)

	BeforeEach(func() {
		userToken = TestUserToken
		appStatus = http.StatusOK
		userScopeStatus = http.StatusOK
		userInfoStatus = http.StatusOK
		rolesStatus = http.StatusOK
		roles = cf.Roles{{Type: cf.RoleSpaceDeveloper}}
		userInfoResponse = userInfo{
			UserId: TestUserId,
		}

		fakeCCServer = mocks.NewServer()
		fakeTokenServer = mocks.NewServer()
		fakeTokenServer.RouteToHandler(http.MethodGet, "/userinfo", ghttp.RespondWithJSONEncodedPtr(&userInfoStatus, &userInfoResponse))
		fakeTokenServer.RouteToHandler(http.MethodPost, "/introspect", ghttp.RespondWithJSONEncodedPtr(&userScopeStatus, &userScopeResponse))
		fakeTokenServer.RouteToHandler("POST", cf.PathCFAuth, ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
			AccessToken: "test-access-token",
			ExpiresIn:   12000,
		}))
		fakeCCServer.Add().Info(fakeTokenServer.URL())
		conf = &cf.Config{}
		conf.API = fakeCCServer.URL()
		logger = lagertest.NewTestLogger("oauth-test")
		cfc = cf.NewCFClient(conf, logger, clock.NewClock())
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

		Context("user info endpoint", func() {
			Context("returns 401 statusCode", func() {
				BeforeEach(func() {
					userInfoStatus = http.StatusUnauthorized
				})
				It("should false", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(isUserSpaceDeveloperFlag).To(BeFalse())
				})
			})

			Context("uua returns not found", func() {
				BeforeEach(func() {
					userInfoStatus = http.StatusNotFound
				})
				It("should return false", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(isUserSpaceDeveloperFlag).To(BeFalse())
				})
			})
			Context("response code 404 but not from cc", func() {
				BeforeEach(func() {
					userInfoStatus = http.StatusNotFound
				})
				It("should return false", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(isUserSpaceDeveloperFlag).To(BeFalse())
				})
			})

			Context("returns non 200,401,404 statusCode", func() {
				BeforeEach(func() {
					userInfoStatus = http.StatusBadRequest
				})
				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(MatchRegexp("Failed to get user info, statuscode :400")))
				})
			})

			Context("is not in json format", func() {
				BeforeEach(func() {
					fakeTokenServer.RouteToHandler(http.MethodGet, "/userinfo", ghttp.RespondWith(http.StatusOK, "non-json-response"))
				})
				It("should error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("cc server is not reachable", func() {
			BeforeEach(func() {
				fakeCCServer.Close()
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`failed IsUserSpaceDeveloper for appId\(test-app-id\): .*connection refused`)))
			})
		})

		Context("apps endpoint", func() {
			Context("returns 400 status code", func() {
				BeforeEach(func() {
					appStatus = http.StatusBadRequest
				})
				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(MatchRegexp("400")))
				})
			})

			Context("returns 401 status code", func() {
				BeforeEach(func() {
					appStatus = http.StatusUnauthorized
				})
				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(MatchRegexp(`CF-NotAuthenticated`)))
				})
			})

			Context("returns 404 status code", func() {
				BeforeEach(func() {
					appStatus = http.StatusNotFound
				})
				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(MatchRegexp(`CF-ResourceNotFound`)))
				})
			})
		})
		Context("roles endpoint", func() {
			Context("roles endpoint 400 status code", func() {
				BeforeEach(func() {
					rolesStatus = http.StatusBadRequest
				})
				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(MatchRegexp(`failed IsUserSpaceDeveloper userId\(test-user-id\), spaceId\(test-space-id\):.*page 1:.*cf.Response\[.*cf.Role\]:.*400`)))
				})
			})

			Context("roles endpoint returns 404 status code", func() {
				BeforeEach(func() {
					rolesStatus = http.StatusNotFound
				})
				It("should return false", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(isUserSpaceDeveloperFlag).To(BeFalse())
				})
			})
		})

		Context("user is not space developer", func() {
			BeforeEach(func() {
				roles = cf.Roles{{Type: cf.RoleOrganizationManager}}
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
				Expect(err).To(MatchError(ContainSubstring("400")))
			})
		})

		Context("userscope response is not in json format", func() {
			BeforeEach(func() {
				fakeTokenServer.RouteToHandler(http.MethodPost, "/introspect", ghttp.RespondWith(http.StatusOK, "non-json-response"))
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
					Scope: []string{cf.CCAdminScope},
				}
			})
			It("should return true", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(isUserAdminFlag).To(BeTrue())
			})
		})
	})
})
