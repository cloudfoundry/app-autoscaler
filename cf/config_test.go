package cf_test

import (
	. "autoscaler/cf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	var (
		conf *CfConfig
		err  error
	)

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &CfConfig{}
			conf.Api = "http://api.example.com"
			conf.GrantType = GrantTypePassword
			conf.Username = "admin"
			conf.ClientId = "admin"
			conf.SkipSSLValidation = false
		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when api is not set", func() {
			BeforeEach(func() {
				conf.Api = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: cf api is empty"))
			})
		})
		Context("when SkipSSLValidation is not set", func() {
			It("should set SkipSSLValidation to default false value", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.SkipSSLValidation).To(Equal(false))
			})
		})

		Context("when SkipSSLValidation is set", func() {
			BeforeEach(func() {
				conf.SkipSSLValidation = true
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.SkipSSLValidation).To(Equal(true))
			})
		})

		Context("when api is not a url", func() {
			BeforeEach(func() {
				conf.Api = "http://a.com%"
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: cf api is not a valid url"))
			})
		})

		Context("when api scheme is empty", func() {
			BeforeEach(func() {
				conf.Api = "a.com"
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: cf api scheme is empty"))
			})
		})

		Context("when api has invalid scheme", func() {
			BeforeEach(func() {
				conf.Api = "badscheme://a.com"
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: cf api scheme is invalid"))
			})
		})

		Context("when api is valid but ends with a '/'", func() {
			BeforeEach(func() {
				conf.Api = "https://a.com/"
			})

			It("should not error and remove the '/'", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Api).To(Equal("https://a.com"))
			})
		})

		Context("when grant type is not supported", func() {
			BeforeEach(func() {
				conf.GrantType = "not-supported"
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: unsupported grant type*")))
			})
		})

		Context("when grant type password", func() {
			BeforeEach(func() {
				conf.GrantType = GrantTypePassword
			})

			Context("when user name is set", func() {
				BeforeEach(func() {
					conf.ClientId = ""
					conf.Username = "admin"
				})
				It("is valid", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when the user name is empty", func() {
				BeforeEach(func() {
					conf.Username = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: user name is empty"))
				})
			})
		})

		Context("when grant type client_credential", func() {
			BeforeEach(func() {
				conf.GrantType = GrantTypeClientCredentials
			})

			Context("when client id is set", func() {
				BeforeEach(func() {
					conf.ClientId = "admin"
					conf.Username = ""
				})
				It("is valid", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("the client id is empty", func() {
				BeforeEach(func() {
					conf.ClientId = ""
				})

				It("returns error", func() {
					Expect(err).To(MatchError("Configuration error: client id is empty"))
				})
			})
		})

	})
})
