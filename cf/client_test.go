package cf_test

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
)

var _ = Describe("Client", func() {

	BeforeEach(func() { fakeCC.Add().Info(fakeLoginServer.URL()) })
	Describe("Login", func() {
		var tokens cf.Tokens
		JustBeforeEach(func() { err = cfc.Login() })

		Context("when the token url is valid", func() {
			Context("when token server returns 200 status code", func() {

				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", cf.PathCFAuth),
							ghttp.VerifyBasicAuth(conf.ClientID, conf.Secret),
							ghttp.VerifyForm(url.Values{
								"grant_type":    {cf.GrantTypeClientCredentials},
								"client_id":     {conf.ClientID},
								"client_secret": {conf.Secret},
							}),
							ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
								AccessToken: "test-access-token",
								ExpiresIn:   12000,
							}),
						),
					)
				})

				It("returns the correct tokens", func() {
					Expect(err).ToNot(HaveOccurred())
					tokens, err = cfc.GetTokens()
					Expect(err).ToNot(HaveOccurred())
					Expect(tokens.AccessToken).To(Equal("test-access-token"))
					Expect(tokens.ExpiresIn).To(Equal(int64(12000)))
				})

			})

			Context("when token server is not running", func() {
				BeforeEach(func() {
					fakeLoginServer.Close()
					fakeLoginServer = nil
				})

				It("should error", func() {
					IsUrlNetOpError(err)
				})
			})

			Context("when token returns a non-200 status code", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", cf.PathCFAuth),
							ghttp.RespondWith(401, ""),
						),
					)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("request client credential grant failed: .*")))
				})
			})

		})
	})

	Context("Thread safety", func() {
		Context("when there are several threads calling refresh at the same time", func() {
			//var tokens Tokens
			BeforeEach(func() {
				fakeCC.Add().Info(fakeLoginServer.URL())
				fakeLoginServer.RouteToHandler(http.MethodPost, cf.PathCFAuth,
					ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
						AccessToken: "test-access-token",
						ExpiresIn:   12000,
					}),
				)
			})

			It("returns the correct tokens", func() {
				Expect(err).ToNot(HaveOccurred())
				wg := sync.WaitGroup{}
				mu := sync.RWMutex{}
				mu.Lock()
				for i := 0; i < 100; i++ {
					wg.Add(1)
					go func() {
						mu.RLock()
						defer mu.RUnlock()
						_, innerErr := cfc.RefreshAuthToken()
						Expect(innerErr).ToNot(HaveOccurred())
						wg.Done()
					}()
				}
				mu.Unlock()
				wg.Wait()
				Expect(fakeLoginServer.Count().Requests(cf.PathCFAuth)).To(Equal(1))
			})

		})
	})

	Describe("RefreshAuthToken", func() {
		var authToken cf.Tokens

		JustBeforeEach(func() {
			authToken, err = cfc.RefreshAuthToken()
		})

		Context("when not logged in", func() {
			var tokens cf.Tokens

			BeforeEach(func() {
				fakeCC.Add().Info(fakeLoginServer.URL())
			})

			Context("when token server returns a 200 status code ", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", cf.PathCFAuth),
							ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
								AccessToken: "test-access-token",
								ExpiresIn:   12000,
							}),
						),
					)
				})

				It("returns valid token", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(authToken).To(Equal(cf.Tokens{
						AccessToken: "test-access-token",
						ExpiresIn:   12000,
					}))
					tokens, err = cfc.GetTokens()
					Expect(err).ToNot(HaveOccurred())
					Expect(tokens.AccessToken).To(Equal("test-access-token"))
					Expect(tokens.ExpiresIn).To(Equal(int64(12000)))
				})

			})

			Context("when token server returns a non-200 status code", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", cf.PathCFAuth),
							ghttp.RespondWith(401, ""),
						),
					)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("request client credential grant failed: .*")))
				})
			})

		})

		Context("when already logged in", func() {
			BeforeEach(func() {
				fakeCC.Add().Info(fakeLoginServer.URL())
				fakeLoginServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", cf.PathCFAuth),
						ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
							AccessToken: "test-access-token",
							ExpiresIn:   12000,
						}),
					),
				)
				err = cfc.Login()
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when auth fails", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", cf.PathCFAuth),
							ghttp.VerifyForm(url.Values{
								"grant_type":    {cf.GrantTypeClientCredentials},
								"client_id":     {conf.ClientID},
								"client_secret": {conf.Secret},
							}),
							ghttp.RespondWith(401, ""),
						),
					)
					cfc.InvalidateToken()
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("request client credential grant failed: .*")))
				})
			})

			Context("when auth succeeds", func() {
				BeforeEach(func() {
					fakeLoginServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", cf.PathCFAuth),
							ghttp.VerifyForm(url.Values{
								"grant_type":    {cf.GrantTypeClientCredentials},
								"client_id":     {conf.ClientID},
								"client_secret": {conf.Secret},
							}),
							ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
								AccessToken: "test-access-token",
								ExpiresIn:   12000,
							}),
						),
					)
				})
				It("returns valid tokens", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(authToken).To(Equal(cf.Tokens{
						AccessToken: "test-access-token",
						ExpiresIn:   12000,
					}))
					tokens, err := cfc.GetTokens()
					Expect(err).ToNot(HaveOccurred())
					Expect(tokens.AccessToken).To(Equal("test-access-token"))
					Expect(tokens.ExpiresIn).To(Equal(int64(12000)))
				})
			})
		})
	})

	Describe("GetTokens", func() {
		var tokens cf.Tokens
		JustBeforeEach(func() {
			tokens, err = cfc.GetTokens()
		})

		BeforeEach(func() {
			cfc = cf.NewCFClient(conf, lager.NewLogger("cf"), fclock)
			fakeCC.Add().Info(fakeLoginServer.URL())
			fakeLoginServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", cf.PathCFAuth),
					ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
						AccessToken: "test-access-token",
						ExpiresIn:   12000,
					}),
				),
			)
			err = cfc.Login()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the token is not going to be expired", func() {
			BeforeEach(func() {
				fclock.Increment(12000*time.Second - cf.TimeToRefreshBeforeTokenExpire)
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
							ghttp.VerifyRequest("POST", cf.PathCFAuth),
							ghttp.VerifyForm(url.Values{
								"grant_type":    {cf.GrantTypeClientCredentials},
								"client_id":     {conf.ClientID},
								"client_secret": {conf.Secret},
							}),
							ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
								AccessToken: "test-access-token-refreshed",
								ExpiresIn:   24000,
							}),
						),
					)
					fclock.Increment(12001*time.Second - cf.TimeToRefreshBeforeTokenExpire)
				})

				It("refreshes tokens", func() {
					Expect(tokens.AccessToken).To(Equal("test-access-token-refreshed"))
					Expect(tokens.ExpiresIn).To(Equal(int64(24000)))
				})

			})

			Context("when refresh fails", func() {
				BeforeEach(func() {
					fakeCC.Add().Info(fakeLoginServer.URL())
					fakeLoginServer.RouteToHandler("POST", "/oauth/token", ghttp.RespondWith(401, ""))
					fclock.Increment(12001*time.Second - cf.TimeToRefreshBeforeTokenExpire)
				})

				It("returns existing tokens", func() {
					Expect(err).To(HaveOccurred())
					Expect(tokens.AccessToken).To(Equal("test-access-token"))
					Expect(tokens.ExpiresIn).To(Equal(int64(12000)))
				})

			})

		})

	})

	Describe("IsTokenAuthorized", func() {
		BeforeEach(func() {
			cfc = cf.NewCFClient(conf, lager.NewLogger("cf"), fclock)
			fakeCC.Add().Info(fakeLoginServer.URL())
			fakeLoginServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", cf.PathCFAuth),
					ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
						AccessToken: "test-access-token",
						ExpiresIn:   12000,
					}),
				),
			)
			err = cfc.Login()
			Expect(err).NotTo(HaveOccurred())
		})

		const (
			invalidToken  = "INVALID_TOKEN"
			validClientID = "VALID_CLIENT_ID"
			wrongClientID = "WRONG_CLIENT_ID"
		)
		var (
			token        string
			isTokenValid bool
		)
		Context("when the token is invalid", func() {

			BeforeEach(func() {
				fakeLoginServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", cf.PathIntrospectToken),
						ghttp.RespondWithJSONEncoded(http.StatusOK, cf.IntrospectionResponse{Active: false}),
					),
				)
			})
			JustBeforeEach(func() {
				isTokenValid, err = cfc.IsTokenAuthorized(token, validClientID)
			})
			BeforeEach(func() {
				token = invalidToken
			})
			It("returns false", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(isTokenValid).To(BeFalse())
			})
		})

		Context("when the token is valid, but for the wrong client id", func() {

			BeforeEach(func() {
				fakeLoginServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", cf.PathIntrospectToken),
						ghttp.RespondWithJSONEncoded(http.StatusOK, cf.IntrospectionResponse{Active: true, ClientId: wrongClientID, Email: validClientID}),
					),
				)
			})

			JustBeforeEach(func() {
				isTokenValid, err = cfc.IsTokenAuthorized(token, validClientID)
			})
			BeforeEach(func() {
				token = invalidToken
			})
			It("returns false", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(isTokenValid).To(BeFalse())
			})
		})

		Context("when the token is valid, and for the right client id", func() {

			BeforeEach(func() {
				fakeLoginServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", cf.PathIntrospectToken),
						ghttp.RespondWithJSONEncoded(http.StatusOK, cf.IntrospectionResponse{Active: true, ClientId: validClientID, Email: "john@doe"}),
					),
				)
			})

			JustBeforeEach(func() {
				isTokenValid, err = cfc.IsTokenAuthorized(token, validClientID)
			})
			BeforeEach(func() {
				token = invalidToken
			})
			It("returns false", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(isTokenValid).To(BeTrue())
			})
		})

	})
})
