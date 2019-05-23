package cf_test

import (
	. "autoscaler/cf"
	"net/http"
	"regexp"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

type userInfo struct {
	UserId string `json:"user_id"`
}

type userScope struct {
	Scope []string `json:"scope"`
}

type spaceDeveoper struct {
	Total int `json:"total_results"`
}

const (
	TEST_USER_TOKEN = "bearer test-user-token"
	TEST_APP_ID     = "test-app-id"
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

	ccInfoStatus   int
	ccInfoResponse Tokens

	userInfoStatus   int
	userInfoResponse userInfo

	userScopeStatus   int
	userScopeResponse userScope

	spaceDeveloperStatus      int
	spaceDeveloperResponse    spaceDeveoper
	spaceDeveloperPathMatcher *regexp.Regexp
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

		spaceDeveloperPathMatcher, _ = regexp.Compile("/v2/users/[A-Za-z0-9\\-]+/spaces")
		fakeCCServer.RouteToHandler(http.MethodGet, spaceDeveloperPathMatcher, ghttp.RespondWithJSONEncodedPtr(&spaceDeveloperStatus, &spaceDeveloperResponse))

		conf = &CFConfig{}
		conf.API = fakeCCServer.URL()
		logger = lagertest.NewTestLogger("oauth-test")
		cfc = NewCFClient(conf, logger, clock.NewClock())
		cfc.Login()
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

		Context("space developer check endpoint returns non-200 status code", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}
				spaceDeveloperStatus = http.StatusBadRequest
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Failed to get space developer permission, statusCode : 400"))
			})
		})

		Context("space developer check endpoint returns non json response", func() {
			BeforeEach(func() {
				userInfoStatus = http.StatusOK
				userInfoResponse = userInfo{
					UserId: TEST_USER_ID,
				}
				fakeCCServer.RouteToHandler(http.MethodGet, spaceDeveloperPathMatcher, ghttp.RespondWith(http.StatusOK, "non-json-response"))

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

				spaceDeveloperStatus = http.StatusOK
				spaceDeveloperResponse = spaceDeveoper{
					Total: 0,
				}

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

				spaceDeveloperStatus = http.StatusOK
				spaceDeveloperResponse = spaceDeveoper{
					Total: 1,
				}

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
