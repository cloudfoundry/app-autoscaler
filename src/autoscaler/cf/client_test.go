package cf_test

import (
	"net"
	"net/http"
	"net/url"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "autoscaler/cf"
)

var _ = Describe("Client", func() {
	var (
		fakeCC          *ghttp.Server
		fakeLoginServer *ghttp.Server
		cfc             CFClient
		conf            *CFConfig
		authToken       string
		tokens          Tokens
		fclock          *fakeclock.FakeClock
		err             error
	)

	BeforeEach(func() {
		fakeCC = ghttp.NewServer()
		fakeLoginServer = ghttp.NewServer()
		conf = &CFConfig{}
		conf.API = fakeCC.URL()
		fclock = fakeclock.NewFakeClock(time.Now())
		err = nil
	})

	AfterEach(func() {
		if fakeCC != nil {
			fakeCC.Close()
		}
		if fakeLoginServer != nil {
			fakeLoginServer.Close()
		}
	})

	Describe("Login", func() {

		JustBeforeEach(func() {
			cfc = NewCFClient(conf, lager.NewLogger("cf"), fclock)
			err = cfc.Login()
		})

		Context("when retrieving endpoints succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", PathCFInfo),
						ghttp.RespondWithJSONEncoded(http.StatusOK, Endpoints{
							AuthEndpoint:    "test-oauth-endpoint",
							TokenEndpoint:   "test-token-endpoint",
							DopplerEndpoint: "test-doppler-endpoint",
						}),
					),
				)
			})

			It("has endpoints", func() {
				Expect(cfc.GetEndpoints().AuthEndpoint).To(Equal("test-oauth-endpoint"))
				Expect(cfc.GetEndpoints().TokenEndpoint).To(Equal("test-token-endpoint"))
				Expect(cfc.GetEndpoints().DopplerEndpoint).To(Equal("test-doppler-endpoint"))
			})
		})

		Context("when retrieving endpoints fails", func() {
			Context("when the Cloud Controller is not running", func() {
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
							ghttp.VerifyRequest("GET", PathCFInfo),
							ghttp.RespondWith(500, ""),
						),
					)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("Error requesting endpoints: .*")))
				})
			})
		})

		Context("when the auth url is valid", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", PathCFInfo),
						ghttp.RespondWithJSONEncoded(http.StatusOK, Endpoints{
							AuthEndpoint:    fakeLoginServer.URL(),
							TokenEndpoint:   "test-token-endpoint",
							DopplerEndpoint: "test-doppler-endpoint",
						}),
					),
				)
			})

			Context("when auth server returns 200 status code", func() {

				BeforeEach(func() {
					conf.ClientID = "test-client-id"
					conf.Secret = "test-client-secret"

					values := url.Values{
						"grant_type":    {GrantTypeClientCredentials},
						"client_id":     {conf.ClientID},
						"client_secret": {conf.Secret},
					}

					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", PathCFAuth),
							ghttp.VerifyBasicAuth(conf.ClientID, conf.Secret),
							ghttp.VerifyForm(values),
							ghttp.RespondWithJSONEncoded(http.StatusOK, Tokens{
								AccessToken: "test-access-token",
								ExpiresIn:   12000,
							}),
						),
					)
				})

				It("returns the correct tokens", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(cfc.GetTokens().AccessToken).To(Equal("test-access-token"))
					Expect(cfc.GetTokens().ExpiresIn).To(Equal(int64(12000)))
				})

			})

			Context("when auth server is not running", func() {
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

			Context("when auth returns a non-200 status code", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", PathCFAuth),
							ghttp.RespondWith(401, ""),
						),
					)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("request token grant failed: .*")))
				})
			})

		})
	})

	Describe("RefreshAuthToken", func() {
		BeforeEach(func() {
			cfc = NewCFClient(conf, lager.NewLogger("cf"), fclock)
		})

		JustBeforeEach(func() {
			authToken, err = cfc.RefreshAuthToken()
		})

		Context("when not logged in", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", PathCFInfo),
						ghttp.RespondWithJSONEncoded(http.StatusOK, Endpoints{
							AuthEndpoint:    fakeLoginServer.URL(),
							TokenEndpoint:   "test-token-endpoint",
							DopplerEndpoint: "test-doppler-endpoint",
						}),
					),
				)
			})

			Context("when auth server returns a 200 status code ", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", PathCFAuth),
							ghttp.RespondWithJSONEncoded(http.StatusOK, Tokens{
								AccessToken: "test-access-token",
								ExpiresIn:   12000,
							}),
						),
					)
				})

				It("returns valid token", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(authToken).To(Equal("Bearer test-access-token"))
					Expect(cfc.GetTokens().AccessToken).To(Equal("test-access-token"))
					Expect(cfc.GetTokens().ExpiresIn).To(Equal(int64(12000)))
				})

			})

			Context("when auth server returns a non-200 status code", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", PathCFAuth),
							ghttp.RespondWith(401, ""),
						),
					)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("request token grant failed: .*")))
				})
			})

		})

		Context("when already logged in", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", PathCFInfo),
						ghttp.RespondWithJSONEncoded(http.StatusOK, Endpoints{
							AuthEndpoint:    fakeLoginServer.URL(),
							TokenEndpoint:   "test-token-endpoint",
							DopplerEndpoint: "test-doppler-endpoint",
						}),
					),
				)
				fakeLoginServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", PathCFAuth),
						ghttp.RespondWithJSONEncoded(http.StatusOK, Tokens{
							AccessToken: "test-access-token",
							ExpiresIn:   12000,
						}),
					),
				)
				cfc.Login()
			})

			Context("when auth fails", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", PathCFAuth),
							ghttp.VerifyForm(url.Values{
								"grant_type":    {GrantTypeClientCredentials},
								"client_id":     {conf.ClientID},
								"client_secret": {conf.Secret},
							}),
							ghttp.RespondWith(401, ""),
						),
					)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("request token grant failed: .*")))
				})
			})

			Context("when auth succeeds", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", PathCFAuth),
							ghttp.VerifyForm(url.Values{
								"grant_type":    {GrantTypeClientCredentials},
								"client_id":     {conf.ClientID},
								"client_secret": {conf.Secret},
							}),
							ghttp.RespondWithJSONEncoded(http.StatusOK, Tokens{
								AccessToken: "test-access-token",
								ExpiresIn:   12000,
							}),
						),
					)

				})
				It("returns valid tokens", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(authToken).To(Equal("Bearer test-access-token"))
					Expect(cfc.GetTokens().AccessToken).To(Equal("test-access-token"))
					Expect(cfc.GetTokens().ExpiresIn).To(Equal(int64(12000)))
				})

			})

		})
	})

	Describe("GetTokens", func() {
		JustBeforeEach(func() {
			tokens = cfc.GetTokens()
		})

		BeforeEach(func() {
			cfc = NewCFClient(conf, lager.NewLogger("cf"), fclock)
			fakeCC.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", PathCFInfo),
					ghttp.RespondWithJSONEncoded(http.StatusOK, Endpoints{
						AuthEndpoint:    fakeLoginServer.URL(),
						TokenEndpoint:   "test-token-endpoint",
						DopplerEndpoint: "test-doppler-endpoint",
					}),
				),
			)
			fakeLoginServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", PathCFAuth),
					ghttp.RespondWithJSONEncoded(http.StatusOK, Tokens{
						AccessToken: "test-access-token",
						ExpiresIn:   12000,
					}),
				),
			)
			cfc.Login()
		})

		Context("when the token is not going to be expired", func() {
			BeforeEach(func() {
				fclock.Increment(12000*time.Second - TimeToRefreshBeforeTokenExpire)
			})
			It("does not refresh tokens", func() {
				Expect(tokens.AccessToken).To(Equal("test-access-token"))
				Expect(tokens.ExpiresIn).To(Equal(int64(12000)))
			})

		})

		Context("when the token is going to be expired", func() {
			Context("when refresh succeeds", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", PathCFAuth),
							ghttp.VerifyForm(url.Values{
								"grant_type":    {GrantTypeClientCredentials},
								"client_id":     {conf.ClientID},
								"client_secret": {conf.Secret},
							}),
							ghttp.RespondWithJSONEncoded(http.StatusOK, Tokens{
								AccessToken: "test-access-token-refreshed",
								ExpiresIn:   24000,
							}),
						),
					)
					fclock.Increment(12001*time.Second - TimeToRefreshBeforeTokenExpire)
				})

				It("refreshes tokens", func() {
					Expect(tokens.AccessToken).To(Equal("test-access-token-refreshed"))
					Expect(tokens.ExpiresIn).To(Equal(int64(24000)))
				})

			})

			Context("when refresh fails", func() {
				BeforeEach(func() {
					fakeCC.RouteToHandler("GET", "/v2/info", ghttp.RespondWith(200, ""))
					fakeLoginServer.RouteToHandler("POST", "/oauth/token", ghttp.RespondWith(401, ""))
					fclock.Increment(12001*time.Second - TimeToRefreshBeforeTokenExpire)
				})

				It("returns existing tokens", func() {
					Expect(tokens.AccessToken).To(Equal("test-access-token"))
					Expect(tokens.ExpiresIn).To(Equal(int64(12000)))
				})

			})

		})

	})
})
