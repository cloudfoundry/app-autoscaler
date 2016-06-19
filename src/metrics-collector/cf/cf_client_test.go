package cf_test

import (
	"bytes"
	. "metrics-collector/cf"
	"metrics-collector/config"
	"net"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-golang/lager"
)

var infoBody = []byte(`
{
   "name": "",
   "build": "",
   "support": "http://support.cloudfoundry.com",
   "version": 0,
   "description": "",
   "authorization_endpoint": "test-oauth-endpoint",
   "token_endpoint": "test-token-endpoint",
   "min_cli_version": null,
   "min_recommended_cli_version": null,
   "api_version": "2.48.0",
   "app_ssh_endpoint": "ssh.bosh-lite.com:2222",
   "app_ssh_host_key_fingerprint": "a6:d1:08:0b:b0:cb:9b:5f:c4:ba:44:2a:97:26:19:8a",
   "app_ssh_oauth_client": "ssh-proxy",
   "routing_endpoint": "https://api.bosh-lite.com/routing",
   "logging_endpoint": "wss://loggregator.bosh-lite.com:443",
   "doppler_logging_endpoint": "test-doppler-endpoint",
   "user": "38b2f682-04bf-48af-9e08-0325aa5c4ea9"
}
`)

var authBody = []byte(`
{
	"access_token":"test-access-token",
	"token_type":"bearer",
	"refresh_token":"test-refresh-token",
	"expires_in":12000,
	"scope":"openid cloud_controller.read password.write cloud_controller.write",
	"jti":"a735f90f-0b49-447d-8f9d-ae2fbc1491dd"
}
`)

var _ = Describe("CfClient", func() {
	Describe("Login", func() {
		var (
			fakeCC          *ghttp.Server
			fakeLoginServer *ghttp.Server
			cfc             CfClient
			conf            *config.Config
			err             error
		)

		BeforeEach(func() {
			fakeCC = ghttp.NewServer()
			fakeLoginServer = ghttp.NewServer()
			conf = &config.Config{
				Cf:      config.DefaultCfConfig,
				Logging: config.DefaultLoggingConfig,
				Server:  config.DefaultServerConfig,
			}

			conf.Cf.Api = fakeCC.URL()
			err = nil
		})

		JustBeforeEach(func() {
			cfc = NewCfClient(&conf.Cf, lager.NewLogger("cf"))
			err = cfc.Login()
		})

		AfterEach(func() {
			if fakeCC != nil {
				fakeCC.Close()
			}
			if fakeLoginServer != nil {
				fakeLoginServer.Close()
			}

		})

		Context("when retrieving endpoints fails", func() {
			Context("when the request fails", func() {
				BeforeEach(func() {
					fakeCC.Close()
					fakeCC = nil
				})

				It("should error", func() {
					Expect(err).To(BeAssignableToTypeOf(&url.Error{}))
					urlErr := err.(*url.Error)
					Expect(urlErr.Err).To(BeAssignableToTypeOf(&net.OpError{}))
				})
			})

			Context("when a non-200 status code is returned", func() {
				BeforeEach(func() {
					fakeCC.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", PATH_CF_INFO),
							ghttp.RespondWith(500, ""),
						),
					)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("Error requesting endpoints: .*")))
				})
			})

		})

		Context("when retrieving endpoints succeeds", func() {
			Context("when the auth url is not valid", func() {
				BeforeEach(func() {
					fakeCC.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", PATH_CF_INFO),
							ghttp.RespondWith(200, bytes.Replace(infoBody, []byte("test-oauth-endpoint"), []byte("http://www.not-exist.com"), -1)),
						),
					)
				})

				It("should have the endpoints populated and return error", func() {
					Expect(cfc.GetEndpoints().AuthEndpoint).To(Equal("http://www.not-exist.com"))
					Expect(cfc.GetEndpoints().TokenEndpoint).To(Equal("test-token-endpoint"))
					Expect(cfc.GetEndpoints().DopplerEndpoint).To(Equal("test-doppler-endpoint"))

					Expect(err).To(BeAssignableToTypeOf(&url.Error{}))
					urlErr := err.(*url.Error)
					Expect(urlErr.Err).To(BeAssignableToTypeOf(&net.OpError{}))

				})

			})

			Context("when the auth url is valid", func() {
				BeforeEach(func() {
					fakeCC.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", PATH_CF_INFO),
							ghttp.RespondWith(200, bytes.Replace(infoBody, []byte("test-oauth-endpoint"), []byte(fakeLoginServer.URL()), -1)),
						),
					)
				})

				Context("when login server fails", func() {
					BeforeEach(func() {
						fakeLoginServer.Close()
						fakeLoginServer = nil
					})

					It("should error", func() {
						Expect(err).To(BeAssignableToTypeOf(&url.Error{}))
						urlErr := err.(*url.Error)
						Expect(urlErr.Err).To(BeAssignableToTypeOf(&net.OpError{}))
					})

				})

				Context("when login in server returns non-200 status code", func() {
					BeforeEach(func() {
						fakeLoginServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("POST", PATH_CF_AUTH),
								ghttp.RespondWith(401, ""),
							),
						)
					})

					It("should error", func() {
						Expect(err).To(MatchError(MatchRegexp("Login failed: .*")))
					})

				})

				Context("when login in server returns 200 status code", func() {
					Context("when using password grant", func() {

						BeforeEach(func() {
							conf.Cf.GrantType = config.GRANT_TYPE_PASSWORD
							conf.Cf.User = "test-user"
							conf.Cf.Pass = "test-pass"

							values := url.Values{
								"grant_type": {config.GRANT_TYPE_PASSWORD},
								"username":   {conf.Cf.User},
								"password":   {conf.Cf.Pass},
							}

							fakeLoginServer.AppendHandlers(
								ghttp.CombineHandlers(
									ghttp.VerifyRequest("POST", PATH_CF_AUTH),
									ghttp.VerifyBasicAuth("cf", ""),
									ghttp.VerifyForm(values),
									ghttp.RespondWith(200, authBody),
								),
							)
						})

						It("should not error and return the correct tokens", func() {
							Expect(err).To(BeNil())
							Expect(cfc.GetTokens().AccessToken).To(Equal("test-access-token"))
							Expect(cfc.GetTokens().RefreshToken).To(Equal("test-refresh-token"))
							Expect(cfc.GetTokens().ExpiresIn).To(Equal(int64(12000)))

						})

					})

					Context("when using client_credentials grant", func() {
						BeforeEach(func() {
							conf.Cf.GrantType = config.GRANT_TYPE_CLIENT_CREDENTIALS
							conf.Cf.ClientId = "test-client-id"
							conf.Cf.Secret = "test-client-secret"

							values := url.Values{
								"grant_type":    {config.GRANT_TYPE_CLIENT_CREDENTIALS},
								"client_id":     {conf.Cf.ClientId},
								"client_secret": {conf.Cf.Secret},
							}
							fakeLoginServer.AppendHandlers(
								ghttp.CombineHandlers(
									ghttp.VerifyRequest("POST", PATH_CF_AUTH),
									ghttp.VerifyBasicAuth(conf.Cf.ClientId, conf.Cf.Secret),
									ghttp.VerifyForm(values),
									ghttp.RespondWith(200, authBody),
								),
							)
						})

						It("should not error and return the correct tokens", func() {
							Expect(err).To(BeNil())
							Expect(cfc.GetTokens().AccessToken).To(Equal("test-access-token"))
							Expect(cfc.GetTokens().RefreshToken).To(Equal("test-refresh-token"))
							Expect(cfc.GetTokens().ExpiresIn).To(Equal(int64(12000)))

						})

					})

				})

			})

		})

	})
})
