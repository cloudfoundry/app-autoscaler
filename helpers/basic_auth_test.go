package helpers_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"net/http"
	"net/http/httptest"
)

var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

var _ = Describe("BasicAuthenticationMiddleware", func() {
	var (
		server   *httptest.Server
		ba       models.BasicAuth
		resp     *http.Response
		username string
		password string
		logger   lager.Logger
	)

	BeforeEach(func() {
		logger = lager.NewLogger("helper-test")
	})

	AfterEach(func() {
		server.Close()
	})

	JustBeforeEach(func() {
		bam, err := helpers.CreateBasicAuthMiddleware(logger, ba)
		Expect(err).NotTo(HaveOccurred())

		server = httptest.NewServer(bam.BasicAuthenticationMiddleware(handler))

		req, err := http.NewRequest("GET", server.URL+"/some-protected-endpoint", nil)
		req.SetBasicAuth(username, password)
		Expect(err).NotTo(HaveOccurred())

		resp, err = http.DefaultClient.Do(req)
		Expect(err).NotTo(HaveOccurred())

		defer resp.Body.Close()
	})

	When("basic auth is enabled", func() {
		BeforeEach(func() {
			ba = models.BasicAuth{
				Username: "username",
				Password: "password",
			}
		})

		When("credentials are correct", func() {
			BeforeEach(func() {
				username = ba.Username
				password = ba.Password
			})

			It("should return 200", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		When("credentials are incorrect", func() {
			BeforeEach(func() {
				username = "wrong-username"
				password = "wrong-password"
			})

			It("should return 401", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})
	})
})
