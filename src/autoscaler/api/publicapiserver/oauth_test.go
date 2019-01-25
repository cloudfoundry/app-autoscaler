package publicapiserver_test

import (
	. "autoscaler/api/publicapiserver"
	"net/http"
	"net/http/httptest"
	"regexp"

	"github.com/gorilla/mux"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Oauth", func() {
	var (
		req    *http.Request
		resp   *httptest.ResponseRecorder
		oam    *OAuthMiddleware
		router *mux.Router

		ccTestServer    *ghttp.Server
		tokenTestServer *ghttp.Server

		ccInfoStatus   int
		ccInfoResponse info

		userInfoStatus   int
		userInfoResponse userInfo

		userScopeStatus   int
		userScopeResponse userScope

		spaceDeveloperStatus   int
		spaceDeveloperResponse spaceDeveoper
	)
	BeforeEach(func() {
		ccTestServer = ghttp.NewServer()
		tokenTestServer = ghttp.NewServer()

		conf.CF.API = ccTestServer.URL()

		ccTestServer.RouteToHandler(http.MethodGet, "/v2/info", ghttp.RespondWithJSONEncodedPtr(&ccInfoStatus, &ccInfoResponse))

		tokenTestServer.RouteToHandler(http.MethodGet, "/userinfo", ghttp.RespondWithJSONEncodedPtr(&userInfoStatus, &userInfoResponse))

		tokenTestServer.RouteToHandler(http.MethodPost, "/check_token", ghttp.RespondWithJSONEncodedPtr(&userScopeStatus, &userScopeResponse))

		spaceDeveloperPathMatcher, _ := regexp.Compile("/v2/users/[A-Za-z0-9\\-]+/spaces")
		ccTestServer.RouteToHandler(http.MethodGet, spaceDeveloperPathMatcher, ghttp.RespondWithJSONEncodedPtr(&spaceDeveloperStatus, &spaceDeveloperResponse))

		resp = httptest.NewRecorder()
		oam = NewOauthMiddleware(logger, conf)

		router = mux.NewRouter()
		router.HandleFunc("/", GetTestHandler())
		router.HandleFunc("/v1/apps/{appId}", GetTestHandler())
		router.Use(oam.Middleware)

	})
	AfterEach(func() {
		ccTestServer.Close()
	})

	JustBeforeEach(func() {
		router.ServeHTTP(resp, req)
	})

	Context("Authorization Header is not provided", func() {
		BeforeEach(func() {
			req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
		})
		It("should fail with 401", func() {
			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Context("App Id is not present", func() {
		BeforeEach(func() {
			req = httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", TEST_USER_TOKEN)
		})
		It("should fail with 400", func() {
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Context("cf info endpoint fails", func() {
		BeforeEach(func() {
			ccInfoStatus = http.StatusInternalServerError

			req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
			req.Header.Add("Authorization", TEST_USER_TOKEN)
		})
		It("should fail with 500", func() {
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Context("user token is invalid", func() {
		BeforeEach(func() {
			ccInfoStatus = http.StatusOK
			ccInfoResponse = info{
				TokenEndpoint: tokenTestServer.URL(),
			}

			userInfoStatus = http.StatusUnauthorized

			req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
			req.Header.Add("Authorization", "bearer invalid-user-token")
		})
		It("should fail with 401", func() {
			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Context("user is admin", func() {
		BeforeEach(func() {
			ccInfoStatus = http.StatusOK
			ccInfoResponse = info{
				TokenEndpoint: tokenTestServer.URL(),
			}

			userInfoStatus = http.StatusOK
			userInfoResponse = userInfo{
				UserId: TEST_USER_ID,
			}

			userScopeStatus = http.StatusOK
			userScopeResponse = userScope{
				Scope: []string{"cloud_controller.admin"},
			}

			req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
			req.Header.Add("Authorization", TEST_USER_TOKEN)
		})
		It("should succeed with 200", func() {
			Expect(resp.Code).To(Equal(http.StatusOK))
		})
	})

	Context("user is not admin nor space developer", func() {
		BeforeEach(func() {
			ccInfoStatus = http.StatusOK
			ccInfoResponse = info{
				TokenEndpoint: tokenTestServer.URL(),
			}

			userInfoStatus = http.StatusOK
			userInfoResponse = userInfo{
				UserId: TEST_USER_ID,
			}

			userScopeStatus = http.StatusOK
			userScopeResponse = userScope{
				Scope: []string{},
			}

			spaceDeveloperStatus = http.StatusOK
			spaceDeveloperResponse = spaceDeveoper{
				Total: 0,
			}

			req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
			req.Header.Add("Authorization", TEST_USER_TOKEN)
		})

		It("should fail with 401", func() {
			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Context("User is space developer", func() {
		BeforeEach(func() {
			ccInfoStatus = http.StatusOK
			ccInfoResponse = info{
				TokenEndpoint: tokenTestServer.URL(),
			}

			userInfoStatus = http.StatusOK
			userInfoResponse = userInfo{
				UserId: TEST_USER_ID,
			}

			userScopeStatus = http.StatusOK
			userScopeResponse = userScope{
				Scope: []string{},
			}

			spaceDeveloperStatus = http.StatusOK
			spaceDeveloperResponse = spaceDeveoper{
				Total: 1,
			}

			req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
			req.Header.Add("Authorization", TEST_USER_TOKEN)
		})

		It("should succeed", func() {
			Expect(resp.Code).To(Equal(http.StatusOK))
		})
	})
})
