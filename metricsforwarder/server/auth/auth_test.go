package auth_test

import (
	"bytes"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"net/http"
	"net/http/httptest"
)

var _ = Describe("Authentication", func() {

	var (
		authTest                  *auth.Auth
		fakeCredentials           *fakes.FakeCredentials
		fakeBindingDB             *fakes.FakeBindingDB
		resp                      *httptest.ResponseRecorder
		req                       *http.Request
		body                      []byte
		vars                      map[string]string
		testAppId                 string
		pki                       *testhelpers.XFCCPKI
		instanceIdentityChainPath string
		additionalCallerCertPath  string
	)

	BeforeEach(func() {
		fakeCredentials = &fakes.FakeCredentials{}
		fakeBindingDB = &fakes.FakeBindingDB{}
		vars = make(map[string]string)
		testAppId = "an-app-id"
		resp = httptest.NewRecorder()
		var err error
		pki, err = testhelpers.GenerateXFCCPKI(GinkgoT().TempDir())
		Expect(err).NotTo(HaveOccurred())
		instanceIdentityChainPath = pki.Config.InstanceCertPath
		additionalCallerCertPath = filepath.Join(filepath.Dir(instanceIdentityChainPath), "additional-caller.pem")
		// Issue another certificate for the same app under the instance-identity intermediate.
		callerCertTemplate := *pki.Leaf
		callerCertTemplate.SerialNumber = big.NewInt(4)
		additionalCallerCert, err := pki.Sign(&callerCertTemplate, pki.Issuer)
		Expect(err).NotTo(HaveOccurred())
		Expect(testhelpers.WriteXFCCCertificates(additionalCallerCertPath, additionalCallerCert)).To(Succeed())
	})
	JustBeforeEach(func() {
		logger := lager.NewLogger("auth-test")
		var err error
		authTest, err = auth.New(logger, fakeCredentials, fakeBindingDB, pki.Config)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Basic Auth tests for publish metrics endpoint", func() {
		Context("a request to publish custom metrics comes", func() {
			Context("credentials are valid", func() {
				It("should validate the credentials", func() {
					req = CreateRequest(body, testAppId)
					req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
					vars["appid"] = "an-app-id"
					nextCalled := 0
					nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						nextCalled = nextCalled + 1
					})

					fakeCredentials.ValidateReturns(true, nil)

					authTest.AuthenticateHandler(nextFunc)(resp, req, vars)
					Expect(resp.Code).To(Equal(http.StatusOK))
					Expect(nextCalled).To(Equal(1))
				})
			})

			Context("credentials are valid but db error occurs", func() {
				It("should validate the credentials", func() {
					req = CreateRequest(body, testAppId)
					req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
					vars["appid"] = "an-app-id"
					nextCalled := 0
					nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						nextCalled = nextCalled + 1
					})

					fakeCredentials.ValidateReturns(true, errors.New("db error"))

					authTest.AuthenticateHandler(nextFunc)(resp, req, vars)
					Expect(resp.Code).To(Equal(http.StatusUnauthorized))
					Expect(nextCalled).To(Equal(0))
				})
			})

			Context("credentials are invalid", func() {
				It("should validate the credentials", func() {
					req = CreateRequest(body, testAppId)
					req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
					vars["appid"] = "an-app-id"
					nextCalled := 0
					nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						nextCalled = nextCalled + 1
					})

					fakeCredentials.ValidateReturns(false, nil)

					authTest.AuthenticateHandler(nextFunc)(resp, req, vars)
					Expect(resp.Code).To(Equal(http.StatusUnauthorized))
					Expect(nextCalled).To(Equal(0))
				})

			})

		})
	})
	Describe("MTLS Auth tests for publish metrics endpoint", func() {
		Context("XFCC contains the trusted leaf from the instance-identity chain", func() {
			It("should call next handler", func() {
				req = CreateRequest(body, testAppId)
				req.Header.Add("X-Forwarded-Client-Cert", MustReadXFCCcert(instanceIdentityChainPath))
				vars["appid"] = "an-app-id"
				nextCalled := 0
				nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					nextCalled = nextCalled + 1
				})

				authTest.AuthenticateHandler(nextFunc)(resp, req, vars)

				Expect(policyDB.GetCredentialCallCount()).To(Equal(0))
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(Equal(1))
			})
		})

		Context("XFCC contains another trusted caller certificate for the same app", func() {
			It("should call next handler", func() {
				req = CreateRequest(body, testAppId)
				req.Header.Add("X-Forwarded-Client-Cert", MustReadXFCCcert(additionalCallerCertPath))
				vars["appid"] = "an-app-id"
				nextCalled := 0
				nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					nextCalled = nextCalled + 1
				})

				authTest.AuthenticateHandler(nextFunc)(resp, req, vars)

				Expect(policyDB.GetCredentialCallCount()).To(Equal(0))
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(Equal(1))
			})
		})

		Context("correct xfcc header including \"'s around the cert", func() {
			It("should call next handler", func() {
				req = CreateRequest(body, testAppId)
				req.Header.Add("X-Forwarded-Client-Cert", fmt.Sprintf("%q", MustReadXFCCcert(instanceIdentityChainPath)))
				vars["appid"] = "an-app-id"
				nextCalled := 0
				nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					nextCalled = nextCalled + 1
				})

				authTest.AuthenticateHandler(nextFunc)(resp, req, vars)

				Expect(policyDB.GetCredentialCallCount()).To(Equal(0))
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(Equal(1))
			})
		})

		Context("valid cert with wrong app-id is supplied", func() {
			It("should return status code 403", func() {
				req = CreateRequest(body, testAppId)
				req.Header.Add("X-Forwarded-Client-Cert", MustReadXFCCcert(instanceIdentityChainPath))
				nextCalled := 0
				nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					nextCalled = nextCalled + 1
				})

				vars["appid"] = "wrong-an-app-id"
				authTest.AuthenticateHandler(nextFunc)(resp, req, vars)

				Expect(policyDB.GetCredentialCallCount()).To(Equal(0))
				Expect(resp.Code).To(Equal(http.StatusForbidden))
				Expect(resp.Body.String()).To(Equal(`{"code":"Forbidden","message":"Unauthorized"}`))
				Expect(nextCalled).To(Equal(0))
			})
		})

		Context("Request from neighbour (different) app arrives for app B", func() {
			When("custom-metrics-submission-strategy is not set in the scaling policy", func() {
				It("It should not call next handler and return with status code 403", func() {
					testAppId = "app-to-scale-id"
					req = CreateRequest(body, testAppId)
					vars["appid"] = testAppId
					req.Header.Add("X-Forwarded-Client-Cert", MustReadXFCCcert(additionalCallerCertPath))
					fakeBindingDB.GetCustomMetricStrategyByAppIdReturns(models.CustomMetricsSameApp, nil)
					nextCalled := 0
					nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						nextCalled = nextCalled + 1
					})

					authTest.AuthenticateHandler(nextFunc)(resp, req, vars)

					Expect(policyDB.GetCredentialCallCount()).To(Equal(0))
					Expect(resp.Code).To(Equal(http.StatusForbidden))
					Expect(resp.Body.String()).To(Equal(`{"code":"Forbidden","message":"Unauthorized"}`))
					Expect(nextCalled).To(Equal(0))
				})
			})
			Context("custom-metrics-submission-strategy is set to bound_app in the scaling policy", func() {
				It("It should call next handler and return with status code 200", func() {
					req = CreateRequest(body, testAppId)
					testAppId = "app-to-scale-id"
					vars["appid"] = testAppId
					req.Header.Add("X-Forwarded-Client-Cert", MustReadXFCCcert(additionalCallerCertPath))
					fakeBindingDB.GetCustomMetricStrategyByAppIdReturns(models.CustomMetricsBoundApp, nil)
					nextCalled := 0
					nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						nextCalled = nextCalled + 1
					})
					fakeBindingDB.IsAppBoundToSameAutoscalerReturns(true, nil)

					authTest.AuthenticateHandler(nextFunc)(resp, req, vars)

					Expect(fakeBindingDB.IsAppBoundToSameAutoscalerCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusOK))
					Expect(resp.Body.String()).To(BeEmpty())
					Expect(nextCalled).To(Equal(1))
				})
			})

		})
	})
	Describe("CheckAuth with credentials present (basic auth 'enabled' or 'only_existing_bindings')", func() {
		Context("when mTLS fails (no XFCC header) and Basic Auth credentials are provided", func() {
			BeforeEach(func() {
				fakeCredentials.ValidateReturns(true, nil)
			})

			It("should call credentials.Validate as fallback", func() {
				req = CreateRequest(body, testAppId)
				req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")

				err := authTest.CheckAuth(req, testAppId)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeCredentials.ValidateCallCount()).To(Equal(1))
				_, appID, cred := fakeCredentials.ValidateArgsForCall(0)
				Expect(appID).To(Equal(testAppId))
				Expect(cred.Username).To(Equal("username"))
				Expect(cred.Password).To(Equal("password"))
			})
		})
		Context("when mTLS fails and no Basic Auth credentials are provided", func() {
			It("should return an error", func() {
				req = CreateRequest(body, testAppId)

				err := authTest.CheckAuth(req, testAppId)
				Expect(err).To(HaveOccurred())
			})
		})
	})
	Describe("CheckAuth with credentials nil (basic auth 'disabled')", func() {
		// Override the outer JustBeforeEach to construct an Auth with a nil credentials provider,
		// mimicking how main.go leaves the provider unset when basic-auth is globally disabled.
		JustBeforeEach(func() {
			logger := lager.NewLogger("auth-test")
			var err error
			authTest, err = auth.New(logger, nil, fakeBindingDB, pki.Config)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when a request with Basic-Auth header but no XFCC header arrives", func() {
			It("should not panic and respond with HTTP 401", func() {
				req = CreateRequest(body, testAppId)
				req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
				vars["appid"] = testAppId
				nextCalled := 0
				nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					nextCalled = nextCalled + 1
				})

				Expect(func() {
					authTest.AuthenticateHandler(nextFunc)(resp, req, vars)
				}).NotTo(Panic())
				Expect(resp.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(Equal(0))
			})
		})
		Context("when a request without any auth header arrives", func() {
			It("should respond with HTTP 401", func() {
				req = CreateRequest(body, testAppId)
				vars["appid"] = testAppId
				nextCalled := 0
				nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					nextCalled = nextCalled + 1
				})

				authTest.AuthenticateHandler(nextFunc)(resp, req, vars)

				Expect(resp.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(Equal(0))
			})
		})
		Context("when a request with a valid mTLS/XFCC header arrives", func() {
			It("should respond with HTTP 200 (mTLS path is unaffected by nil credentials)", func() {
				req = CreateRequest(body, testAppId)
				req.Header.Add("X-Forwarded-Client-Cert", MustReadXFCCcert(instanceIdentityChainPath))
				vars["appid"] = testAppId
				nextCalled := 0
				nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					nextCalled = nextCalled + 1
				})

				authTest.AuthenticateHandler(nextFunc)(resp, req, vars)

				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(Equal(1))
			})
		})
	})
})

func MustReadXFCCcert(fileName string) string {
	file, err := os.ReadFile(fileName)
	Expect(err).ShouldNot(HaveOccurred())
	block, _ := pem.Decode(file)
	Expect(block).ShouldNot(BeNil())
	return base64.StdEncoding.EncodeToString(block.Bytes)
}

func CreateRequest(body []byte, appId string) *http.Request {
	req, err := http.NewRequest(http.MethodPost, serverUrl+"/v1/apps/"+appId+"/metrics", bytes.NewReader(body))
	Expect(err).ToNot(HaveOccurred())
	req.Header.Add("Content-Type", "application/json")
	return req
}
