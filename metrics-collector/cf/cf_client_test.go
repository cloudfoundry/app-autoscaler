package cf_test

import (
	. "github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/cf"
	"github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/cf/fakes"
	"github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CfClient", func() {

	Describe("Login", func() {
		var (
			cfc  CfClient
			conf config.CfConfig
			err  error
		)

		JustBeforeEach(func() {
			cfc = NewCfClient(conf)
			err = cfc.Login()
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

	})

})
