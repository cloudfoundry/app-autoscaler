package publicapiserver_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	"code.cloudfoundry.org/lager/v3/lagertest"
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

		When("Authorization header is not preset", func() {
			BeforeEach(func() {
				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
			})
			It("should fail with 401", func() {
				CheckResponse(resp, http.StatusUnauthorized, models.ErrorResponse{
					Code:    "Unauthorized",
					Message: "Authorization header is not present",
				})

			})
		})

		Context("Invalid user token format", func() {
			When("Authorization header does not contain a bearer token", func() {
				BeforeEach(func() {
					req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
					req.Header.Add("Authorization", INVALID_USER_TOKEN_WITHOUT_BEARER)
				})
				It("should fail with 401", func() {
					Eventually(logger.Buffer).Should(Say("authorization credentials should specify bearer scheme"))
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
					Eventually(logger.Buffer).Should(Say("authorization credentials should contain scheme and token separated by space"))
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
		Context("App does not exist", func() {
			BeforeEach(func() {
				fakeCFClient.IsUserAdminReturns(false, nil)
				fakeCFClient.IsUserSpaceDeveloperReturns(false, cf.CfResourceNotFound)
				req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
				req.Header.Add("Authorization", TEST_USER_TOKEN)
			})
			It("should fail with 404", func() {
				CheckResponse(resp, http.StatusNotFound, models.ErrorResponse{
					Code:    "App not found",
					Message: "The app guid supplied does not exist",
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
				By("checking if the token is from an admin user")
				Expect(fakeCFClient.IsUserAdminCallCount()).To(Equal(1))
				Expect(fakeCFClient.IsUserAdminArgsForCall(0)).To(Equal(TEST_BEARER_TOKEN))

				CheckResponse(resp, http.StatusInternalServerError, models.ErrorResponse{
					Code:    http.StatusText(http.StatusInternalServerError),
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
				By("checking if the token is from an admin user")
				Expect(fakeCFClient.IsUserAdminCallCount()).To(Equal(1))
				Expect(fakeCFClient.IsUserAdminArgsForCall(0)).To(Equal(TEST_BEARER_TOKEN))

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
		Context("isspacedeveloper check fails unauthorised", func() {
			BeforeEach(func() {
				fakeCFClient.IsUserAdminReturns(false, nil)
				fakeCFClient.IsUserSpaceDeveloperReturns(false, fmt.Errorf("wrapped error %w", cf.ErrUnauthorized))

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

})

func GetTestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Success"))
		Expect(err).NotTo(HaveOccurred())
	}
}

func CheckResponse(resp *httptest.ResponseRecorder, statusCode int, errResponse models.ErrorResponse) {
	GinkgoHelper()
	Expect(resp.Code).To(Equal(statusCode))
	var errResp models.ErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&errResp)
	Expect(err).NotTo(HaveOccurred())
	Expect(errResp).To(Equal(errResponse))
}
