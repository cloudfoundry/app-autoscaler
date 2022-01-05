package cf_test

import (
	"net/http"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
)

var _ = Describe("UAA Client", func() {
	var (
		fakeUAAServer *ghttp.Server
		uaaClient     UaaClient
		conf          *CFConfig
		authToken     string
		err           error
	)

	BeforeEach(func() {
		fakeUAAServer = ghttp.NewServer()
		conf = &CFConfig{}
		err = nil
	})

	AfterEach(func() {
		if fakeUAAServer != nil {
			fakeUAAServer.Close()
		}
	})

	Describe("RefreshAuthToken", func() {
		BeforeEach(func() {
			uaaClient = NewUaaClient(conf, lager.NewLogger("uaa-client"), fakeUAAServer.URL())
		})

		JustBeforeEach(func() {
			authToken, err = uaaClient.RefreshAuthToken()
		})

		Context("GetAuthToken", func() {

			Context("when uaa server returns a 200 status code for getting Oauth token", func() {
				BeforeEach(func() {
					jsonData := []byte(`
						{
							"access_token":"test-access-token",
							"token_type":"Bearer",
							"expires_in":599,
							"scope":"cloud_controller.write doppler.firehose",
							"jti":"28edda5c-4e37-4a63-9ba3-b32f48530a51"
						}
					`)
					fakeUAAServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", PathCFAuth),
							ghttp.RespondWith(http.StatusOK, jsonData),
						),
					)
				})

				It("returns valid token", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(authToken).To(Equal("Bearer test-access-token"))
				})

			})

			Context("when uaa server returns a 401 status code", func() {
				BeforeEach(func() {
					fakeUAAServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", PathCFAuth),
							ghttp.RespondWith(401, ""),
						),
					)
				})

				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(MatchRegexp("Received a status code 401 Unauthorized")))
				})
			})

		})
	})

})
