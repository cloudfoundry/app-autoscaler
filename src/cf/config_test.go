package cf_test

import (
	. "cf"

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

		Context("when api is not a url", func() {
			BeforeEach(func() {
				conf.Api = "a.com%"
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: cf api is invalid"))
			})
		})

		Context("when api valid but not regulated", func() {
			BeforeEach(func() {
				conf.Api = "a.com/"
			})

			It("should not error and regulate the api", func() {
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
