package publicapiserver_test

import (
	. "autoscaler/api/publicapiserver"
	"autoscaler/fakes"
	"fmt"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/gorilla/mux"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Oauth", func() {
	var (
		req          *http.Request
		resp         *httptest.ResponseRecorder
		oam          *OAuthMiddleware
		router       *mux.Router
		fakeCFClient *fakes.FakeCFClient
	)
	BeforeEach(func() {

		fakeCFClient = &fakes.FakeCFClient{}

		oam = NewOauthMiddleware(lagertest.NewTestLogger("oauth"), fakeCFClient)

		router = mux.NewRouter()
		router.HandleFunc("/", GetTestHandler())
		router.HandleFunc("/v1/apps/{appId}", GetTestHandler())
		router.Use(oam.Middleware)

		resp = httptest.NewRecorder()
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

	Context("AppId is not present", func() {
		BeforeEach(func() {
			req = httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", TEST_USER_TOKEN)
		})
		It("should fail with 400", func() {
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Context("isadminuser check fails", func() {
		BeforeEach(func() {
			fakeCFClient.IsUserAdminReturns(false, fmt.Errorf("Failed to get user scope"))

			req = httptest.NewRequest(http.MethodGet, "/v1/apps/"+TEST_APP_ID, nil)
			req.Header.Add("Authorization", TEST_USER_TOKEN)
		})
		It("should fail with 500", func() {
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
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
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
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
			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
		})
	})

})
