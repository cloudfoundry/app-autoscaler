package auth_test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("XFCC chain verification for custom metrics", func() {
	var (
		pki         *testhelpers.XFCCPKI
		credentials *fakes.FakeCredentials
		bindingDB   *fakes.FakeBindingDB
		request     *http.Request
		logger      *lagertest.TestLogger
	)

	BeforeEach(func() {
		var err error
		pki, err = testhelpers.GenerateXFCCPKI(GinkgoT().TempDir())
		Expect(err).NotTo(HaveOccurred())
		credentials = &fakes.FakeCredentials{}
		credentials.ValidateReturns(true, nil)
		bindingDB = &fakes.FakeBindingDB{}
		logger = lagertest.NewTestLogger("metricsforwarder-xfcc")
		request = httptest.NewRequest("POST", "/v1/apps/an-app-id/metrics", nil)
		request.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		request.Header.Set("X-Forwarded-Client-Cert", base64.StdEncoding.EncodeToString(pki.Leaf.Raw))
	})

	It("accepts a trusted app identity without using Basic auth", func() {
		authenticator, err := auth.New(logger, credentials, bindingDB, pki.Config)
		Expect(err).NotTo(HaveOccurred())
		Expect(authenticator.CheckAuth(request, "an-app-id")).To(Succeed())
		Expect(credentials.ValidateCallCount()).To(BeZero())
	})

	It("uses route parameters in the authentication middleware", func() {
		authenticator, err := auth.New(logger, credentials, bindingDB, pki.Config)
		Expect(err).NotTo(HaveOccurred())
		request = mux.SetURLVars(request, map[string]string{"appid": "an-app-id"})
		response := httptest.NewRecorder()
		authenticator.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})).ServeHTTP(response, request)
		Expect(response.Code).To(Equal(http.StatusNoContent))
	})

	It("rejects trusted certificates without an app identity", func() {
		template := *pki.Leaf
		template.RawSubject = nil
		template.Subject.OrganizationalUnit = []string{"organization:validorg", "space:validspace"}
		leaf, err := pki.Sign(&template, pki.Issuer)
		Expect(err).NotTo(HaveOccurred())
		request.Header.Set("X-Forwarded-Client-Cert", base64.StdEncoding.EncodeToString(leaf.Raw))
		authenticator, err := auth.New(logger, credentials, bindingDB, pki.Config)
		Expect(err).NotTo(HaveOccurred())
		Expect(authenticator.CheckAuth(request, "an-app-id")).To(MatchError(auth.ErrorNoAppIDFound))
		Expect(credentials.ValidateCallCount()).To(BeZero())
	})

	Context("when a trusted neighbour app submits metrics", func() {
		It("does not bypass a failed strategy lookup", func() {
			failure := errors.New("strategy lookup failed")
			bindingDB.GetCustomMetricStrategyByAppIdReturns(models.CustomMetricsSameApp, failure)
			authenticator, err := auth.New(logger, credentials, bindingDB, pki.Config)
			Expect(err).NotTo(HaveOccurred())
			Expect(authenticator.CheckAuth(request, "neighbour-app")).To(MatchError(failure))
		})

		It("does not bypass a failed binding lookup", func() {
			failure := errors.New("binding lookup failed")
			bindingDB.GetCustomMetricStrategyByAppIdReturns(models.CustomMetricsBoundApp, nil)
			bindingDB.IsAppBoundToSameAutoscalerReturns(false, failure)
			authenticator, err := auth.New(logger, credentials, bindingDB, pki.Config)
			Expect(err).NotTo(HaveOccurred())
			Expect(authenticator.CheckAuth(request, "neighbour-app")).To(MatchError(failure))
		})

		It("rejects apps that are not bound to the same service instance", func() {
			bindingDB.GetCustomMetricStrategyByAppIdReturns(models.CustomMetricsBoundApp, nil)
			bindingDB.IsAppBoundToSameAutoscalerReturns(false, nil)
			authenticator, err := auth.New(logger, credentials, bindingDB, pki.Config)
			Expect(err).NotTo(HaveOccurred())
			Expect(authenticator.CheckAuth(request, "neighbour-app")).To(MatchError(auth.ErrorAppNotBound))
		})
	})

	It("rejects foreign certificates without falling back to valid Basic auth", func() {
		foreign, err := testhelpers.GenerateXFCCPKI(GinkgoT().TempDir())
		Expect(err).NotTo(HaveOccurred())
		request.Header.Set("X-Forwarded-Client-Cert", base64.StdEncoding.EncodeToString(foreign.Leaf.Raw))
		authenticator, err := auth.New(logger, credentials, bindingDB, pki.Config)
		Expect(err).NotTo(HaveOccurred())
		response := httptest.NewRecorder()
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})
		authenticator.AuthenticateHandler(next)(response, request, map[string]string{"appid": "an-app-id"})
		Expect(response.Code).To(Equal(http.StatusUnauthorized))
		Expect(nextCalled).To(BeFalse())
		Expect(credentials.ValidateCallCount()).To(BeZero())
	})

	DescribeTable("only disables chain checks in shadow/off modes", func(mode string) {
		pki.Config.VerificationMode = mode
		foreign, err := testhelpers.GenerateXFCCPKI(GinkgoT().TempDir())
		Expect(err).NotTo(HaveOccurred())
		request.Header.Set("X-Forwarded-Client-Cert", base64.StdEncoding.EncodeToString(foreign.Leaf.Raw))
		authenticator, err := auth.New(logger, credentials, bindingDB, pki.Config)
		Expect(err).NotTo(HaveOccurred())
		Expect(authenticator.CheckAuth(request, "an-app-id")).To(Succeed())
		Expect(authenticator.CheckAuth(request, "other-app")).NotTo(Succeed())
		Expect(credentials.ValidateCallCount()).To(BeZero())
	},
		Entry("shadow", "shadow"),
		Entry("off", "off"),
	)

	DescribeTable("rejects malformed XFCC without fallback", func(header string) {
		request.Header.Set("X-Forwarded-Client-Cert", header)
		authenticator, err := auth.New(logger, credentials, bindingDB, pki.Config)
		Expect(err).NotTo(HaveOccurred())
		Expect(authenticator.CheckAuth(request, "an-app-id")).NotTo(Succeed())
		Expect(credentials.ValidateCallCount()).To(BeZero())
	},
		Entry("bad base64", "@@@"),
		Entry("bad DER", base64.StdEncoding.EncodeToString([]byte("not a certificate"))),
	)

	It("rejects an invalid mode at construction", func() {
		_, err := auth.New(logger, credentials, bindingDB, models.XFCCAuth{VerificationMode: "invalid"})
		Expect(err).To(MatchError(ContainSubstring("invalid XFCC verification mode")))
	})
})
