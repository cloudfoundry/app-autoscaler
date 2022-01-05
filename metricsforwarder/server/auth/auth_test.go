package auth_test

import (
	"bytes"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server/auth"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"net/http"
	"net/http/httptest"
)

var _ = Describe("Authentication", func() {

	var (
		authTest        *auth.Auth
		fakeCredentials *fakes.FakeCredentials
		resp            *httptest.ResponseRecorder
		req             *http.Request
		body            []byte
		vars            map[string]string
	)

	BeforeEach(func() {
		fakeCredentials = &fakes.FakeCredentials{}
		vars = make(map[string]string)
		resp = httptest.NewRecorder()
	})

	JustBeforeEach(func() {
		logger := lager.NewLogger("auth-test")
		var err error
		authTest, err = auth.New(logger, fakeCredentials)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Basic Auth tests for publish metrics endpoint", func() {
		Context("a request to publish custom metrics comes", func() {
			Context("credentials are valid", func() {
				It("should validate the credentials", func() {
					req = CreateRequest(body)
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
					req = CreateRequest(body)
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
					req = CreateRequest(body)
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
		const validClientCert1 = "../../../../../test-certs/validmtls_client-1.crt"
		Context("correct xfcc header with correct CA is supplied for cert 1", func() {
			It("should call next handler", func() {
				req = CreateRequest(body)
				req.Header.Add("X-Forwarded-Client-Cert", MustReadXFCCcert(validClientCert1))
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

		Context("correct xfcc header with correct CA is supplied for cert 2", func() {
			It("should call next handler", func() {
				req = CreateRequest(body)
				const validClientCert2 = "../../../../../test-certs/validmtls_client-2.crt"
				req.Header.Add("X-Forwarded-Client-Cert", MustReadXFCCcert(validClientCert2))
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
				req = CreateRequest(body)
				req.Header.Add("X-Forwarded-Client-Cert", fmt.Sprintf("%q", MustReadXFCCcert(validClientCert1)))
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
				req = CreateRequest(body)
				req.Header.Add("X-Forwarded-Client-Cert", MustReadXFCCcert(validClientCert1))
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
	})

})

func MustReadXFCCcert(fileName string) string {
	file, err := ioutil.ReadFile(fileName)
	Expect(err).ShouldNot(HaveOccurred())
	block, _ := pem.Decode(file)
	Expect(block).ShouldNot(BeNil())
	return base64.StdEncoding.EncodeToString(block.Bytes)
}

func CreateRequest(body []byte) *http.Request {
	req, err := http.NewRequest(http.MethodPost, serverUrl+"/v1/apps/an-app-id/metrics", bytes.NewReader(body))
	Expect(err).ToNot(HaveOccurred())
	req.Header.Add("Content-Type", "application/json")
	return req
}
