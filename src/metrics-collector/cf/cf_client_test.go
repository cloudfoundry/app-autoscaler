package cf_test

import (
	. "metrics-collector/cf"
	"metrics-collector/config"
	"net"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("CfClient", func() {
	Describe("Login", func() {
		var (
			fakeCC *ghttp.Server
			cfc    CfClient
			conf   *config.Config
			err    error
		)

		BeforeEach(func() {
			fakeCC = ghttp.NewServer()
			conf = config.DefaultConfig()
			conf.Cf.Api = fakeCC.URL()
			err = nil
		})

		JustBeforeEach(func() {
			cfc = NewCfClient(&conf.Cf)
			err = cfc.Login()
		})

		AfterEach(func() {
			if fakeCC != nil {
				fakeCC.Close()
			}
		})

		Context("when retrieving endpoints fail", func() {
			Context("when a non-200 status code is returned", func() {
				BeforeEach(func() {
					fakeCC.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/v2/info"),
							ghttp.RespondWith(500, ""),
						),
					)
				})

				It("returns an error", func() {
					Expect(err).To(MatchError(MatchRegexp("Error retrieving cf endpoints: .*")))
				})
			})

			Context("when the request fails", func() {
				BeforeEach(func() {
					fakeCC.Close()
					fakeCC = nil
				})

				It("returns an error", func() {
					Expect(err).To(BeAssignableToTypeOf(&url.Error{}))
					urlErr := err.(*url.Error)
					Expect(urlErr.Err).To(BeAssignableToTypeOf(&net.OpError{}))
				})
			})
		})

		/**
		Context("when retrieving endpoints succeed", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/internal/bulk/apps", "batch_size=2&format=fingerprint&token={}"),
						ghttp.VerifyBasicAuth("the-username", "the-password"),
						ghttp.RespondWith(200, `{
								"token": {"id":"the-token-id"},
								"fingerprints": [
									{
										"process_guid": "process-guid-1",
										"etag": "1234567.890"
									},
									{
										"process_guid": "process-guid-2",
										"etag": "2345678.901"
									}
								]
							}`),
					),
				)
			})

			It("retrieves the endpoints", func() {

			})

			Context("when grant with password", func() {
				BeforeEach(func() {
					conf = fakes.FakeCfConfig
					conf.Api = testApiUrl
					conf.GrantType = GRANT_TYPE_PASSWORD
				})

				It("should not error, and get correct tokens and endpoints", func() {
					Expect(err).To(BeNil())
					Expect(cfc.GetEndpoints().AuthEndpoint).To(Equal(testAuthUrl))
					Expect(cfc.GetEndpoints().DopplerEndpoint).To(Equal(fakes.FAKE_DOPPLER_ENDPOINT))
					Expect(cfc.GetEndpoints().TokenEndpoint).To(Equal(fakes.FAKE_TOKEN_ENDPOINT))
					Expect(cfc.GetTokens().AccessToken).To(Equal(fakes.FAKE_ACCESS_TOKEN))
					Expect(cfc.GetTokens().RefreshToken).To(Equal(fakes.FAKE_REFRESH_TOKEN))
				})
			})

			Context("when grant with client credentials", func() {
				BeforeEach(func() {
					conf = fakes.FakeCfConfig
					conf.Api = testApiUrl
					conf.GrantType = GRANT_TYPE_CLIENT_CREDENTIALS
				})

				It("should not error, and get correct endpoints and tokens", func() {
					Expect(err).To(BeNil())
					Expect(cfc.GetEndpoints().AuthEndpoint).To(Equal(testAuthUrl))
					Expect(cfc.GetEndpoints().DopplerEndpoint).To(Equal(fakes.FAKE_DOPPLER_ENDPOINT))
					Expect(cfc.GetEndpoints().TokenEndpoint).To(Equal(fakes.FAKE_TOKEN_ENDPOINT))
					Expect(cfc.GetTokens().AccessToken).To(Equal(fakes.FAKE_ACCESS_TOKEN))
					Expect(cfc.GetTokens().RefreshToken).To(Equal(fakes.FAKE_REFRESH_TOKEN))
				})
			})

			Context("when API endpoint is not reachable", func() {
				BeforeEach(func() {
					conf = fakes.FakeCfConfig
					conf.Api = "http://www.not-exist-server.com"
				})

				It("should error and return empty endpoints and tokens", func() {
					Expect(err).To(HaveOccurred())
					Expect(cfc.GetEndpoints().AuthEndpoint).To(BeEmpty())
					Expect(cfc.GetEndpoints().DopplerEndpoint).To(BeEmpty())
					Expect(cfc.GetEndpoints().TokenEndpoint).To(BeEmpty())
					Expect(cfc.GetTokens().AccessToken).To(BeEmpty())
					Expect(cfc.GetTokens().RefreshToken).To(BeEmpty())
				})
			})

			Context("when login with not supported grant type", func() {
				BeforeEach(func() {
					conf = fakes.FakeCfConfig
					conf.Api = testApiUrl
					conf.GrantType = "not-exist-grant-type"
				})

				It("should error and return empty oauth token", func() {
					Expect(err).NotTo(BeNil())
					Expect(cfc.GetTokens().AccessToken).To(BeEmpty())
					Expect(cfc.GetTokens().RefreshToken).To(BeEmpty())
				})
			})

			Context("when login with wrong password", func() {
				BeforeEach(func() {
					conf = fakes.FakeCfConfig
					conf.Api = testApiUrl
					conf.GrantType = GRANT_TYPE_PASSWORD
					conf.Pass = "not-exist-password"
				})

				It("should error and return empty tokens", func() {
					Expect(err).NotTo(BeNil())
					Expect(cfc.GetTokens().AccessToken).To(BeEmpty())
					Expect(cfc.GetTokens().RefreshToken).To(BeEmpty())
				})
			})

			Context("when login with wrong user", func() {
				BeforeEach(func() {
					conf = fakes.FakeCfConfig
					conf.Api = testApiUrl
					conf.GrantType = GRANT_TYPE_PASSWORD
					conf.User = "not-exist-user"
				})

				It("should error and return empty oauth token", func() {
					Expect(err).NotTo(BeNil())
					Expect(cfc.GetTokens().AccessToken).To(BeEmpty())
					Expect(cfc.GetTokens().RefreshToken).To(BeEmpty())
				})
			})

			Context("when login with wrong client id", func() {
				BeforeEach(func() {
					conf = fakes.FakeCfConfig
					conf.Api = testApiUrl
					conf.GrantType = GRANT_TYPE_CLIENT_CREDENTIALS
					conf.ClientId = "not-exist-client-id"
				})

				It("should error and return empty oauth token", func() {
					Expect(err).NotTo(BeNil())
					Expect(cfc.GetTokens().AccessToken).To(BeEmpty())
					Expect(cfc.GetTokens().RefreshToken).To(BeEmpty())
				})
			})

			Context("when login with wrong client secret", func() {
				BeforeEach(func() {
					conf = fakes.FakeCfConfig
					conf.Api = testApiUrl
					conf.GrantType = GRANT_TYPE_CLIENT_CREDENTIALS
					conf.ClientId = "not-exist-client-secret"
				})

				It("should error and return empty oauth token", func() {
					Expect(err).NotTo(BeNil())
					Expect(cfc.GetTokens().AccessToken).To(BeEmpty())
					Expect(cfc.GetTokens().RefreshToken).To(BeEmpty())
				})
			})

			**/
	})
})
