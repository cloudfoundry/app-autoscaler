package publicapiserver_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/publicapiserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
)

var _ = Describe("Middleware", func() {
	var (
		req              *http.Request
		resp             *httptest.ResponseRecorder
		mw               *Middleware
		router           *mux.Router
		fakeCFClient     *fakes.FakeCFClient
		logger           *lagertest.TestLogger
		checkBindingFunc api.CheckBindingFunc
	)
	Describe("Oauth", func() {

		BeforeEach(func() {

			fakeCFClient = &fakes.FakeCFClient{}
			logger = lagertest.NewTestLogger("oauth")
			mw = NewMiddleware(logger, fakeCFClient, func(appId string) bool {
				return true
			}, "")

			router = mux.NewRouter()
			router.HandleFunc("/", GetTestHandler())
			router.HandleFunc("/v1/apps/{appId}", GetTestHandler())
			router.Use(mw.Oauth)

			resp = httptest.NewRecorder()
		})

		JustBeforeEach(func() {
			router.ServeHTTP(resp, req)
		})

		Context("User token is not present in Authorization header", func() {
			BeforeEach(func() {
				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
			})
			It("should fail with 401", func() {
				CheckResponse(resp, http.StatusUnauthorized, models.ErrorResponse{
					Code:    "Unauthorized",
					Message: "User token is not present in Authorization header",
				})

			})
		})
		Context("Invalid user token format", func() {
			Context("when user token is not a bearer token", func() {
				BeforeEach(func() {
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
					req.Header.Add("Authorization", INVALID_USER_TOKEN_WITHOUT_BEARER)
				})
				It("should fail with 401", func() {
					Eventually(logger.Buffer).Should(Say("Token should start with bearer"))
					CheckResponse(resp, http.StatusUnauthorized, models.ErrorResponse{
						Code:    "Unauthorized",
						Message: "Invalid bearer token",
					})
				})
			})

			Context("when user token contains more than two parts separated by space", func() {
				BeforeEach(func() {
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
					req.Header.Add("Authorization", INVALID_USER_TOKEN)
				})
				It("should fail with 401", func() {
					Eventually(logger.Buffer).Should(Say("Token should contain two parts separated by space"))
					CheckResponse(resp, http.StatusUnauthorized, models.ErrorResponse{
						Code:    "Unauthorized",
						Message: "Invalid bearer token",
					})
				})
			})
		})

		Context("AppId is not present", func() {
			BeforeEach(func() {
				req = httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Authorization", TEST_USER_TOKEN)
			})
			It("should fail with 400", func() {
				CheckResponse(resp, http.StatusBadRequest, models.ErrorResponse{
					Code:    "Bad Request",
					Message: "Malformed or missing appId",
				})
			})
		})

		Context("isadminuser check fails", func() {
			BeforeEach(func() {
				fakeCFClient.IsUserAdminReturns(false, fmt.Errorf("Failed to get user scope"))

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
				req.Header.Add("Authorization", TEST_USER_TOKEN)
			})
			It("should fail with 500", func() {
				CheckResponse(resp, http.StatusInternalServerError, models.ErrorResponse{
					Code:    "Internal-Server-Error",
					Message: "Failed to check if user is admin",
				})
			})
		})

		Context("user is admin", func() {
			BeforeEach(func() {
				fakeCFClient.IsUserAdminReturns(true, nil)

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
				req.Header.Add("Authorization", TEST_USER_TOKEN)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("isspacedeveloper check fails", func() {
			BeforeEach(func() {
				fakeCFClient.IsUserAdminReturns(false, nil)
				fakeCFClient.IsUserSpaceDeveloperReturns(false, fmt.Errorf("failed to check space developer permissions"))

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
				req.Header.Add("Authorization", TEST_USER_TOKEN)
			})
			It("should fail with 500", func() {
				CheckResponse(resp, http.StatusInternalServerError, models.ErrorResponse{
					Code:    "Internal-Server-Error",
					Message: "Failed to check space developer permission",
				})
			})
		})

		Context("user is space developer", func() {
			BeforeEach(func() {
				fakeCFClient.IsUserAdminReturns(false, nil)
				fakeCFClient.IsUserSpaceDeveloperReturns(true, nil)

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
				req.Header.Add("Authorization", TEST_USER_TOKEN)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("user is neither admin nor space developer", func() {
			BeforeEach(func() {
				fakeCFClient.IsUserAdminReturns(false, nil)
				fakeCFClient.IsUserSpaceDeveloperReturns(false, nil)

				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
				req.Header.Add("Authorization", TEST_USER_TOKEN)
			})
			It("should fail with 401", func() {
				CheckResponse(resp, http.StatusUnauthorized, models.ErrorResponse{
					Code:    "Unauthorized",
					Message: "You are not authorized to perform the requested action",
				})
			})
		})
	})

	Describe("CheckBinding", func() {

		JustBeforeEach(func() {
			mw = NewMiddleware(lagertest.NewTestLogger("middleware"), fakeCFClient, checkBindingFunc, "")

			router = mux.NewRouter()
			router.HandleFunc("/", GetTestHandler())
			router.HandleFunc(routes.PublicApiPolicyPath, GetTestHandler())
			router.Use(mw.CheckServiceBinding)

			resp = httptest.NewRecorder()
			router.ServeHTTP(resp, req)
		})
		Context("policy api", func() {
			Context("Binding exists", func() {
				BeforeEach(func() {
					checkBindingFunc = func(appId string) bool {
						return true
					}
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/policy", nil)
				})
				It("should succeed with 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
				})
			})
			Context("Binding does not exists", func() {
				BeforeEach(func() {
					checkBindingFunc = func(appId string) bool {
						return false
					}
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID+"/policy", nil)
				})
				It("should fail with 403", func() {
					Expect(resp.Code).To(Equal(http.StatusForbidden))
				})
			})
		})
	})

	Describe("RejectCredentialOperationInServiceOffering", func() {
		BeforeEach(func() {

			fakeCFClient = &fakes.FakeCFClient{}
			logger = lagertest.NewTestLogger("oauth")
			mw = NewMiddleware(logger, fakeCFClient, func(appId string) bool {
				return true
			}, "")

			router = mux.NewRouter()
			router.HandleFunc("/", GetTestHandler())
			router.HandleFunc("/v1/apps/{appId}", GetTestHandler())
			router.Use(mw.RejectCredentialOperationInServiceOffering)

			resp = httptest.NewRecorder()
		})

		JustBeforeEach(func() {
			router.ServeHTTP(resp, req)
		})
	})

})
