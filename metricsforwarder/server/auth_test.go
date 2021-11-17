package server_test

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
	"net/http/httptest"

	"github.com/patrickmn/go-cache"
)

var _ = Describe("Authentication", func() {

	var (
		authTest        *auth.Auth
		credentialCache cache.Cache
		policyDB        *fakes.FakePolicyDB
		fakeCredentials *fakes.FakeCredentials
		resp            *httptest.ResponseRecorder
		req             *http.Request
		body            []byte
		vars            map[string]string
	)

	BeforeEach(func() {
		policyDB = &fakes.FakePolicyDB{}
		fakeCredentials = &fakes.FakeCredentials{}
		credentialCache = *cache.New(10*time.Minute, -1)
		vars = make(map[string]string)
		resp = httptest.NewRecorder()
		credentialCache.Flush()
	})

	JustBeforeEach(func() {
		logger := lager.NewLogger("auth-test")
		var err error
		authTest, err = auth.New(logger, fakeCredentials, credentialCache, 10*time.Minute)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Basic Auth tests for publish metrics endpoint", func() {

		credentials := &models.Credential{
			Username: "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu",
			Password: "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G",
		}

		Context("a valid request to publish custom metrics comes", func() {
			Context("credentials exists in the cache", func() {
				It("should get the credentials from cache without searching from database and calls next handler", func() {
					credentialCache.Set("an-app-id", credentials, 10*time.Minute)
					req = CreateRequest(body)
					req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
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

			Context("credentials do not exists in the cache but exist in the database", func() {
				It("should: get the credentials from database, add it to the cache and calls next handler", func() {
					req = CreateRequest(body)
					req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
					vars["appid"] = "an-app-id"
					nextCalled := 0
					nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						nextCalled = nextCalled + 1
					})
					fakeCredentials.GetReturns(credentials, nil)

					authTest.AuthenticateHandler(nextFunc)(resp, req, vars)

					Expect(fakeCredentials.GetCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusOK))
					Expect(nextCalled).To(Equal(1))
					//fills the cache
					_, found := credentialCache.Get("an-app-id")
					Expect(found).To(Equal(true))
				})

			})

			Context("when credentials neither exists in the cache nor exist in the database", func() {
				It("should search in both cache & database and returns status code 401", func() {
					req = CreateRequest(body)
					req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
					vars["appid"] = "an-app-id"
					nextCalled := 0
					nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						nextCalled = nextCalled + 1
					})
					fakeCredentials.GetReturns(nil, sql.ErrNoRows)

					authTest.AuthenticateHandler(nextFunc)(resp, req, vars)

					Expect(fakeCredentials.GetCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusUnauthorized))
					errJson := &models.ErrorResponse{}
					err := json.Unmarshal(resp.Body.Bytes(), errJson)
					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Unauthorized",
						Message: "Unauthorized",
					}))
					Expect(nextCalled).To(Equal(0))
				})

			})

			Context("when a stale credentials exists in the cache", func() {
				It("should search in the database and calls next handler", func() {
					req = CreateRequest(body)
					credentialCache.Set("an-app-id", &models.Credential{Username: "some-stale-hashed-username", Password: "some-stale-hashed-password"}, 10*time.Minute)
					fakeCredentials.GetReturns(credentials, nil)
					req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
					vars["appid"] = "an-app-id"
					nextCalled := 0
					nextFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						nextCalled = nextCalled + 1
					})

					authTest.AuthenticateHandler(nextFunc)(resp, req, vars)
					Expect(policyDB.GetCredentialCallCount()).To(Equal(0))
					Expect(fakeCredentials.GetCallCount()).To(Equal(1))
					Expect(resp.Code).To(Equal(http.StatusOK))
					Expect(nextCalled).To(Equal(1))
				})
			})
		})
	})

	Describe("MTLS Auth tests for publish metrics endpoint", func() {
		const validClientCert1 = "../../../../test-certs/validmtls_client-1.crt"
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
				const validClientCert2 = "../../../../test-certs/validmtls_client-2.crt"
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
