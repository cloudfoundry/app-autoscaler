package publicapiserver_test

import (
	"net/http"
	"net/http/httptest"

	. "autoscaler/api/publicapiserver"
	"autoscaler/fakes"
	"autoscaler/routes"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CheckBinding", func() {
	var (
		req           *http.Request
		resp          *httptest.ResponseRecorder
		cbm           *CheckBindingMiddleware
		router        *mux.Router
		fakeBindingDB *fakes.FakeBindingDB
	)
	BeforeEach(func() {

		fakeBindingDB = &fakes.FakeBindingDB{}

		cbm = NewCheckBindingMiddleware(lagertest.NewTestLogger("checkbinding_middleware"), fakeBindingDB)

		router = mux.NewRouter()
		router.HandleFunc("/", GetTestHandler())
		router.HandleFunc(routes.PublicApiPolicyPath, GetTestHandler())
		router.HandleFunc(routes.PublicApiCustomMetricsCredentialPath, GetTestHandler())
		router.Use(cbm.CheckServiceBinding)

		resp = httptest.NewRecorder()
	})

	JustBeforeEach(func() {
		router.ServeHTTP(resp, req)
	})
	Context("DeleteCustomMetricsCredential", func() {
		Context("Binding exists", func() {
			BeforeEach(func() {
				fakeBindingDB.CheckServiceBindingStub = func(appId string) bool {
					return true
				}
				req = httptest.NewRequest(http.MethodDelete, "/v1/apps/"+TEST_APP_ID+"/custom_metrics_credential", nil)
			})
			It("should fail with 403", func() {
				Expect(resp.Code).To(Equal(http.StatusForbidden))
			})
		})
		Context("Binding does not exists", func() {
			BeforeEach(func() {
				fakeBindingDB.CheckServiceBindingStub = func(appId string) bool {
					return false
				}
				req = httptest.NewRequest(http.MethodDelete, "/v1/apps/"+TEST_APP_ID+"/custom_metrics_credential", nil)
			})
			It("should succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})
	})
	Context("Other routes", func() {
		Context("Binding exists", func() {
			BeforeEach(func() {
				fakeBindingDB.CheckServiceBindingStub = func(appId string) bool {
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
				fakeBindingDB.CheckServiceBindingStub = func(appId string) bool {
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
